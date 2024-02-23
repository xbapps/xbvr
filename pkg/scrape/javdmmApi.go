package scrape

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

const (
	PARAM_SORT      = "match"
	count           = "2"
	dmm_BaseAddress = "https://api.dmm.com/"
	//dmm_itemListSearchDigitalUrl  = dmm_BaseAddress + "affiliate/v3/ItemList?api_id=" + dmm_appid + "&affiliate_id=" + dmm_affiliateid + "&site=FANZA&service=digital&sort=" + PARAM_SORT + "&output=json"
	dmm_itemListSearchDigitalUrl  = dmm_BaseAddress + "affiliate/v3/ItemList?site=FANZA&service=digital&output=json&sort=" + PARAM_SORT
	dmm_actorListSearchDigitalUrl = dmm_BaseAddress + "affiliate/v3/ActressSearch"
)

type JSONResponse struct {
	Request struct {
		Parameters struct {
			AffiliateID string `json:"affiliate_id"`
			APIID       string `json:"api_id"`
			Floor       string `json:"floor"`
			Keyword     string `json:"keyword"`
			Service     string `json:"service"`
			Site        string `json:"site"`
		} `json:"parameters"`
	} `json:"request"`
	Result struct {
		FirstPosition int64 `json:"first_position"`
		Items         []struct {
			URL          string `json:"URL"`
			AffiliateURL string `json:"affiliateURL"`
			CategoryName string `json:"category_name"`
			ContentID    string `json:"content_id"`
			Date         string `json:"date"`
			FloorCode    string `json:"floor_code"`
			FloorName    string `json:"floor_name"`
			ImageURL     struct {
				Large string `json:"large"`
				List  string `json:"list"`
				Small string `json:"small"`
			} `json:"imageURL"`
			Iteminfo struct {
				Actress []struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
					Ruby string `json:"ruby"`
				} `json:"actress"`
				Director []struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
					Ruby string `json:"ruby"`
				} `json:"director"`
				Genre []struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
				} `json:"genre"`
				Label []struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
				} `json:"label"`
				Maker []struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
				} `json:"maker"`
				Series []struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
				} `json:"series"`
			} `json:"iteminfo"`
			Prices struct {
				Deliveries struct {
					Delivery []struct {
						Price string `json:"price"`
						Type  string `json:"type"`
					} `json:"delivery"`
				} `json:"deliveries"`
				Price string `json:"price"`
			} `json:"prices"`
			ProductID string `json:"product_id"`
			Review    struct {
				Average string `json:"average"`
				Count   int64  `json:"count"`
			} `json:"review"`
			SampleImageURL struct {
				SampleL struct {
					Image []string `json:"image"`
				} `json:"sample_l"`
				SampleS struct {
					Image []string `json:"image"`
				} `json:"sample_s"`
			} `json:"sampleImageURL"`
			SampleMovieURL struct {
				PcFlag      int64  `json:"pc_flag"`
				Size476_306 string `json:"size_476_306"`
				Size560_360 string `json:"size_560_360"`
				Size644_414 string `json:"size_644_414"`
				Size720_480 string `json:"size_720_480"`
				SpFlag      int64  `json:"sp_flag"`
			} `json:"sampleMovieURL"`
			ServiceCode string `json:"service_code"`
			ServiceName string `json:"service_name"`
			Title       string `json:"title"`
			Volume      string `json:"volume"`
		} `json:"items"`
		ResultCount int64 `json:"result_count"`
		Status      int64 `json:"status"`
		TotalCount  int64 `json:"total_count"`
	} `json:"result"`
}

func addAPIParam(originalURL string) (string, error) {

	if config.Config.Advanced.DmmApiId == "" || config.Config.Advanced.DmmAffiliateId == "" {
		return "", errors.New("is not set DmmApiId and DmmAffiliateId param")
	}

	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}

	queryParams := parsedURL.Query()
	queryParams.Set("api_id", config.Config.Advanced.DmmApiId)
	queryParams.Set("affiliate_id", config.Config.Advanced.DmmAffiliateId)
	parsedURL.RawQuery = queryParams.Encode()

	return parsedURL.String(), nil
}

func addQueryParam(originalURL string, paramname string, value string) (string, error) {
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}
	queryParams := parsedURL.Query()
	queryParams.Set(paramname, value)
	parsedURL.RawQuery = queryParams.Encode()
	return parsedURL.String(), nil
}

func ConvertFormat(input string) string {
	re := regexp.MustCompile(`([a-zA-Z]{4})-(\d{3,})`)
	matches := re.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		prefix := match[1]
		numStr := match[2]

		// 数字が4桁以上の場合はゼロパディングを行わない
		num, err := strconv.Atoi(numStr)
		if err == nil {
			if num < 10000 {
				numStr = fmt.Sprintf("%05d", num)
			}
		}

		// 修正後の文字列を組み立てる
		replacement := prefix + numStr
		input = strings.Replace(input, match[0], replacement, -1)
	}

	return input
}

func ConvertToDVDId(input string) string {
	re := regexp.MustCompile(`^([H_]*)(\d*)([a-zA-Z]+)(\d+)$`)
	matches := re.FindStringSubmatch(input)

	if len(matches) == 5 {
		prefix := strings.ToUpper(strings.Replace(matches[3], "_", "", 1))
		num, err := strconv.Atoi(matches[4])
		if err == nil {
			numStr := fmt.Sprintf("%03d", num) // 3桁の数字に変換
			return fmt.Sprintf("%s-%s", prefix, numStr)
		}
	}
	return ""
}

