package parser

import "strings"

// NumberedParser 处理 "编号." 开头的格式:
//
//	0000.云与海 阿YueYue (DJ沈念版)      ← 歌名 歌手 (版本)
//	0000.可可托海的牧羊人 王琪  (DJ沈念)  ← 多空格
//	0000忘川彼岸 零一九零贰（DJ名龙版）    ← 无分隔符编号
//	2375.我有情你没爱（DJ版）-李子恒      ← 歌名(版本)-歌手
//	2324.狂拽酷炫吊炸天 龙奔 DJ           ← 裸版本词尾巴
//
// 拆分策略:优先中划线分割;否则从右向左生成歌手候选(中文取末段,英文歌手名
// 可能多词),最终由验证方(网易云匹配)选优。
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
	name = stripCopySuffix(name)
	rest, version := takeTrailingVersion(name)

	// 形态A: "歌名(版本)-歌手" —— 中划线分割,歌名里可能还带版本括号段
	if idx := strings.LastIndex(rest, "-"); idx > 0 && idx < len(rest)-1 {
		title := strings.TrimSpace(rest[:idx])
		artist := stripCopySuffix(strings.TrimSpace(rest[idx+1:]))
		tTitle, tVer := takeTrailingVersion(title)
		if tTitle != "" && artist != "" && !strings.Contains(artist, "-") {
			return []Result{{
				Title:   tTitle,
				Artist:  artist,
				Version: joinVersion(tVer, version),
				Parser:  p.Name(),
			}}, true
		}
	}

	// 形态B: "歌名 歌手 [DJ]" —— 裸版本词剔除后按空格分词,裸词并入版本
	cleaned, bareVer := stripVersionWords(rest)
	tokens := strings.Fields(cleaned)
	version = joinVersion(version, bareVer)
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
		// 末段纯英文且还有余量:歌手可能是两词(Justin Bieber)
		add(2)
		add(1)
	} else {
		add(1)
		if len(tokens) >= 3 {
			add(2)
		}
	}
	return out, true
}

// joinVersion 合并两段版本描述
func joinVersion(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + " " + b
	}
}

// isASCII 纯ASCII(英文歌手名判断)
func isASCII(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}
