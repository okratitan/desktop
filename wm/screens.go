// +build linux

package wm // import "fyne.io/desktop/wm"

import (
	"math"

	"fyne.io/fyne"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil/xwindow"

	"fyne.io/desktop"
)

type x11ScreensProvider struct {
	screens []*desktop.Screen
	active  *desktop.Screen
	primary *desktop.Screen
	single  bool
	x       *x11WM
	root    xproto.Window
}

// NewX11ScreensProvider returns a screen provider for use in x11 desktop mode
func NewX11ScreensProvider(mgr desktop.WindowManager) desktop.ScreenList {
	screensProvider := &x11ScreensProvider{}
	screensProvider.x = mgr.(*x11WM)
	err := randr.Init(screensProvider.x.x.Conn())
	if err != nil {
		fyne.LogError("Could not initialize randr", err)
		screensProvider.setupSingleScreen()
		return screensProvider
	}
	screensProvider.root = xproto.Setup(screensProvider.x.x.Conn()).DefaultScreen(screensProvider.x.x.Conn()).Root
	randr.SelectInput(screensProvider.x.x.Conn(), screensProvider.root, randr.NotifyMaskScreenChange)
	screensProvider.setupScreens()

	return screensProvider
}

func (xsp *x11ScreensProvider) RefreshScreens() {
	xsp.screens = nil
	xsp.active = nil
	xsp.primary = nil
	if xsp.single {
		xsp.setupSingleScreen()
		return
	}
	xsp.setupScreens()
}

func (xsp *x11ScreensProvider) Screens() []*desktop.Screen {
	return xsp.screens
}

func (xsp *x11ScreensProvider) Active() *desktop.Screen {
	return xsp.active
}

func (xsp *x11ScreensProvider) Primary() *desktop.Screen {
	return xsp.primary
}

func (xsp *x11ScreensProvider) ScreenForWindow(win desktop.Window) *desktop.Screen {
	if len(xsp.screens) <= 1 {
		return xsp.screens[0]
	}
	fr := win.(*client).frame
	if fr == nil {
		return xsp.Primary()
	}
	return xsp.ScreenForGeometry(int(fr.x), int(fr.y), int(fr.width), int(fr.height))
}

func (xsp *x11ScreensProvider) ScreenForGeometry(x int, y int, width int, height int) *desktop.Screen {
	if len(xsp.screens) <= 1 {
		return xsp.screens[0]
	}
	for i := 0; i < len(xsp.screens); i++ {
		xx, yy, ww, hh := xsp.screens[i].X, xsp.screens[i].Y,
			xsp.screens[i].Width, xsp.screens[i].Height
		middleW := width / 2
		middleH := height / 2
		middleW += x
		middleH += y
		if middleW >= xx && middleH >= yy &&
			middleW <= xx+ww && middleH <= yy+hh {
			return xsp.screens[i]
		}
	}
	return xsp.screens[0]
}

func getScale(widthPx, widthMm uint16) float32 {
	dpi := float32(widthPx) / (float32(widthMm) / 25.4)
	if dpi > 1000 || dpi < 10 {
		dpi = 96
	}
	return float32(math.Round(float64(dpi)/96.0*10.0)) / 10.0
}

func (xsp *x11ScreensProvider) setupScreens() {
	resources, err := randr.GetScreenResources(xsp.x.x.Conn(), xsp.root).Reply()
	if err != nil || len(resources.Outputs) == 0 {
		fyne.LogError("Could not get randr screen resources", err)
		xsp.setupSingleScreen()
		return
	}

	var primaryInfo *randr.GetOutputInfoReply
	primary, err := randr.GetOutputPrimary(xsp.x.x.Conn(), xsp.root).Reply()
	if err == nil {
		primaryInfo, _ = randr.GetOutputInfo(xsp.x.x.Conn(), primary.Output, 0).Reply()
	}
	primaryFound := false
	for _, output := range resources.Outputs {
		outputInfo, err := randr.GetOutputInfo(xsp.x.x.Conn(), output, 0).Reply()
		if err != nil {
			fyne.LogError("Could not get randr output", err)
			continue
		}
		if outputInfo.Crtc == 0 || outputInfo.Connection == randr.ConnectionDisconnected {
			continue
		}
		crtcInfo, err := randr.GetCrtcInfo(xsp.x.x.Conn(), outputInfo.Crtc, 0).Reply()
		if err != nil {
			fyne.LogError("Could not get randr crtcs", err)
			continue
		}
		insertIndex := -1
		for i, screen := range xsp.screens {
			if screen.X >= int(crtcInfo.X) && screen.Y >= int(crtcInfo.Y) {
				insertIndex = i
				break
			}
		}
		if insertIndex == -1 {
			xsp.screens = append(xsp.screens, &desktop.Screen{Name: string(outputInfo.Name),
				X: int(crtcInfo.X), Y: int(crtcInfo.Y), Width: int(crtcInfo.Width), Height: int(crtcInfo.Height),
				Scale: getScale(crtcInfo.Width, uint16(outputInfo.MmWidth))})
			insertIndex = len(xsp.screens) - 1
		} else {
			xsp.screens = append(xsp.screens, nil)
			copy(xsp.screens[insertIndex+1:], xsp.screens[insertIndex:])
			xsp.screens[insertIndex] = &desktop.Screen{Name: string(outputInfo.Name),
				X: int(crtcInfo.X), Y: int(crtcInfo.Y), Width: int(crtcInfo.Width), Height: int(crtcInfo.Height),
				Scale: getScale(crtcInfo.Width, uint16(outputInfo.MmWidth))}
		}
		if primaryInfo != nil {
			if string(primaryInfo.Name) == string(outputInfo.Name) {
				primaryFound = true
				xsp.primary = xsp.screens[insertIndex]
				xsp.active = xsp.screens[insertIndex]
			}
		}
	}
	if !primaryFound {
		xsp.primary = xsp.screens[0]
		xsp.active = xsp.screens[0]
	}
}

func (xsp *x11ScreensProvider) setupSingleScreen() {
	xsp.single = true
	xsp.screens = append(xsp.screens, &desktop.Screen{Name: "Screen0",
		X: xwindow.RootGeometry(xsp.x.x).X(), Y: xwindow.RootGeometry(xsp.x.x).Y(),
		Width: xwindow.RootGeometry(xsp.x.x).Width(), Height: xwindow.RootGeometry(xsp.x.x).Height(),
		Scale: 1.0})
	xsp.primary = xsp.screens[0]
	xsp.active = xsp.screens[0]
}
