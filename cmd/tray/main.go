package main

import (
	"github.com/cld9x/xbvr/xbase/tray"
	"github.com/marcsauter/single"
)

func main() {
	s := single.New("xbvr")
	s.Lock()
	defer s.Unlock()

	tray.Run()
}
