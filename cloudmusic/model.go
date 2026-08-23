package cloudmusic

import "strings"

type Song struct {
	ID int `json:"id"`

	Name string `json:"name"`

	Artists []Artist `json:"artists"`

	Album Album `json:"album"`
}

type Artist struct {
	ID int `json:"id"`

	Name string `json:"name"`
}

type Album struct {
	ID int `json:"id"`

	Name string `json:"name"`

	PicURL string `json:"picUrl"`

	PicURL2 string `json:"pic_str"`
}

// ArtistNames 返回全部歌手名,用 " / " 连接
func (s Song) ArtistNames() string {
	names := make([]string, 0, len(s.Artists))
	for _, a := range s.Artists {
		names = append(names, a.Name)
	}
	return strings.Join(names, " / ")
}
