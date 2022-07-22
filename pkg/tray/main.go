package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ProtonMail/go-appdir"
	"github.com/getlantern/systray"
	"github.com/marcsauter/single"
	"github.com/skratchdot/open-golang/open"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/server"
	"github.com/xbapps/xbvr/ui"
)

var version = "CURRENT"
var commit = "HEAD"
var branch = "master"
var date = "moment ago"

func main() {
	s := single.New("xbvr")
	s.Lock()
	defer s.Unlock()

	systray.Run(onReady, onExit)
}

func onExit() {
	systray.Quit()
	os.Exit(0)
}

func onReady() {
	go func() {
		server.StartServer(version, commit, branch, date)
	}()

	if runtime.GOOS == "windows" {
		iconData, _ := ui.Assets.ReadFile("dist/icons/xbvr-win.ico")
		systray.SetIcon(iconData)
	} else {
		iconData, _ := ui.Assets.ReadFile("dist/icons/xbvr-128.png")
		systray.SetIcon(iconData)
	}
	systray.SetTooltip(fmt.Sprintf("XBVR"))

	systray.AddSeparator()

	mOpenUI := systray.AddMenuItem("Open UI", "Open UI")
	mOpenConfig := systray.AddMenuItem("Open config folder", "Open config folder")

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit XBVR")

	for {
		select {
		case <-mOpenUI.ClickedCh:
			go open.Run(fmt.Sprintf("http://localhost:%v", config.Config.Server.Port))
		case <-mOpenConfig.ClickedCh:
			go open.Run(appdir.New("xbvr").UserConfig())
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}
