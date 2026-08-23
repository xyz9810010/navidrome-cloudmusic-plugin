package netease

import (
	"encoding/json"
	"fmt"
)

// SongDetail 通过歌曲 ID 拉取详情。
// 搜索接口不返回封面 URL,封面等信息需要走这个接口补全。
func SongDetail(id int) (*Song, error) {
	body, err := httpGet(fmt.Sprintf("https://music.163.com/api/song/detail?id=%d&ids=%%5B%d%%5D", id, id))
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("song detail empty: id=%d", id)
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
	return &result.Songs[0], nil
}

// CoverURL 返回封面地址,picUrl 优先、pic_str 兜底
func (s Song) CoverURL() string {
	if s.Album.PicURL != "" {
		return s.Album.PicURL
	}
	return s.Album.PicURL2
}

// AlbumDetail 专辑详情(/api/v1/album)
func AlbumDetail(id int) (name, description, picURL string, err error) {
	body, err := httpGet(fmt.Sprintf("https://music.163.com/api/v1/album/%d", id))
	if err != nil || body == nil {
		return "", "", "", err
	}
	var result struct {
		Album struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			PicURL      string `json:"picUrl"`
		} `json:"album"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", "", err
	}
	return result.Album.Name, result.Album.Description, result.Album.PicURL, nil
}

// ArtistDetail 歌手详情(/api/v1/artist)
func ArtistDetail(id int) (briefDesc string, err error) {
	body, err := httpGet(fmt.Sprintf("https://music.163.com/api/v1/artist/%d", id))
	if err != nil || body == nil {
		return "", err
	}
	var result struct {
		Artist struct {
			BriefDesc string `json:"briefDesc"`
		} `json:"artist"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Artist.BriefDesc, nil
}
