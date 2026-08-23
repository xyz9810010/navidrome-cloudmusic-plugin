package main

import "strings"

// 标签推断规则:从网易云专辑 type、匹配到的网易云曲名、本地专辑名推断流派标签。
// 网易云没有匿名流派接口,这套规则覆盖华语曲库最有价值的维度。

var tagRules = []struct {
	keyword  string   // 小写包含匹配(曲名/专辑名)
	tag      string   // 推断出的标签
	kind     int      // 0=曲名 1=专辑名 2=两者
}{
	{"live", "现场", 0}, {"现场", "现场", 2}, {"現場", "现场", 2}, {"演唱會", "现场", 1},
	{"原唱", "翻唱", 0}, {"翻唱", "翻唱", 2}, {"女声版", "翻唱", 2}, {"男声版", "翻唱", 2},
	{"深情版", "翻唱", 0}, {"cover", "翻唱", 0},
	{"dj", "DJ", 0}, {"remix", "Remix", 0}, {"混音", "Remix", 0},
	{"伴奏", "伴奏", 0}, {"纯音乐", "轻音乐", 0}, {"轻音乐", "轻音乐", 2}, {"钢琴版", "钢琴", 0},
	{"钢琴曲", "钢琴", 2}, {"吉他版", "吉他", 0},
	{"原声带", "原声带", 1}, {"原聲帶", "原声带", 1}, {"ost", "原声带", 1}, {"主题曲", "原声带", 1},
	{"片尾曲", "原声带", 1}, {"电影音乐", "原声带", 1}, {"電影原聲", "原声带", 1},
	{"游戏音乐", "原声带", 1},
	{"精选", "精选集", 1}, {"精選", "精选集", 1}, {"合辑", "合辑", 1}, {"單曲集", "单曲集", 1},
	{"单曲集", "单曲集", 1},
	{"国风", "国风", 2}, {"古风", "古风", 2}, {"民谣", "民谣", 2}, {"摇滚", "摇滚", 2},
	{"说唱", "说唱", 2}, {"rap", "说唱", 0}, {"电音", "电子", 2}, {"电子", "电子", 2},
	{"爵士", "爵士", 2}, {"古典", "古典", 2},
}

// ndTypeTag 网易云专辑 type 字段映射;"专辑"是默认形态,不打标签
var ndTypeTag = map[string]string{
	"EP":        "EP",
	"Single":    "单曲",
	"精选集":      "精选集",
	"EP/Single": "EP",
	"Demo":      "Demo",
}

// DeriveTags 推断标签。matchedSongName 为本地歌曲在网易云匹配到的曲名(可为空)。
func DeriveTags(albumName, ndAlbumType, matchedSongName string) []string {
	var tags []string
	seen := map[string]bool{}
	add := func(t string) {
		if t != "" && !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}

	if t, ok := ndTypeTag[ndAlbumType]; ok {
		add(t)
	}

	lowerSong := strings.ToLower(matchedSongName)
	lowerAlbum := strings.ToLower(albumName)
	for _, r := range tagRules {
		if r.kind == 0 && strings.Contains(lowerSong, r.keyword) {
			add(r.tag)
		} else if r.kind == 1 && strings.Contains(lowerAlbum, r.keyword) {
			add(r.tag)
		} else if r.kind == 2 && (strings.Contains(lowerSong, r.keyword) || strings.Contains(lowerAlbum, r.keyword)) {
			add(r.tag)
		}
		if len(tags) >= 4 {
			break
		}
	}
	return tags
}

// MergeGenre 把标签列表拼成 GENRE 字段值(Navidrome 默认以 ; 分隔多流派)
func MergeGenre(tags []string) string {
	return strings.Join(tags, ";")
}
