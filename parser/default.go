package parser

import "strings"

// DefaultParser 处理 "歌名 - 歌手" / "歌名-歌手" / "歌名 (版本) - 歌手" 格式:
//
//	100 Ways - 王嘉尔.flac
//	晴天 (女声版) - GYBeat.flac
//	叹云兮-鞠婧祎.flac                ← 无空格中划线
//	烟火里的尘埃-华晨宇 (1).flac       ← 重复下载副本后缀
type DefaultParser struct{}

func (DefaultParser) Name() string { return "default" }

func (p DefaultParser) Parse(filename string) (Result, bool) {
	name := strings.TrimSpace(stripExt(filename))
	name = stripCopySuffix(name)
	// 取最后一个中划线(歌名里可能自带连字符,歌手在最后)
	idx := strings.LastIndex(name, "-")
	if idx <= 0 || idx == len(name)-1 {
		return Result{}, false
	}
	title := strings.TrimSpace(name[:idx])
	artist := strings.TrimSpace(name[idx+1:])
	if title == "" || artist == "" || strings.Contains(artist, "-") {
		// artist 里还有中划线说明是 "A-B-C" 多段乱名,不猜
		return Result{}, false
	}
	title, version := takeTrailingVersion(title)
	artist = stripCopySuffix(artist)
	return Result{Title: title, Artist: artist, Version: version, Parser: p.Name()}, true
}

func (p DefaultParser) Candidates(filename string) ([]Result, bool) {
	r, ok := p.Parse(filename)
	if !ok {
		return nil, false
	}
	return []Result{r}, true
}
