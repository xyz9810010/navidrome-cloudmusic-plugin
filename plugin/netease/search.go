package netease

import (
	"encoding/json"
	"net/url"
	"strconv"
)

const searchAPI = "https://music.163.com/api/search/get"

// search 搜索,type: 1歌曲 10专辑 100歌手
func search(keyword string, typeID, limit int) ([]byte, error) {
	data := url.Values{}
	data.Set("s", keyword)
	data.Set("type", strconv.Itoa(typeID))
	data.Set("limit", strconv.Itoa(limit))
	return httpPostForm(searchAPI, data)
}

// SearchSong 搜歌曲
func SearchSong(keyword string) ([]Song, error) {
	body, err := search(keyword, 1, 30)
	if err != nil || body == nil {
		return nil, err
	}
	var result struct {
		Result struct {
			Songs []Song `json:"songs"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Result.Songs, nil
}

// AlbumResult 专辑搜索结果
type AlbumResult struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	PicURL string `json:"picUrl"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
}

// SearchAlbum 搜专辑
func SearchAlbum(keyword string) ([]AlbumResult, error) {
	body, err := search(keyword, 10, 20)
	if err != nil || body == nil {
		return nil, err
	}
	var result struct {
		Result struct {
			Albums []AlbumResult `json:"albums"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Result.Albums, nil
}

// ArtistResult 歌手搜索结果
type ArtistResult struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	PicURL string `json:"picUrl"`
	Img1v1 string `json:"img1v1Url"`
}

// SearchArtist 搜歌手
func SearchArtist(keyword string) ([]ArtistResult, error) {
	body, err := search(keyword, 100, 10)
	if err != nil || body == nil {
		return nil, err
	}
	var result struct {
		Result struct {
			Artists []ArtistResult `json:"artists"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Result.Artists, nil
}
