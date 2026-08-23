package cloudmusic

import (
	"encoding/json"
	"io"
	"net/url"
)

func Search(keyword string) ([]Song, error) {

	body, err := searchPost(keyword, 1, 30)

	if err != nil {

		return nil, err

	}

	var result struct {
		Result struct {
			Songs []Song `json:"songs"`
		} `json:"result"`
	}

	if err = json.Unmarshal(body, &result); err != nil {

		return nil, err

	}

	return result.Result.Songs, nil

}

// AlbumResult 专辑搜索结果
type AlbumResult struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	PicURL  string `json:"picUrl"`
	Type    string `json:"type"`    // 专辑 / EP / Single / 精选集
	Company string `json:"company"` // 唱片公司
	Artist  struct {
		Name string `json:"name"`
	} `json:"artist"`
}

// SearchAlbum 搜专辑(type=10)
func SearchAlbum(keyword string) ([]AlbumResult, error) {

	body, err := searchPost(keyword, 10, 10)

	if err != nil {

		return nil, err

	}

	var result struct {
		Result struct {
			Albums []AlbumResult `json:"albums"`
		} `json:"result"`
	}

	if err = json.Unmarshal(body, &result); err != nil {

		return nil, err

	}

	return result.Result.Albums, nil

}

func searchPost(keyword string, typeID, limit int) ([]byte, error) {

	data := url.Values{}

	data.Set("s", keyword)

	data.Set("type", jsonInt(typeID))

	data.Set("limit", jsonInt(limit))

	resp, err := httpPostForm(
		"https://music.163.com/api/search/get",
		data,
	)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)

}
