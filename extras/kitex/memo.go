package kitex

import (
	"reflect"

	"github.com/masterkeysrd/kite/style"
)

// --- Memoization Helpers ------------------------------------------------------

// memoComplexityThreshold is the minimum subtree node count required for
// ComponentNode to activate automatic memoization. Components whose rendered
// tree is cheap to re-render (score ≤ threshold) skip the reflection overhead.
const memoComplexityThreshold = 5

// computeComplexity returns the complexity of a Node, dispatching through the
// nodeInternal interface when available. Falls back to 1 for ComponentNodes
// (which wrap their rendered complexity in complexityScore).
func computeComplexity(n Node) int {
	if n == nil {
		return 0
	}
	if ni, ok := n.(nodeInternal); ok {
		return ni.complexity()
	}
	return 1
}

// buildElementInfo computes the score and provider-presence for an elementNode
// from its children slice and is called by every element factory.
func buildElementInfo(children []Node) (int, bool, bool) {
	s := 1
	hasP := false
	hasDirectP := false
	for _, c := range children {
		if c != nil {
			s += computeComplexity(c)
			if ni, ok := c.(nodeInternal); ok {
				isP := ni.isProvider()
				if isP {
					hasDirectP = true
					hasP = true
				} else if !hasP && ni.containsProvider() {
					hasP = true
				}
			}
		}
	}
	return s, hasP, hasDirectP
}

// EquatableProps allows prop structures to bypass reflection-based comparisons
// by implementing a custom, direct equality method.
type EquatableProps interface {
	Equal(other any) bool
}

// deepEqualProps performs a depth-limited recursive equality check between two
// arbitrary prop values using reflection. It returns true only when the values
// are provably equal at every level up to maxDepth.
//
// Special cases:
//   - reflect.Func values are compared via the existing funcEquals helper so
//     that stable closures with the same pointer compare as equal.
//   - When depth == 0 the function conservatively returns false to avoid
//     unbounded recursion on large nested structures.
func deepEqualProps(oldProps, newProps any, maxDepth int) bool {
	if maxDepth <= 0 {
		return false
	}
	if oldProps == nil && newProps == nil {
		return true
	}
	if oldProps == nil || newProps == nil {
		return false
	}

	if eq, ok := oldProps.(EquatableProps); ok {
		return eq.Equal(newProps)
	}

	ov := reflect.ValueOf(oldProps)
	nv := reflect.ValueOf(newProps)

	// Dereference pointers once.
	if ov.Kind() == reflect.Pointer {
		if ov.IsNil() {
			return nv.Kind() == reflect.Pointer && nv.IsNil()
		}
		if nv.Kind() != reflect.Pointer || nv.IsNil() {
			return false
		}
		ov = ov.Elem()
		nv = nv.Elem()
	}

	if ov.Type() != nv.Type() {
		return false
	}

	return deepEqualValues(ov, nv, maxDepth)
}

// deepEqualValues is the reflection-recursive core of deepEqualProps.
// depth tracks the maximum number of nested *composite* levels (struct-in-struct,
// slice-of-structs, etc.) that are recursed into. Scalar leaf comparisons
// (int, string, bool, float, etc.) never consume depth and always succeed or
// fail immediately without further recursion.
func deepEqualValues(ov, nv reflect.Value, depth int) bool {
	if ov.CanInterface() {
		if s, ok := reflect.TypeAssert[style.Style](ov); ok {
			if other, ok := reflect.TypeAssert[style.Style](nv); ok {
				return s.Equal(other)
			}
		}
	}
	switch ov.Kind() {
	case reflect.Func:
		// Use the existing funcEquals helper which compares function pointers.
		return funcEquals(ov.Interface(), nv.Interface())

	case reflect.Struct:
		// Entering a struct costs one depth level.
		if depth <= 0 {
			return false
		}
		for i := range ov.NumField() {
			of := ov.Field(i)
			nf := nv.Field(i)
			// Skip unexported fields — reflection cannot read them.
			if !of.CanInterface() {
				continue
			}
			if !deepEqualValues(of, nf, depth-1) {
				return false
			}
		}
		return true

	case reflect.Slice:
		if depth <= 0 {
			return false
		}
		if ov.IsNil() != nv.IsNil() {
			return false
		}
		if ov.Len() != nv.Len() {
			return false
		}
		for i := range ov.Len() {
			if !deepEqualValues(ov.Index(i), nv.Index(i), depth-1) {
				return false
			}
		}
		return true

	case reflect.Array:
		if depth <= 0 {
			return false
		}
		for i := range ov.Len() {
			if !deepEqualValues(ov.Index(i), nv.Index(i), depth-1) {
				return false
			}
		}
		return true

	case reflect.Map:
		if depth <= 0 {
			return false
		}
		if ov.IsNil() != nv.IsNil() {
			return false
		}
		if ov.Len() != nv.Len() {
			return false
		}
		for _, k := range ov.MapKeys() {
			nvVal := nv.MapIndex(k)
			if !nvVal.IsValid() {
				return false
			}
			if !deepEqualValues(ov.MapIndex(k), nvVal, depth-1) {
				return false
			}
		}
		return true

	case reflect.Pointer, reflect.Interface:
		if depth <= 0 {
			return false
		}
		if ov.IsNil() {
			return nv.IsNil()
		}
		if nv.IsNil() {
			return false
		}
		return deepEqualValues(ov.Elem(), nv.Elem(), depth-1)

	default:
		// Scalar leaf (bool, int*, uint*, float*, complex*, string, chan, etc.).
		// Scalars compare directly — they do NOT consume depth because they
		// cannot cause unbounded recursion.
		if ov.CanInterface() {
			return ov.Interface() == nv.Interface()
		}
		return false
	}
}
