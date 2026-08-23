package parser

// Chain 解析器链:按序尝试,第一个适用者生效。
// 可自由调整顺序或删减实现开关,如只保留 numbered:
//	parser.Chain = []parser.Parser{parser.NumberedParser{}}
var Chain = []Parser{
	NumberedParser{}, // "0000.歌名 歌手 (DJ版)" 系
	DefaultParser{},  // "歌名 - 歌手" 系
}

// Parse 自动选择解析器解析文件名
func Parse(filename string) (Result, bool) {
	for _, p := range Chain {
		if r, ok := p.Parse(filename); ok {
			r.Parser = p.Name()
			return r, true
		}
	}
	return Result{}, false
}

// ParseCandidates 返回全部候选(跨解析器),供验证方评分选优
func ParseCandidates(filename string) ([]Result, bool) {
	var all []Result
	for _, p := range Chain {
		if cands, ok := p.Candidates(filename); ok {
			for _, c := range cands {
				c.Parser = p.Name()
				all = append(all, c)
			}
			if len(all) > 0 {
				return all, true // 第一个适用的解析器的候选集合
			}
		}
	}
	return nil, false
}
