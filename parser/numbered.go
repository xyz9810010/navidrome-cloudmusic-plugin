package parser

import "strings"

// NumberedParser 处理 "编号.歌名 歌手 (版本)" 格式:
//
//	0000.云与海 阿YueYue (DJ沈念版)
//	0000.可可托海的牧羊人 王琪  (DJ沈念)   ← 多空格
//	0000忘川彼岸 零一九零贰（DJ名龙版）      ← 无分隔符编号
//
// 拆分策略(按用户设计):从右向左生成歌手候选,中文取末段;纯英文歌手名
// 可能多词(Justin Bieber),额外给两词候选;最终由验证方(网易云匹配)选优。
type NumberedParser struct{}

func (NumberedParser) Name() string { return "numbered" }

func (p NumberedParser) Parse(filename string) (Result, bool) {
	cands, ok := p.Candidates(filename)
	if !ok || len(cands) == 0 {
		return Result{}, false
	}
	r := cands[0]
	r.Parser = p.Name()
	return r, true
}

func (p NumberedParser) Candidates(filename string) ([]Result, bool) {
	name, numbered := stripLeadingNum(stripExt(filename))
	if !numbered {
		return nil, false
	}
	rest, version := takeTrailingVersion(name)
	tokens := strings.Fields(rest)
	if len(tokens) < 2 {
		return nil, false // 只有歌名没有歌手段,拆不了
	}

	var out []Result
	add := func(artistTokens int) {
		split := len(tokens) - artistTokens
		if split <= 0 {
			return
		}
		out = append(out, Result{
			Title:   strings.Join(tokens[:split], " "),
			Artist:  strings.Join(tokens[split:], " "),
			Version: version,
			Parser:  p.Name(),
		})
	}

	last := tokens[len(tokens)-1]
	if isASCII(last) && len(tokens) >= 3 {
		// 末段是纯英文且还有余量:歌手可能是两词(Justin Bieber)
		add(2)
		add(1)
	} else {
		add(1)
		if len(tokens) >= 3 {
			add(2) // 备选:歌手两词
		}
	}
	return out, true
}

// isASCII 纯ASCII(英文歌手名判断)
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return len(s) > 0
}
