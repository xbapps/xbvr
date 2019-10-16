package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/ProtonMail/go-appdir"
	"github.com/getlantern/systray"
	"github.com/marcsauter/single"
	"github.com/skratchdot/open-golang/open"
	"github.com/xbapps/xbvr/pkg/assets"
	"github.com/xbapps/xbvr/pkg/xbvr"
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
		xbvr.StartServer(version, commit, branch, date)
	}()

	if runtime.GOOS == "windows" {
		systray.SetIcon(assets.FileIconsXbvrWinIco)
	} else {
		systray.SetIcon(assets.FileIconsXbvr128Png)
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
			go open.Run("http://localhost:9999")
		case <-mOpenConfig.ClickedCh:
			go open.Run(appdir.New("xbvr").UserConfig())
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}
