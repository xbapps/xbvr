package tray

import (
	"fmt"
	"os"

	"github.com/ProtonMail/go-appdir"
	"github.com/cld9x/xbvr/xbase"
	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

func Run() {
	systray.Run(onReady, onExit)
}

func onExit() {
	systray.Quit()
	os.Exit(0)
}

func onReady() {
	go func() {
		xbase.StartServer()
	}()

	systray.SetIcon(iconData)
	systray.SetTooltip(fmt.Sprintf("XBVR"))

	systray.AddSeparator()

	mOpenUI := systray.AddMenuItem("Open UI", "Open UI")
	mOpenConfig := systray.AddMenuItem("Open config folder", "Open config folder")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit XBVR")


	for {
		select {
		case <-mOpenUI.ClickedCh:
			go open.Run("http://localhost:9999")
		case <-mOpenConfig.ClickedCh:
			go open.Run(appdir.New("xbvr").UserConfig())
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}
