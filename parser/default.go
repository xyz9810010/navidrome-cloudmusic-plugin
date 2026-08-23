package parser

import "strings"

// DefaultParser 处理常见 "歌名 - 歌手" / "歌名 (版本) - 歌手" 格式:
//
//	100 Ways - 王嘉尔.flac
//	晴天 (女声版) - GYBeat.flac
type DefaultParser struct{}

func (DefaultParser) Name() string { return "default" }

func (p DefaultParser) Parse(filename string) (Result, bool) {
	name := strings.TrimSpace(stripExt(filename))
	// 取最后一个 " - "(歌名里可能自带连字符)
	idx := strings.LastIndex(name, " - ")
	if idx <= 0 || idx == len(name)-3 {
		return Result{}, false
	}
	title := strings.TrimSpace(name[:idx])
	artist := strings.TrimSpace(name[idx+3:])
	if title == "" || artist == "" {
		return Result{}, false
	}
	title, version := takeTrailingVersion(title)
	return Result{Title: title, Artist: artist, Version: version, Parser: p.Name()}, true
}

func (p DefaultParser) Candidates(filename string) ([]Result, bool) {
	r, ok := p.Parse(filename)
	if !ok {
		return nil, false
	}
	return []Result{r}, true
}
