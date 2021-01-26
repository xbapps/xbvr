package main

//go:generate fileb0x .assets.toml

import (
	"github.com/xbapps/xbvr/pkg/server"
)

var version = "CURRENT"
var commit = "HEAD"
var branch = "master"
var date = "moment ago"

func main() {
	server.StartServer(version, commit, branch, date)
}
