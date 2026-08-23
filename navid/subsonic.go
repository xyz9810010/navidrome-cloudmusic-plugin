// Package navid 封装 Navidrome 的 Subsonic API 访问(tagger / match 共用)。
package navid

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client Subsonic API 客户端(token 认证)
type Client struct {
	Base   string
	User   string
	Salt   string
	Token  string
	HTTP   *http.Client
}

func New(base, user, password string) *Client {
	salt := make([]byte, 6)
	_, _ = rand.Read(salt)
	s := hex.EncodeToString(salt)
	sum := md5.Sum([]byte(password + s))
	return &Client{
		Base:   strings.TrimRight(base, "/"),
		User:   user,
		Salt:   s,
		Token:  hex.EncodeToString(sum[:]),
		HTTP:   http.DefaultClient,
	}
}

func (c *Client) get(endpoint string, params map[string]string, out any) error {
	v := url.Values{}
	v.Set("u", c.User)
	v.Set("t", c.Token)
	v.Set("s", c.Salt)
	v.Set("v", "1.16.1")
	v.Set("c", "navid-client")
	v.Set("f", "json")
	for k, val := range params {
		v.Set(k, val)
	}
	resp, err := c.HTTP.Get(c.Base + "/rest/" + endpoint + "?" + v.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var envelope struct {
		SubsonicResponse struct {
			Status string `json:"status"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if envelope.SubsonicResponse.Status != "ok" {
		return fmt.Errorf("API错误(%s): %s", endpoint, string(body[:min(len(body), 200)]))
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

type Album struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Genre  string `json:"genre"`
}

type Song struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Path   string `json:"path"`
	Suffix string `json:"suffix"`
}

// SearchSongs 全库搜索歌曲
func (c *Client) SearchSongs(query string, count int) ([]Song, error) {
	var out struct {
		SubsonicResponse struct {
			SearchResult3 struct {
				Song []Song `json:"song"`
			} `json:"searchResult3"`
		} `json:"subsonic-response"`
	}
	if err := c.get("search3.view", map[string]string{
		"query": query, "songCount": fmt.Sprint(count), "albumCount": "0", "artistCount": "0",
	}, &out); err != nil {
		return nil, err
	}
	return out.SubsonicResponse.SearchResult3.Song, nil
}

// StartScan 触发全量重扫
func (c *Client) StartScan() error {
	return c.get("startScan.view", nil, nil)
}

// GetAlbums 分页拉专辑(alphabeticalByName)
func (c *Client) GetAlbums(offset, size int) ([]Album, error) {
	var out struct {
		SubsonicResponse struct {
			AlbumList2 struct {
				Album []Album `json:"album"`
			} `json:"albumList2"`
		} `json:"subsonic-response"`
	}
	if err := c.get("getAlbumList2.view", map[string]string{
		"type": "alphabeticalByName", "size": fmt.Sprint(size), "offset": fmt.Sprint(offset),
	}, &out); err != nil {
		return nil, err
	}
	return out.SubsonicResponse.AlbumList2.Album, nil
}

// GetAlbumSongs 专辑内曲目(含音乐库相对路径)
func (c *Client) GetAlbumSongs(albumID string) ([]Song, error) {
	var out struct {
		SubsonicResponse struct {
			Album struct {
				Songs []Song `json:"song"`
			} `json:"album"`
		} `json:"subsonic-response"`
	}
	if err := c.get("getAlbum.view", map[string]string{"id": albumID}, &out); err != nil {
		return nil, err
	}
	return out.SubsonicResponse.Album.Songs, nil
}
