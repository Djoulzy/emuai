package emulator

type KeyAction uint8

const (
	KeyActionPress KeyAction = iota + 1
	KeyActionRepeat
	KeyActionRelease
)

type KeyCode uint16

const (
	KeyCodeUnknown KeyCode = iota
	KeyCodeEnter
	KeyCodeEscape
	KeyCodeBackspace
	KeyCodeDelete
	KeyCodeTab
	KeyCodeSpace
	KeyCodeLeft
	KeyCodeRight
	KeyCodeUp
	KeyCodeDown
	KeyCodeHome
	KeyCodeEnd
)

type KeyModifiers struct {
	Shift   bool
	Control bool
	Alt     bool
	Super   bool
}

type KeyEvent struct {
	Code      KeyCode
	Rune      rune
	Action    KeyAction
	Modifiers KeyModifiers
}

type KeyEventSink interface {
	HandleKeyEvent(event KeyEvent)
}
