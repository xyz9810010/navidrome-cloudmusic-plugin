// Package parser 音乐文件名解析:从特殊命名的文件名拆出 歌名/歌手/版本。
//
//	0000.云与海 阿YueYue (DJ沈念版).mp3  →  {云与海, 阿YueYue, DJ沈念版}
//	100 Ways - 王嘉尔.flac               →  {100 Ways, 王嘉尔, ""}
//
// 解析器链可自由组合(见 parser.go 的 Chain),遇新格式新增 Parser 实现即可。
package parser

import (
	"strings"
	"unicode/utf8"
)

// Result 解析结果
type Result struct {
	Title   string // 歌名(不含版本段)
	Artist  string // 歌手
	Version string // 版本段,如 "DJ沈念版";无则为空
	Parser  string // 使用的解析器名
	Score   int    // 候选启发分(100=结构确定;拆分候选按启发排序,可由验证方回填覆盖)
}

// Parser 文件名解析器
type Parser interface {
	Name() string
	// Parse 解析文件名(带扩展名);ok=false 表示该解析器不适用
	Parse(filename string) (Result, bool)
	// Candidates 多候选解析(从最可能到最不可能),供外部用网易云等验证方评分选优。
	// 结构确定的解析器返回单元素切片即可。
	Candidates(filename string) ([]Result, bool)
}

// stripExt 去扩展名
func stripExt(filename string) string {
	if i := strings.LastIndex(filename, "."); i > 0 {
		return filename[:i]
	}
	return filename
}

// stripLeadingNum 去掉开头的编号:"0000." / "0000忘川"(0开头编号)。
// 纯数字开头+空格(如 "100 Ways")不算编号,避免误伤英文歌名。
func stripLeadingNum(s string) (string, bool) {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i > 4 {
		return s, false
	}
	rest := s[i:]
	hasDot := strings.HasPrefix(rest, ".") || strings.HasPrefix(rest, "、") || strings.HasPrefix(rest, "．")
	zeroPadded := s[0] == '0' && i >= 3
	if !hasDot && !zeroPadded {
		return s, false
	}
	rest = strings.TrimLeft(rest, ".、．-— 　")
	return rest, true
}

// takeTrailingVersion 摘出结尾的括号版本段(可多层混排):
//
//	"云与海 阿YueYue (DJ沈念版)"          → ("云与海 阿YueYue", "DJ沈念版")
//	"盛雪呼啦圈 梁佳祺 (激情版)（DJ）"     → ("盛雪呼啦圈 梁佳祺", "激情版 DJ")
func takeTrailingVersion(s string) (string, string) {
	var parts []string
	for {
		rest, v := takeOneTrailingBracket(s)
		if rest == s {
			break
		}
		parts = append([]string{v}, parts...)
		s = rest
	}
	return s, strings.Join(parts, " ")
}

// stripCopySuffix 剥掉重复下载产生的尾部副本后缀:"华晨宇 (1)" → "华晨宇"
func stripCopySuffix(s string) string {
	s = strings.TrimSpace(s)
	for {
		r, _ := utf8.DecodeLastRuneInString(s)
		if r != ')' && r != '）' {
			return s
		}
		idx := strings.LastIndexAny(s, "(（")
		if idx <= 0 {
			return s
		}
		inner := strings.Trim(s[idx:], "()（） ")
		if inner == "" || !isASCIIDigits(inner) {
			return s
		}
		s = strings.TrimSpace(s[:idx])
	}
}

func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// versionWords 版本修饰词(裸词形态,与 cloudmusic/matcher.go 保持同步)
var versionWords = []string{
	"DJ沈念版", "DJ名龙版", "DJheap九天版", "DJ版", "DJ",
	"激情版", "温柔版", "深情版", "女声版", "男声版", "童声版",
	"加速版", "降速版", "完整版", "超燃版", "新版", "重制版",
	"Live", "live", "Remix", "remix", "RMX", "Instrumental",
	"伴奏", "纯音乐", "翻唱", "抖音版", "快手版",
}

// stripVersionWords 去裸版本词,并返回被剔除的词(收进 Version)
func stripVersionWords(s string) (string, string) {
	var removed []string
	for _, w := range versionWords {
		if strings.Contains(s, w) {
			removed = append(removed, w)
			s = strings.ReplaceAll(s, w, " ")
		}
	}
	return s, strings.Join(removed, " ")
}

// takeOneTrailingBracket 剥一层结尾括号段;没有则原样返回
func takeOneTrailingBracket(s string) (string, string) {
	s = strings.TrimSpace(s)
	if len(s) < 3 {
		return s, ""
	}
	last, _ := utf8.DecodeLastRuneInString(s)
	var open, close_ string
	switch last {
	case ')':
		open, close_ = "(", ")"
	case '）':
		open, close_ = "（", "）"
	default:
		return s, ""
	}
	idx := strings.LastIndex(s, open)
	if idx <= 0 {
		return s, ""
	}
	version := strings.Trim(s[idx:], open+close_+"()（） ")
	rest := strings.TrimSpace(s[:idx])
	if rest == "" || version == "" {
		return s, ""
	}
	return rest, version
}
