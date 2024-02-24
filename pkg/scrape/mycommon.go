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

// URLとドメインの一致を確認
func domainMatch(urlString, pattern string) bool {
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

// DMM API呼び出し時のQueryパラメータを追加
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

// Query Parameterを追加する
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

// Query Parameterの名前を置き換える
func replaceQueryParam(originalURL string, paramname string, newParamname string) (string, error) {
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}

	queryParams := parsedURL.Query()
	values := queryParams[paramname]
	if len(values) > 0 {
		// パラメーターが見つかった場合、新しいパラメーター名に置き換える
		queryParams.Del(paramname)
		for _, value := range values {
			queryParams.Add(newParamname, value)
		}
		parsedURL.RawQuery = queryParams.Encode()
	} else {
		return "", errors.New("not found parameter")
	}

	return parsedURL.String(), nil
}

// DVD-ID形式をDMM形式に変換
func ConvertFormat(input string) string {
	re := regexp.MustCompile(`([a-zA-Z]{3,4})-(\d{3,})`)
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
		replacement := prefix + numStr
		input = strings.Replace(input, match[0], replacement, -1)
	}

	return input
}

// DMM形式をDVD-ID形式に変換
func ConvertToDVDId(input string) string {
	re := regexp.MustCompile(`^([H_]*)(\d*)([a-zA-Z]+)(\d+)`)
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

func isQuoted(input string) bool {
	// 文字列がダブルクオーテーションで始まり、終わるかどうかを確認
	return strings.HasPrefix(input, "\"") && strings.HasSuffix(input, "\"")
}

func getQuotedString(input string) (string, error) {
	// ダブルクオーテーションで始まり、終わるかどうかを確認
	if strings.HasPrefix(input, "\"") && strings.HasSuffix(input, "\"") {
		// ダブルクオーテーションを除去して返す
		return strings.Trim(input, "\""), nil
	}
	// ダブルクオーテーションで囲まれた文字列が見つからない場合、エラーを返す
	return "", errors.New("quoted string not found")
}
