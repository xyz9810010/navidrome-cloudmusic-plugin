package netease

import (
	"strings"
	"unicode"
)

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

// Keyword 拼接搜索关键词:曲名去序号前缀、去括号版本段(搜原曲,歌词通用)
func (in MatchInput) Keyword() string {
	return CleanKeyword(strings.TrimSpace(in.Artist + " " + searchTitle(in.Title)))
}

// stripTrackNum 去掉曲名开头的音轨序号:"0008.盛雪呼啦圈" → "盛雪呼啦圈"。
// 规则:数字串后跟分隔符,或 0 开头的三位以上编号("0000忘川彼岸"),
// 否则视为年份等正文保留("2020年的雪")。
func stripTrackNum(s string) string {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i > 4 {
		return s
	}
	rest := s[i:]
	hasSep := false
	for _, sp := range []string{".", "、", "．", "-", "_", "—", "–", " ", "　"} {
		if strings.HasPrefix(rest, sp) {
			hasSep = true
			break
		}
	}
	if !hasSep && !(s[0] == '0' && i >= 3) {
		return s
	}
	rest = strings.TrimLeft(rest, ".、．-_-—– 　")
	if rest == "" {
		return s
	}
	return rest
}

// stripTrackNums 序号全清理:开头序号 + 任意位置的零填充编号 token。
// 序号不一定在开头("盛雪呼啦圈专用音乐 0008 梁佳祺"),年份(2020)不以0开头不受影响
func stripTrackNums(s string) string {
	s = stripTrackNum(s)
	out := make([]string, 0, 8)
	for _, f := range strings.Fields(s) {
		t := strings.Trim(f, ".,、．-_—")
		if (len(t) == 3 || len(t) == 4) && t[0] == '0' {
			allDigit := true
			for i := 0; i < len(t); i++ {
				if t[i] < '0' || t[i] > '9' {
					allDigit = false
					break
				}
			}
			if allDigit {
				continue // 零填充序号,剔除
			}
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}

// stripBrackets 去掉括号段(中英文括号/【】/{}),保留其余文本
func stripBrackets(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(', '（', '[', '【', '〔', '{':
			depth++
		case ')', '）', ']', '】', '〕', '}':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// stripVersionWords 去版本修饰词
func stripVersionWords(s string) string {
	for _, w := range versionWords {
		s = strings.ReplaceAll(s, w, " ")
	}
	return s
}

// searchTitle 搜索用曲名:去序号(含中部)、去括号段、去版本词(保留空格)
func searchTitle(s string) string {
	return stripVersionWords(stripBrackets(stripTrackNums(s)))
}

// versionWords 版本修饰词,核心名比较时剔除(长词在前防止部分替换)
var versionWords = []string{
	"DJ沈念版", "DJ名龙版", "DJheap九天版", "DJ版", "DJ",
	"激情版", "温柔版", "深情版", "女声版", "男声版", "童声版",
	"加速版", "降速版", "完整版", "超燃版", "新版", "重制版",
	"Live", "live", "Remix", "remix", "RMX", "Instrumental",
	"伴奏", "纯音乐", "翻唱", "抖音版", "快手版",
}

// coreTitle 核心曲名:去序号、去括号段、去版本词、归一化。
// "0008.盛雪呼啦圈专用音乐 梁佳祺 (激情版)（DJ）" → "盛雪呼啦圈专用音乐梁佳祺"
func coreTitle(s string) string {
	return norm(searchTitle(s))
}

// Score 计算单首候选歌曲的匹配分
func Score(in MatchInput, song Song) int {
	score := 0

	title := norm(stripTrackNums(song.Name))
	wantTitle := norm(stripTrackNums(in.Title))
	if wantTitle != "" && title == wantTitle {
		score += scoreTitleExact
	} else if wantTitle != "" && (strings.Contains(title, wantTitle) || strings.Contains(wantTitle, title)) {
		score += scoreTitlePartial
	} else {
		// 核心名二次比较:序号/版本段干扰时仍能命中
		ct, cw := coreTitle(song.Name), coreTitle(in.Title)
		if ct != "" && ct == cw {
			score += scoreTitleExact
		} else if ct != "" && cw != "" && (strings.Contains(ct, cw) || strings.Contains(cw, ct)) {
			score += scoreTitlePartial
		}
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

// MatchAlbum 专辑匹配评分,返回最佳结果与分数(≥50 视为可信)
func MatchAlbum(wantAlbum, wantArtist string, albums []AlbumResult) (*AlbumResult, int) {
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
	return best, bestScore
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

// trad2simp 常用繁→简映射(歌名常用字)。两条包保持同步:
// plugin/netease/matcher.go 与 cloudmusic/matcher.go
var trad2simp = map[rune]rune{
	'愛': '爱', '麼': '么', '們': '们', '說': '说', '話': '话', '誰': '谁',
	'見': '见', '時': '时', '間': '间', '聽': '听', '聲': '声', '風': '风',
	'雲': '云', '飛': '飞', '無': '无', '後': '后', '來': '来', '個': '个',
	'裡': '里', '裏': '里', '為': '为', '與': '与', '從': '从', '還': '还',
	'沒': '没', '記': '记', '憶': '忆', '舊': '旧', '華': '华', '單': '单',
	'雙': '双', '萬': '万', '藍': '蓝', '綠': '绿', '紅': '红', '黃': '黄',
	'銀': '银', '鐵': '铁', '島': '岛', '嶼': '屿', '橋': '桥', '鄉': '乡',
	'國': '国', '園': '园', '書': '书', '畫': '画', '劍': '剑', '馬': '马',
	'車': '车', '樂': '乐', '讀': '读', '寫': '写', '戲': '戏', '劇': '剧',
	'電': '电', '視': '视', '機': '机', '網': '网', '線': '线', '絲': '丝',
	'結': '结', '給': '给', '絕': '绝', '續': '续', '總': '总', '簡': '简',
	'藝': '艺', '麗': '丽', '歸': '归', '當': '当', '發': '发', '變': '变',
	'親': '亲', '覺': '觉', '觀': '观', '歡': '欢', '憂': '忧', '懷': '怀',
	'應': '应', '戀': '恋', '夢': '梦', '淚': '泪', '陽': '阳', '陰': '阴',
	'霧': '雾', '煙': '烟', '塵': '尘', '熱': '热', '涼': '凉', '凍': '冻',
	'淨': '净', '潔': '洁', '滅': '灭', '燈': '灯', '燒': '烧', '點': '点',
	'長': '长', '門': '门', '問': '问', '聞': '闻', '開': '开', '閒': '闲',
	'關': '关', '隊': '队', '陣': '阵', '階': '阶', '隨': '随', '際': '际',
	'難': '难', '雞': '鸡', '離': '离', '頭': '头', '顆': '颗', '項': '项',
	'順': '顺', '須': '须', '顧': '顾', '頓': '顿', '頗': '颇', '頌': '颂',
	'預': '预', '領': '领', '頻': '频', '題': '题', '顏': '颜', '額': '额',
	'飄': '飘', '餘': '余', '養': '养', '館': '馆', '驚': '惊', '驕': '骄',
	'體': '体', '魚': '鱼', '鳥': '鸟', '鳴': '鸣', '龍': '龙', '龜': '龟',
	'貓': '猫', '麵': '面', '廟': '庙', '賣': '卖', '買': '买', '貝': '贝',
	'貧': '贫', '賞': '赏', '賜': '赐', '賠': '赔', '賢': '贤', '贈': '赠',
	'趕': '赶', '趙': '赵', '蹤': '踪', '軌': '轨', '軍': '军', '輕': '轻',
	'輪': '轮', '載': '载', '較': '较', '輯': '辑', '輸': '输', '辦': '办',
	'辭': '辞', '邊': '边', '這': '这', '進': '进', '遠': '远', '適': '适',
	'選': '选', '遺': '遗', '遼': '辽', '遞': '递', '釋': '释', '鐘': '钟',
	'鎖': '锁', '鏡': '镜', '鑽': '钻', '閃': '闪', '閉': '闭', '閱': '阅',
	'悶': '闷', '獨': '独', '遲': '迟', '遜': '逊', '遙': '遥', '燦': '灿',
	'爛': '烂', '燭': '烛', '歲': '岁', '歷': '历', '靜': '静', '頂': '顶',
}

// norm 归一化:全角→半角、繁→简、小写,只保留字母/数字(汉字属于字母)。
// 标点、空格全部去掉:"下个,路口,见" == "下个路口见" == "下個路口見"
func norm(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0xFF01 && r <= 0xFF5E {
			r -= 0xFEE0 // 全角 ASCII → 半角
		}
		if m, ok := trad2simp[r]; ok {
			r = m
		}
		r = unicode.ToLower(r)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SearchTitle 导出:曲名清洗(去序号/括号版本段/版本词),供 agent 做"曲名内拆歌手"
func SearchTitle(s string) string { return searchTitle(s) }

// CleanKeyword 清洗搜索关键词:全角转半角、繁转简、去标点、保留空格分隔
func CleanKeyword(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0xFF01 && r <= 0xFF5E {
			r -= 0xFEE0
		}
		if m, ok := trad2simp[r]; ok {
			r = m
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
