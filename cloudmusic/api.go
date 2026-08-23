package cloudmusic

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func SearchRequest(keyword string) (*http.Response, error) {

	data := url.Values{}

	data.Set(
		"s",
		keyword,
	)

	data.Set(
		"type",
		"1",
	)

	data.Set(
		"limit",
		"30",
	)

	return http.PostForm(
		"https://music.163.com/api/search/get",
		data,
	)

}

func httpPostForm(rawURL string, data url.Values) (*http.Response, error) {

	req, err := http.NewRequest(
		"POST",
		rawURL,
		strings.NewReader(data.Encode()),
	)

	if err != nil {

		return nil, err

	}

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	req.Header.Set(
		"Referer",
		"https://music.163.com/",
	)

	return http.DefaultClient.Do(req)

}

func jsonInt(n int) string {

	return strconv.Itoa(n)

}
