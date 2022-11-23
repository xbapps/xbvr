package api

import (
	"encoding/json"
	"html"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/gocolly/colly"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

type VideoSourceResponse struct {
	VideoSources []VideoSource `json:"video_sources"`
}

func LoadHeresphereScene(url string) HeresphereVideo {
	response, err := http.Get(url)

	if err != nil {
		return HeresphereVideo{}
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error from %s %s", url, err)
	}

	var video HeresphereVideo
	err = json.Unmarshal(responseData, &video)
	if err != nil {
		log.Errorf("Error from %s %s", url, err)
	}

	return video
}

func LoadDeovrScene(url string) DeoScene {
	response, err := http.Get(url)

	if err != nil {
		return DeoScene{}
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error from %s %s", url, err)
	}

	var video DeoScene
	err = json.Unmarshal(responseData, &video)
	if err != nil {
		log.Errorf("Error from %s %s", url, err)
	}

	db, _ := models.GetDB()
	defer db.Close()

	return video
}

type VideoSource struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
}

func ScrapeHtml(scrapeParams string) VideoSourceResponse {
	c := colly.NewCollector(colly.UserAgent(scrape.UserAgent))
	var params models.TrailerScrape
	json.Unmarshal([]byte(scrapeParams), &params)

	var srcs []VideoSource
	c.OnHTML(`html`, func(e *colly.HTMLElement) {
		e.ForEach(params.HtmlElement, func(id int, e *colly.HTMLElement) {
			if params.ExtractRegex == "" {
				origURLtmp := e.Attr(params.ContentPath)
				quality := e.Attr(params.QualityPath)
				if origURLtmp != "" {
					if params.ContentBaseUrl != "" {
						origURLtmp = params.ContentBaseUrl + origURLtmp
					}
					srcs = append(srcs, VideoSource{URL: origURLtmp, Quality: quality})
				}
			} else {
				//  extract match with regex expression if one was specified
				var re = regexp.MustCompile(params.ExtractRegex)
				r := re.FindStringSubmatch(e.Text)
				if len(r) > 0 {
					if r[1] != "" {
						srcs = append(srcs, VideoSource{URL: r[1], Quality: "unknown"})
					}
				}
			}
		})
	})
	c.Visit(params.SceneUrl)

	r := VideoSourceResponse{
		VideoSources: srcs,
	}
	return r

}

func ScrapeJson(scrapeParams string) VideoSourceResponse {
	c := colly.NewCollector(colly.UserAgent(scrape.UserAgent))
	var params models.TrailerScrape
	json.Unmarshal([]byte(scrapeParams), &params)

	var srcs []VideoSource
	c.OnHTML(`html`, func(e *colly.HTMLElement) {
		e.ForEach(params.HtmlElement, func(id int, e *colly.HTMLElement) {
			txt := e.Text
			//  extract json with regex expression if one was specified
			if params.ExtractRegex != "" {
				var re = regexp.MustCompile(params.ExtractRegex)
				r := re.FindStringSubmatch(txt)
				if len(r) > 0 {
					if r[1] != "" {
						txt = r[1]
					}
				}
			}

			srcs = extractFromJson(txt, params, srcs)
		})
	})
	c.Visit(params.SceneUrl)

	r := VideoSourceResponse{
		VideoSources: srcs,
	}
	return r
}

func LoadJson(scrapeParams string) VideoSourceResponse {
	var params models.TrailerScrape
	json.Unmarshal([]byte(scrapeParams), &params)

	response, err := http.Get(params.SceneUrl)

	if err != nil {
		return VideoSourceResponse{}
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Error from %s %s", params.SceneUrl, err)
	}

	var resp VideoSourceResponse
	resp.VideoSources = extractFromJson(string(responseData), params, []VideoSource{})

	return resp
}

func extractFromJson(inputJson string, params models.TrailerScrape, srcs []VideoSource) []VideoSource {
	JsonMetadata := strings.TrimSpace(inputJson)

	// if the path to a record exists loop through each video source
	if params.RecordPath != "" {
		u := gjson.Get(JsonMetadata, params.RecordPath)
		u.ForEach(func(key, value gjson.Result) bool {
			url := gjson.Get(value.String(), params.ContentPath).String()
			quality := gjson.Get(value.String(), params.QualityPath).String()
			encoding := ""
			if params.EncodingPath != "" {
				encoding = gjson.Get(value.String(), params.EncodingPath).String() + "-"
			}

			if url != "" {
				srcs = append(srcs, VideoSource{URL: url, Quality: encoding + quality})
			}
			return true // keep iterating
		})
	} else {
		// get single entry, ie not a repeating group of video sources
		if gjson.Get(JsonMetadata, params.ContentPath).String() != "" {
			quality := strings.TrimSpace(html.UnescapeString(gjson.Get(JsonMetadata, params.QualityPath).String()))
			url := strings.TrimSpace(html.UnescapeString(gjson.Get(JsonMetadata, params.ContentPath).String()))
			encoding := ""
			if params.EncodingPath != "" {
				encoding = gjson.Get(JsonMetadata, params.EncodingPath).String() + "-"
			}
			if params.ContentBaseUrl != "" {
				if params.ContentBaseUrl[len(params.ContentBaseUrl)-1:] == "/" && string(url[0]) == "/" {
					url = params.ContentBaseUrl + url[1:]
				} else {
					url = params.ContentBaseUrl + url
				}
			}
			srcs = append(srcs, VideoSource{URL: url, Quality: encoding + quality})
		}
	}
	return srcs
}

func LoadUrl(url string) VideoSourceResponse {

	var srcs []VideoSource
	srcs = append(srcs, VideoSource{URL: url, Quality: "Unknown"})

	r := VideoSourceResponse{
		VideoSources: srcs,
	}
	return r
}
