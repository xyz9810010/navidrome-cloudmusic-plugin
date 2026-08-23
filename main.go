package main

import (
	"fmt"
	"strings"

	"navidrome-cloudmusic-plugin/cloudmusic"
)

func main() {
	// 本地音乐元数据
	input := cloudmusic.MatchInput{
		Title:  "晴天",
		Artist: "周杰伦",
		Album:  "叶惠美",
	}

	fmt.Println("搜索关键词:", input.Keyword())
	fmt.Println("================")

	// 搜索 + 智能匹配
	song, score, err := cloudmusic.SearchAndMatch(input)
	if err != nil {
		panic(err)
	}
	if song == nil {
		fmt.Println("没有搜索结果")
		return
	}

	fmt.Println("歌曲:", song.Name)
	fmt.Println("歌手:", song.ArtistNames())
	fmt.Println("专辑:", song.Album.Name)
	fmt.Println("匹配得分:", score)
	fmt.Println("网易云ID:", song.ID)

	// 封面:搜索接口不给 URL,用 song/detail 补全
	detail, err := cloudmusic.SongDetail(song.ID)
	if err != nil {
		fmt.Println("封面: 获取失败,", err)
	} else {
		fmt.Println("封面:", detail.CoverURL())
	}

	// 歌词
	lyric, err := cloudmusic.Lyric(song.ID)
	if err != nil {
		fmt.Println("歌词: 获取失败,", err)
	} else {
		lines := strings.Split(lyric, "\n")
		if len(lines) > 5 {
			lines = lines[:5]
		}
		fmt.Println("歌词预览:")
		for _, line := range lines {
			fmt.Println("  ", line)
		}
	}
}
