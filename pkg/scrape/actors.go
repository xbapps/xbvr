package scrape

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func Actors(wg *sync.WaitGroup, knownActors []string, out chan<- models.ScrapedActor) error {
	defer wg.Done()

	siteCollector := colly.NewCollector(
		colly.AllowedDomains("www.hottiesvr.com"),
		colly.CacheDir(siteCacheDir),
		colly.UserAgent(userAgent),
	)

	actorCollector := colly.NewCollector(
		colly.AllowedDomains("www.hottiesvr.com"),
		colly.CacheDir(sceneCacheDir),
		colly.UserAgent(userAgent),
	)

	siteCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	actorCollector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	actorCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sa := models.ScrapedActor{}

		sa.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]

		tmp := strings.Split(sa.HomepageURL, "/")
		sa.ActorID = strings.Split(tmp[len(tmp)-1], "-")[0]
		sa.ImageURL = e.ChildAttr(`.item-img img`, "src")

		sa.Name = strings.TrimSpace(e.ChildText(`.model-page h1`))

		aliases := strings.Replace(e.ChildText(`.item-aliases`), "Aliases", "", 1)
		if len(strings.TrimSpace(aliases)) > 0 {
			tmpAliases := trimSpaceFromSlice(strings.Split(aliases, ", "))
			if len(tmpAliases) > 0 {
				j, _ := json.Marshal(tmpAliases)
				sa.Aliases = string(j)
			}
		}

		sa.Bio = strings.TrimSpace(strings.Replace(e.ChildText(`.item-bio`), "Bio", "", 1))

		statKeys := []string{}
		statVals := []string{}

		e.ForEach(`.facts dl dt`, func(id int, e *colly.HTMLElement) {
			statKeys = append(statKeys, strings.TrimSpace(e.Text))
		})
		e.ForEach(`.facts dl dd`, func(id int, e *colly.HTMLElement) {
			statVals = append(statVals, strings.TrimSpace(e.Text))
		})

		if len(statKeys) == len(statVals) {
			stats := make(map[string]string)
			for i := 0; i < len(statKeys); i++ {
				stats[statKeys[i]] = statVals[i]
			}

			if stats["Date of Birth"] != "" {
				// January 01, 1990
				// TODO(jrebey): check if we need to parse this
				sa.Birthday = stats["Date of Birth"]
			}
			if stats["Hair Color"] != "" {
				sa.HairColor = stats["Hair Color"]
			}
			if stats["Eye Color"] != "" {
				sa.EyeColor = stats["Eye Color"]
			}
			if stats["Ethnicity"] != "" {
				sa.Ethnicity = stats["Ethnicity"]
			}
			if stats["Height"] != "" {
				// 168 cm - 5 feet and 6 inches
				// TODO(jrebey): check if we need to parse this
				sa.Height = stats["Height"]
			}
			if stats["Weight"] != "" {
				// 52 kg - 114 lbs
				// TODO(jrebey): check if we need to parse this
				sa.Weight = stats["Weight"]
			}
			if stats["Measurements"] != "" {
				// 34A-25-35
				// JP 86-55-85 (US 34-22-33)
				// TODO(jrebey): check if we need to parse this
				sa.Measurements = stats["Measurements"]
			}
			if stats["Country of Origin"] != "" {
				sa.Nationality = stats["Country of Origin"]
			}
		}

		e.ForEach(`.nav-social li a`, func(id int, e *colly.HTMLElement) {
			sn := strings.TrimSpace(e.Text)
			if sn == "Facebook" {
				sa.Facebook = e.Attr("href")
			}
			if sn == "Instagram" {
				sa.Instagram = e.Attr("href")
			}
			if sn == "Reddit" {
				sa.Reddit = e.Attr("href")
			}
		})

		sa.Twitter = e.ChildAttr(`a.twitter-timeline`, "href")

		out <- sa
	})

	siteCollector.OnHTML(`.model a`, func(e *colly.HTMLElement) {
		actorURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownActors, actorURL) {
			actorCollector.Visit(actorURL)
		}

	})

	siteCollector.Visit("https://www.hottiesvr.com/virtualreality/alphabet")

	return nil
}