func getJSONResponse(url string) (*JSONResponse, error) {
	// HTTP GETリクエストを送信してレスポンスを取得
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスのBodyを読み込み
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// JSONデータを構造体にパース
	var jsonResponse JSONResponse
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return nil, err
	}

	return &jsonResponse, nil
}

func ScrapeDMMapi(out *[]models.ScrapedScene, queryString string) {

	sceneCollector := createCollector("api.dmm.com")

	sceneCollector.OnResponse(func(r *colly.Response) {
		log.Println("OnResponse." + r.Ctx.Get("url"))

		body, err := ioutil.ReadAll(bytes.NewReader(r.Body))
		if err != nil {
			log.Println("Error:", err)
			return
		}

		// JSONデータを構造体にパース
		var jsonResponse JSONResponse
		if err := json.Unmarshal(body, &jsonResponse); err != nil {
			log.Println("Error:", err)
			return
		}

		if jsonResponse.Result.ResultCount == 0 {
			log.Info("not found")
			return
		}

		log.Println("Response:", jsonResponse.Result.Items[0].Title)

		sc := models.ScrapedScene{}
		sc.SceneType = "VR"
		//ContentID := ""

		sc.Tags = append(sc.Tags, `JAVR`)
		sc.Tags = append(sc.Tags, `FANZA`)

		if jsonResponse.Result.ResultCount == 0 {
			return
		}
		sc.Title = jsonResponse.Result.Items[0].Title
		sc.Studio = jsonResponse.Result.Items[0].Iteminfo.Label[0].Name
		log.Info("(0)" + jsonResponse.Result.Items[0].ProductID)
		dvdId := strings.ToUpper(jsonResponse.Result.Items[0].ProductID)
		log.Info("(dvdId)" + dvdId)
		sc.SceneID = ConvertToDVDId(dvdId)
		log.Info("(1)" + sc.SceneID)

		sc.SiteID = dvdId
		siteParts := strings.Split(sc.SceneID, `-`)
		if len(siteParts) > 0 {
			sc.Site = siteParts[0]
		}

		tmpDate, _ := goment.New(strings.TrimSpace(jsonResponse.Result.Items[0].Date), "YYYY-MM-DD HH:mm:ss")
		sc.Released = tmpDate.Format("YYYY-MM-DD")

		sc.Covers = append(sc.Covers, jsonResponse.Result.Items[0].ImageURL.Large)
		sc.HomepageURL = jsonResponse.Result.Items[0].URL
		sc.Studio = jsonResponse.Result.Items[0].Iteminfo.Maker[0].Name
		sc.Duration, _ = strconv.Atoi(jsonResponse.Result.Items[0].Volume)

		log.Info("(2)" + sc.SceneID)
		for _, item := range jsonResponse.Result.Items[0].Iteminfo.Genre {
			tag := ProcessJavrTag(item.Name)
			sc.Tags = append(sc.Tags, tag)
		}

		sc.ActorDetails = make(map[string]models.ActorDetails)
		for _, item := range jsonResponse.Result.Items[0].Iteminfo.Actress {
			sc.Cast = append(sc.Cast, item.Name)
			sc.Aliases = append(sc.Aliases, item.Ruby)
			actressurl, err := addQueryParam(dmm_actorListSearchDigitalUrl, "actress_id", strconv.FormatInt(item.ID, 10))
			if err != nil {
				//url := Addquedmm_actorListSearchDigitalUrl + strconv.FormatInt(item.ID, 10)
				log.Info("actordetail url:" + actressurl)
				log.Info("detail :" + sc.ActorDetails[item.Name].Source)
				sc.ActorDetails[item.Name] = models.ActorDetails{Source: "dmm scrape", ProfileUrl: actressurl}
				log.Info("add actordetail name:" + item.Name)
				log.Info("add actor profileurl:" + sc.ActorDetails[item.Name].ProfileUrl)
			}
		}

		log.Info("(3)" + sc.SceneID)
		// Screenshots
		for _, item := range jsonResponse.Result.Items[0].SampleImageURL.SampleL.Image {
			sc.Gallery = append(sc.Gallery, item)
		}
		// Synopsis
		sc.Synopsis = sc.Title

		// Apply post-processing for error-correcting code
		//PostProcessJavScene(&sc, dvdId)

		log.Info("(4)" + sc.SceneID)
		if sc.SceneID != "" {
			*out = append(*out, sc)
		}

	})

	// Allow comma-separated scene id's
	scenes := strings.Split(queryString, ",")
	queryurl, err := addAPIParam(dmm_itemListSearchDigitalUrl)
	if err != nil {
		return
	}
	queryurl, err = addQueryParam(queryurl, "hits", count)
	if err != nil {
		return
	}

	for _, v := range scenes {
		queryurl, err = addQueryParam(queryurl, "keyword", ConvertFormat(strings.ToLower(v)))
		if err == nil {
			//sceneCollector.Visit(dmm_itemListSearchDigitalUrl + "&hits=" + count + "&keyword=" + ConvertFormat(strings.ToLower(v)))
			sceneCollector.Visit(queryurl)
		}

	}

	sceneCollector.Wait()
}
