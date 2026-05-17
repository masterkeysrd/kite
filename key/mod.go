package key

type Mod uint16

const (
	ModShift Mod = 1 << iota
	ModAlt
	ModCtrl
	ModMeta

	// Kitty Protocol modifiers (add 4 to the above)
	ModHyper
	ModSuper

	// Lock states
	ModCapsLock
	ModNumLock
	ModScrollLock
)
