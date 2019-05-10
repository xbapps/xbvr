package command

import (
	"os"
	"path/filepath"

	"github.com/cld9x/xbvr/xbase"
	"gopkg.in/urfave/cli.v1"
)

func ActionAddVolume(c *cli.Context) {
	if len(c.Args()) == 1 {
		path := c.Args()[0]
		if fi, err := os.Stat(path); os.IsNotExist(err) || !fi.IsDir() {
			log.Fatal("Path does not exist or is not a directory")
		}

		path, _ = filepath.Abs(path)

		db, _ := xbase.GetDB()
		defer db.Close()

		var vol []xbase.Volume
		db.Where(&xbase.Volume{Path: path}).Find(&vol)

		if len(vol) > 0 {
			log.Fatal("Volume already exists")
		}

		nv := xbase.Volume{Path: path, IsEnabled: true, IsAvailable: true}
		nv.Save()

		log.Info("Added new volume", path)
	}
}

func init() {
	RegisterCommand(cli.Command{
		Name:  "volume",
		Usage: "manage Volumes",
		Subcommands: []cli.Command{
			{
				Name:     "add",
				Category: "volume",
				Usage:    "add new volume",
				Action:   ActionAddVolume,
			},
		},
	})
}
