package cloudmusic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SongDetail 通过歌曲 ID 拉取详情。
// 搜索接口不返回封面 URL,封面等信息需要走这个接口补全。
func SongDetail(id int) (*Song, error) {
	url := fmt.Sprintf("https://music.163.com/api/song/detail?id=%d&ids=%%5B%d%%5D", id, id)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Songs []Song `json:"songs"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if len(result.Songs) == 0 {
		return nil, fmt.Errorf("song detail not found: id=%d", id)
	}

	song := result.Songs[0]
	return &song, nil
}

// CoverURL 返回封面地址,picUrl 优先、pic_str 兜底
func (s Song) CoverURL() string {
	if s.Album.PicURL != "" {
		return s.Album.PicURL
	}
	return s.Album.PicURL2
}
