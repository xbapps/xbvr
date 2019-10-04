package main

//go:generate fileb0x .assets.toml

import (
	"github.com/xbapps/xbvr/pkg/xbvr"
)

var version = "CURRENT"
var commit = "HEAD"
var branch = "master"
var date = "moment ago"

func main() {
	xbvr.StartServer(version, commit, branch, date)
}
