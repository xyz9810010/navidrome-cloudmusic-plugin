package netease

// Song 搜索结果歌曲
type Song struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Artists []Artist `json:"artists"`
	Album   Album   `json:"album"`
}

// Artist 歌手
type Artist struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	PicURL   string `json:"picUrl"`
	Img1v1   string `json:"img1v1Url"`
	Brief    string `json:"briefDesc"`
}

// Album 专辑
type Album struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	PicURL string `json:"picUrl"`
	// 搜索接口偶尔只给 picId,pic_str 为字符串形态兜底
	PicURL2 string `json:"pic_str"`
}

// ArtistNames 返回全部歌手名,用 " / " 连接
func (s Song) ArtistNames() string {
	names := make([]string, 0, len(s.Artists))
	for _, a := range s.Artists {
		names = append(names, a.Name)
	}
	return join(names, " / ")
}

func join(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
