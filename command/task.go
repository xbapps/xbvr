package command

import (
	"github.com/cld9x/xbvr/xbase"
	"github.com/cld9x/xbvr/xbase/scrape"
	"gopkg.in/urfave/cli.v1"
)

func ActionCleanTags(c *cli.Context) {
	xbase.RenameTags()
	xbase.CountTags()
}

func ActionScrape(c *cli.Context) {
	scrape.ScrapeNA()
	scrape.ScrapeBadoink()
	scrape.ScrapeMilfVR()
	scrape.ScrapeVRB()
	scrape.ScrapeWankz()
	scrape.ScrapeVirtualTaboo()
}

func init() {
	RegisterCommand(cli.Command{
		Name:  "task",
		Usage: "run various tasks",
		Subcommands: []cli.Command{
			{
				Name:     "clean-tags",
				Category: "tasks",
				Usage:    "clean tags",
				Action:   ActionCleanTags,
			},
			{
				Name:     "scrape",
				Category: "tasks",
				Usage:    "run scrapers",
				Action:   ActionScrape,
			},
		},
	})
}
