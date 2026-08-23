package cloudmusic

import "strings"

// MatchInput 本地音乐元数据,用于和网易云搜索结果匹配
type MatchInput struct {
	Title  string // 歌曲名,如 "晴天"
	Artist string // 歌手名,如 "周杰伦"
	Album  string // 专辑名,可为空,如 "叶惠美"
}

// 评分权重
const (
	scoreTitleExact    = 50 // 歌曲名完全一致
	scoreTitlePartial  = 20 // 歌曲名包含关系
	scoreArtistExact   = 50 // 歌手完全一致
	scoreArtistPartial = 25 // 歌手包含关系(如 feat / 组合)
	scoreCoverMarked   = 30 // 翻唱且标题标注了原唱,如 "晴天 (原唱 周杰伦)"
	scoreAlbumExact    = 20 // 专辑完全一致
	scoreAlbumPartial  = 10 // 专辑包含关系
)

// Keyword 拼接搜索关键词
func (in MatchInput) Keyword() string {
	return strings.TrimSpace(in.Artist + " " + in.Title)
}

// Score 计算单首候选歌曲的匹配分
func Score(in MatchInput, song Song) int {
	score := 0

	// 歌曲名
	title := norm(song.Name)
	wantTitle := norm(in.Title)
	if wantTitle != "" && title == wantTitle {
		score += scoreTitleExact
	} else if wantTitle != "" && (strings.Contains(title, wantTitle) || strings.Contains(wantTitle, title)) {
		score += scoreTitlePartial
	}

	// 歌手(任一位歌手命中即可)
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

	// 翻唱但标注了原唱:标题含 "原唱" 且包含原唱歌手名
	if wantArtist != "" && strings.Contains(title, "原唱") && strings.Contains(title, wantArtist) {
		score += scoreCoverMarked
	}

	// 专辑
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

// SearchAndMatch 搜索 + 匹配一步完成,返回最佳结果
func SearchAndMatch(in MatchInput) (*Song, int, error) {
	songs, err := Search(in.Keyword())
	if err != nil {
		return nil, 0, err
	}
	song, score := Match(in, songs)
	return song, score, nil
}

// norm 归一化:小写、去空格
func norm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "　", "") // 全角空格
	return s
}
