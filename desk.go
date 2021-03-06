package desktop // import "fyne.io/desktop"

// Desktop defines an embedded or full desktop environment that we can run.
type Desktop interface {
	Run()
	RunApp(AppData) error
	Settings() DeskSettings
	ContentSizePixels(screen *Screen) (uint32, uint32)
	Screens() ScreenList

	IconProvider() ApplicationProvider
	WindowManager() WindowManager
	Modules() []Module
}

var instance Desktop

// Instance returns the current desktop environment and provides access to injected functionality.
func Instance() Desktop {
	return instance
}

// SetInstance is an internal call :( TODO
func SetInstance(desk Desktop) {
	instance = desk
}
