package cloudmusic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Lyric 拉取 LRC 格式歌词,无歌词时返回空字符串
func Lyric(id int) (string, error) {
	url := fmt.Sprintf("https://music.163.com/api/song/lyric?id=%d&lv=1&kv=1&tv=-1", id)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Lrc struct {
			Version int    `json:"version"`
			Lyric   string `json:"lyric"`
		} `json:"lrc"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Lrc.Lyric == "" {
		return "", fmt.Errorf("no lyric: id=%d", id)
	}

	return result.Lrc.Lyric, nil
}
