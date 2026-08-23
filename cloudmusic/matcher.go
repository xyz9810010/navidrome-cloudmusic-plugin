package cloudmusic

import (
	"strings"
	"unicode"
)

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

// trad2simp 常用繁→简映射(歌名常用字)。与 plugin/netease/matcher.go 保持同步
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

// Norm 导出归一化(供 tagger 等复用)
func Norm(s string) string { return norm(s) }

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
