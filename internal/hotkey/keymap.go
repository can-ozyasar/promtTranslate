package hotkey

// Key represents a keyboard key code (Linux evdev KEY_* constants).
type Key uint16

// Modifier bitmask constants.
type Modifier uint8

const (
	ModAlt   Modifier = 1 << 0
	ModShift Modifier = 1 << 1
	ModCtrl  Modifier = 1 << 2
	ModMeta  Modifier = 1 << 3
)

// Combo is a modifier+key combination that triggers an action.
type Combo struct {
	Mods Modifier
	Key  Key
	Name string
}

// evdev KEY_* constants we care about.
const (
	KeySpace Key = 57
	KeyC     Key = 46
	KeyLeftAlt   Key = 56
	KeyRightAlt  Key = 100
	KeyLeftShift Key = 42
	KeyRightShift Key = 54
	KeyLeftCtrl  Key = 29
	KeyRightCtrl Key = 97
)

// DefaultCombos returns the standard hotkey combos.
// Alt+Space → WriteMode, Alt+Shift+Space → ReadMode, Alt+Shift+C → Reload
func DefaultCombos() []Combo {
	return []Combo{
		{Mods: ModAlt, Key: KeySpace, Name: "write_mode"},
		{Mods: ModAlt | ModShift, Key: KeySpace, Name: "read_mode"},
		{Mods: ModAlt | ModShift, Key: KeyC, Name: "reload"},
	}
}

// modifierKeys lists all key codes that are treated as modifiers.
var modifierKeys = map[Key]Modifier{
	KeyLeftAlt:    ModAlt,
	KeyRightAlt:   ModAlt,
	KeyLeftShift:  ModShift,
	KeyRightShift: ModShift,
	KeyLeftCtrl:   ModCtrl,
	KeyRightCtrl:  ModCtrl,
}

// IsModifier returns true if the key is a modifier key.
func IsModifier(k Key) bool {
	_, ok := modifierKeys[k]
	return ok
}

// ModifierOf returns the Modifier bit for a modifier key.
func ModifierOf(k Key) Modifier {
	return modifierKeys[k]
}
