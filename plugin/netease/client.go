// Package netease 是网易云 API 客户端的插件(WASM)版本。
// 与根模块 cloudmusic 包逻辑一致,HTTP 全部改走 Navidrome 宿主的 HTTPSend。
package netease

import (
	"net/url"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"

func baseHeaders() map[string]string {
	return map[string]string{
		"User-Agent": userAgent,
		"Referer":    "https://music.163.com/",
	}
}

// httpGet GET 请求,返回响应体
func httpGet(rawURL string) ([]byte, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method:  "GET",
		URL:     rawURL,
		Headers: baseHeaders(),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.StatusCode != 200 {
		return nil, nil
	}
	return resp.Body, nil
}

// httpPostForm 表单 POST,返回响应体
func httpPostForm(rawURL string, data url.Values) ([]byte, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method: "POST",
		URL:    rawURL,
		Headers: map[string]string{
			"User-Agent":   userAgent,
			"Referer":      "https://music.163.com/",
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body: []byte(data.Encode()),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.StatusCode != 200 {
		return nil, nil
	}
	return resp.Body, nil
}

// PicWithRes 网易云图片按需缩放
func PicWithRes(pic, res string) string {
	if pic == "" {
		return ""
	}
	pic = strings.Replace(pic, "http://", "https://", 1)
	if !strings.Contains(pic, "music.126.net") {
		return pic
	}
	return pic + "?param=" + res + "y" + res
}
