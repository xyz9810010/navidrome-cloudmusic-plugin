package parser

import "testing"

func TestNumberedParser(t *testing.T) {
	cases := []struct {
		file             string
		title, artist, v string
	}{
		{"0000.云与海 阿YueYue (DJ沈念版).mp3", "云与海", "阿YueYue", "DJ沈念版"},
		{"0000.可可托海的牧羊人 王琪  (DJ沈念).flac", "可可托海的牧羊人", "王琪", "DJ沈念"},
		{"0000忘川彼岸 零一九零贰（DJ名龙版）.mp3", "忘川彼岸", "零一九零贰", "DJ名龙版"},
		{"0000.一曲相思 半阳 （DJ版）.mp3", "一曲相思", "半阳", "DJ版"},
		{"0008.盛雪呼啦圈专用音乐 梁佳祺  (激情版)（DJ）.mp3", "盛雪呼啦圈专用音乐", "梁佳祺", "激情版 DJ"},
		{"0001.Fire  姜鹏（DJ）.mp3", "Fire", "姜鹏", "DJ"},
	}
	for _, c := range cases {
		r, ok := Parse(c.file)
		if !ok || r.Title != c.title || r.Artist != c.artist || r.Version != c.v {
			t.Errorf("%s\n  got  {Title:%q Artist:%q Version:%q} ok=%v\n  want {Title:%q Artist:%q Version:%q}",
				c.file, r.Title, r.Artist, r.Version, ok, c.title, c.artist, c.v)
		}
	}
}

func TestDefaultParser(t *testing.T) {
	cases := []struct {
		file             string
		title, artist, v string
	}{
		{"100 Ways - 王嘉尔.flac", "100 Ways", "王嘉尔", ""},
		{"晴天 (女声版) - GYBeat.flac", "晴天", "GYBeat", "女声版"},
	}
	for _, c := range cases {
		r, ok := Parse(c.file)
		if !ok || r.Title != c.title || r.Artist != c.artist || r.Version != c.v {
			t.Errorf("%s\n  got {Title:%q Artist:%q Version:%q} ok=%v", c.file, r.Title, r.Artist, r.Version, ok)
		}
	}
}

func TestNewFormats(t *testing.T) {
	cases := []struct {
		file             string
		title, artist, v string
	}{
		// 无空格中划线 + 副本后缀
		{"叹云兮-鞠婧祎.flac", "叹云兮", "鞠婧祎", ""},
		{"烟火里的尘埃-华晨宇 (1).flac", "烟火里的尘埃", "华晨宇", ""},
		// 编号 + 中划线
		{"2375.我有情你没爱（DJ版）-李子恒.mp3", "我有情你没爱", "李子恒", "DJ版"},
		{"0073.赤伶 （DJ版）-名龙.mp3", "赤伶", "名龙", "DJ版"},
		// 裸版本词尾巴
		{"2324.狂拽酷炫吊炸天 龙奔 DJ.mp3", "狂拽酷炫吊炸天", "龙奔", "DJ"},
		{"0007.老铁没毛病666 龙奔（DJ）.mp3", "老铁没毛病666", "龙奔", "DJ"},
	}
	for _, c := range cases {
		r, ok := Parse(c.file)
		if !ok || r.Title != c.title || r.Artist != c.artist || r.Version != c.v {
			t.Errorf("%s\n  got  {Title:%q Artist:%q Version:%q} ok=%v\n  want {Title:%q Artist:%q Version:%q}",
				c.file, r.Title, r.Artist, r.Version, ok, c.title, c.artist, c.v)
		}
	}
}

func TestEnglishArtistCandidates(t *testing.T) {
	// 纯英文末段时,两词歌手候选应排在最前
	cands, ok := ParseCandidates("0001.Love Yourself Justin Bieber.flac")
	if !ok || len(cands) < 2 {
		t.Fatalf("候选不足: %+v", cands)
	}
	if cands[0].Artist != "Justin Bieber" || cands[0].Title != "Love Yourself" {
		t.Fatalf("首选应为两词歌手: %+v", cands[0])
	}
}

func TestNotApplicable(t *testing.T) {
	if _, ok := Parse("周杰伦 - 晴天 (原唱 周杰伦) - 其他.flac"); !ok {
		// default 解析器应能处理多个 " - "(取最后一个)
		t.Fatal("多分隔符应可解析")
	}
	if _, ok := Parse("随便一个名字.flac"); ok {
		// 无编号无分隔符:numbered 不适用、default 不适用
		t.Fatal("不应解析")
	}
}
