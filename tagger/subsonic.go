package main

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

// SubsonicClient Navidrome Subsonic API 客户端(token 认证)
type SubsonicClient struct {
	Base   string
	User   string
	Salt   string
	Token  string
	Client *http.Client
}

func NewSubsonicClient(base, user, password string) *SubsonicClient {
	salt := make([]byte, 6)
	_, _ = rand.Read(salt)
	s := hex.EncodeToString(salt)
	sum := md5.Sum([]byte(password + s))
	return &SubsonicClient{
		Base:   strings.TrimRight(base, "/"),
		User:   user,
		Salt:   s,
		Token:  hex.EncodeToString(sum[:]),
		Client: http.DefaultClient,
	}
}

func (c *SubsonicClient) get(endpoint string, params map[string]string, out any) error {
	v := url.Values{}
	v.Set("u", c.User)
	v.Set("t", c.Token)
	v.Set("s", c.Salt)
	v.Set("v", "1.16.1")
	v.Set("c", "tagger")
	v.Set("f", "json")
	for k, val := range params {
		v.Set(k, val)
	}
	resp, err := c.Client.Get(c.Base + "/rest/" + endpoint + "?" + v.Encode())
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
	Path   string `json:"path"`
	Suffix string `json:"suffix"`
}

// GetAlbums 分页拉专辑(alphabeticalByName)
func (c *SubsonicClient) GetAlbums(offset, size int) ([]Album, error) {
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
func (c *SubsonicClient) GetAlbumSongs(albumID string) ([]Song, error) {
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

// GetGenres 流派 -> 专辑数(验证用)
func (c *SubsonicClient) GetGenres() (map[string]int, error) {
	var out struct {
		SubsonicResponse struct {
			Genres struct {
				Genre []struct {
					Name       string `json:"value"`
					AlbumCount int    `json:"albumCount"`
				} `json:"genre"`
			} `json:"genres"`
		} `json:"subsonic-response"`
	}
	if err := c.get("getGenres.view", nil, &out); err != nil {
		return nil, err
	}
	m := map[string]int{}
	for _, g := range out.SubsonicResponse.Genres.Genre {
		m[g.Name] = g.AlbumCount
	}
	return m, nil
}
