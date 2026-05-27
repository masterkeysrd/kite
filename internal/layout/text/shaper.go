package text

import "container/list"

const defaultBudgetBytes = 256 * 1024 // 256 KiB

// cacheKey uniquely identifies a shaping request.
type cacheKey struct {
	text         string
	fontSubsetID int
}

// cacheEntry holds the cached clusters for one (text, fontSubsetID) pair.
type cacheEntry struct {
	key      cacheKey
	clusters []Cluster
	cost     int // estimated byte cost toward the budget
}

// Shaper shapes text into grapheme clusters and caches results by
// (text, fontSubsetID) under a configurable byte-budget LRU policy.
// fontSubsetID is always 0 in kitex v2 (single font subset).
//
// Shaper is not safe for concurrent use.
type Shaper struct {
	budgetBytes int
	usedBytes   int
	index       map[cacheKey]*list.Element // key → list element
	lru         *list.List                 // front = MRU, back = LRU
}

// NewShaper creates a Shaper with the given byte budget. Pass 0 to use the
// default 256 KiB budget.
func NewShaper(budgetBytes int) *Shaper {
	if budgetBytes <= 0 {
		budgetBytes = defaultBudgetBytes
	}
	return &Shaper{
		budgetBytes: budgetBytes,
		index:       make(map[cacheKey]*list.Element),
		lru:         list.New(),
	}
}

// Shape returns the grapheme clusters for text, using fontSubsetID 0.
// The same []Cluster pointer is returned on every cache hit.
func (s *Shaper) Shape(text string) []Cluster {
	return s.shapeWith(text, 0)
}

// MeasureRun returns the total display cell width of text, reusing the
// shaping cache.
func (s *Shaper) MeasureRun(text string) int {
	total := 0
	for _, c := range s.Shape(text) {
		total += c.CellWidth
	}
	return total
}

func (s *Shaper) shapeWith(text string, fontSubsetID int) []Cluster {
	key := cacheKey{text: text, fontSubsetID: fontSubsetID}
	if elem, ok := s.index[key]; ok {
		s.lru.MoveToFront(elem)
		return elem.Value.(*cacheEntry).clusters
	}

	clusters := Shape(text)
	cost := entryCost(text, clusters)

	e := &cacheEntry{key: key, clusters: clusters, cost: cost}
	elem := s.lru.PushFront(e)
	s.index[key] = elem
	s.usedBytes += cost

	// Evict LRU entries until within budget, but always keep the entry we
	// just inserted (Len > 1 guard).
	for s.usedBytes > s.budgetBytes && s.lru.Len() > 1 {
		back := s.lru.Back()
		if back == nil {
			break
		}
		evicted := back.Value.(*cacheEntry)
		s.usedBytes -= evicted.cost
		delete(s.index, evicted.key)
		s.lru.Remove(back)
	}

	return clusters
}

// entryCost estimates the byte budget consumed by a single cache entry.
//
//   - Key: len(text) bytes for the string content.
//   - Value: each Cluster is 3 fields; on 64-bit systems a Cluster struct is
//     approximately 40 bytes ([]byte header = 24 B, int = 8 B, BreakClass
//     (int) = 8 B).
func entryCost(text string, clusters []Cluster) int {
	return len(text) + len(clusters)*40
}
