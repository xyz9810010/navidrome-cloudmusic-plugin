package netease

import (
	"encoding/json"
	"net/url"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
)

// QQAlbum QQ 音乐专辑搜索结果
type QQAlbum struct {
	MID    string `json:"albumMID"`
	Name   string `json:"albumName"`
	Singer string `json:"singerName"`
}

// QQSearchAlbum 搜索 QQ 音乐专辑(封面源:QQ 的封面无"博纳正版授权"等水印)
func QQSearchAlbum(keyword string) ([]QQAlbum, error) {
	raw := "https://c.y.qq.com/soso/fcgi-bin/client_search_cp?w=" +
		url.QueryEscape(keyword) + "&t=8&format=json&p=1&n=5"
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method:  "GET",
		URL:     raw,
		Headers: map[string]string{"Referer": "https://y.qq.com/", "User-Agent": userAgent},
		TimeoutMs: 8000,
	})
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return nil, err
	}
	var result struct {
		Data struct {
			Album struct {
				List []QQAlbum `json:"list"`
			} `json:"album"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, err
	}
	return result.Data.Album.List, nil
}

// QQAlbumCoverURL QQ 专辑封面直链(800x800,固定规格)
func QQAlbumCoverURL(mid string) string {
	return "https://y.gtimg.cn/music/photo_new/T002R800x800M000" + mid + ".jpg"
}
