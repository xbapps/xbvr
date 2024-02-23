package scrape

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/xbapps/xbvr/pkg/config"
)

func domainMatch(urlString, pattern string) bool {
	// URLをパースしてホスト部分（ドメイン）を取得
	u, err := url.Parse(urlString)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return false
	}
	host := u.Hostname()

	pattern = "^" + regexp.QuoteMeta(pattern) + "$"
	pattern = strings.ReplaceAll(pattern, "\\*", "[^.]+") // ワイルドカードを正規表現に変換
	re := regexp.MustCompile(pattern)
	isMatch := re.MatchString(host)

	return isMatch
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
