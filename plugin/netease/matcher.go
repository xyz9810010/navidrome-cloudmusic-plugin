package netease

import "strings"

// MatchInput 本地音乐元数据,用于和网易云搜索结果匹配
type MatchInput struct {
	Title  string
	Artist string
	Album  string
}

// 评分权重(与主项目 cloudmusic/matcher.go 一致)
const (
	scoreTitleExact    = 50
	scoreTitlePartial  = 20
	scoreArtistExact   = 50
	scoreArtistPartial = 25
	scoreCoverMarked   = 30
	scoreAlbumExact    = 20
	scoreAlbumPartial  = 10
)

// Keyword 拼接搜索关键词
func (in MatchInput) Keyword() string {
	return strings.TrimSpace(in.Artist + " " + in.Title)
}

// Score 计算单首候选歌曲的匹配分
func Score(in MatchInput, song Song) int {
	score := 0

	title := norm(song.Name)
	wantTitle := norm(in.Title)
	if wantTitle != "" && title == wantTitle {
		score += scoreTitleExact
	} else if wantTitle != "" && (strings.Contains(title, wantTitle) || strings.Contains(wantTitle, title)) {
		score += scoreTitlePartial
	}

	wantArtist := norm(in.Artist)
	for _, a := range song.Artists {
		name := norm(a.Name)
		if wantArtist == "" || name == "" {
			continue
		}
		if name == wantArtist {
			score += scoreArtistExact
			break
		}
		if strings.Contains(name, wantArtist) || strings.Contains(wantArtist, name) {
			score += scoreArtistPartial
			break
		}
	}

	if wantArtist != "" && strings.Contains(title, "原唱") && strings.Contains(title, wantArtist) {
		score += scoreCoverMarked
	}

	album := norm(song.Album.Name)
	wantAlbum := norm(in.Album)
	if wantAlbum != "" && album != "" {
		if album == wantAlbum {
			score += scoreAlbumExact
		} else if strings.Contains(album, wantAlbum) || strings.Contains(wantAlbum, album) {
			score += scoreAlbumPartial
		}
	}

	return score
}

// Match 返回得分最高的歌曲和分数;候选为空返回 nil
func Match(in MatchInput, songs []Song) (*Song, int) {
	var best *Song
	bestScore := 0
	for i := range songs {
		if s := Score(in, songs[i]); s > bestScore {
			bestScore = s
			best = &songs[i]
		}
	}
	return best, bestScore
}

// MatchAlbum 专辑匹配评分
func MatchAlbum(wantAlbum, wantArtist string, albums []AlbumResult) *AlbumResult {
	var best *AlbumResult
	bestScore := 0
	wa, wr := norm(wantAlbum), norm(wantArtist)
	for i := range albums {
		a := &albums[i]
		name, artist := norm(a.Name), norm(a.Artist.Name)
		score := 0
		if wa != "" && name == wa {
			score += 50
		} else if wa != "" && (strings.Contains(name, wa) || strings.Contains(wa, name)) {
			score += 20
		}
		if wr != "" && artist != "" {
			if artist == wr {
				score += 50
			} else if strings.Contains(artist, wr) || strings.Contains(wr, artist) {
				score += 25
			}
		}
		if score > bestScore {
			bestScore = score
			best = a
		}
	}
	return best
}

// MatchArtist 歌手匹配:精确名优先,其次包含关系
func MatchArtist(want string, artists []ArtistResult) *ArtistResult {
	w := norm(want)
	if w == "" {
		return nil
	}
	// 第一轮:名字完全一致
	for i := range artists {
		if norm(artists[i].Name) == w {
			return &artists[i]
		}
	}
	// 第二轮:包含关系
	for i := range artists {
		name := norm(artists[i].Name)
		if name != "" && (strings.Contains(name, w) || strings.Contains(w, name)) {
			return &artists[i]
		}
	}
	return nil
}

// norm 归一化:小写、去空格
func norm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "　", "")
	return s
}
