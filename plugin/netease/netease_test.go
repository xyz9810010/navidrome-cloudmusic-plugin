package netease

import "testing"

func TestNormPunctuation(t *testing.T) {
	cases := [][2]string{
		{"下个,路口,见", "下个路口见"},
		{"下个、路口、见", "下个路口见"},
		{"晴天 (女声版))", "晴天(女声版)"},
		{"hello! world?", "helloworld"},
		{"全角！符号？", "全角符号"},
		{"２０２０", "2020"},       // 全角数字
		{"ＦＡＩＲＹ", "fairy"},    // 全角字母
		{"下個路口見", "下个路口见"},    // 繁体
		{"後來的我們", "后来的我们"},    // 繁体
		{"愛情來了", "爱情来了"},      // 繁体
	}
	for _, c := range cases {
		if norm(c[0]) != norm(c[1]) {
			t.Fatalf("归一化不等: %q vs %q (got %q / %q)", c[0], c[1], norm(c[0]), norm(c[1]))
		}
	}
}

func TestCleanKeyword(t *testing.T) {
	if got := CleanKeyword("下个,路口,见  李宇春"); got != "下个路口见 李宇春" {
		t.Fatalf("got %q", got)
	}
}

func TestMatchPunctuationVariant(t *testing.T) {
	in := MatchInput{Title: "下个,路口,见", Artist: "李宇春", Album: "李宇春"}
	songs := []Song{
		{Name: "下个路口见", Artists: []Artist{{Name: "李宇春"}}, Album: Album{Name: "李宇春"}},
	}
	best, score := Match(in, songs)
	if best == nil || best.Name != "下个路口见" || score != 120 {
		t.Fatalf("标点变体应满分命中: %+v score=%d", best, score)
	}
}

func TestMatchPrefersOriginal(t *testing.T) {
	in := MatchInput{Title: "晴天", Artist: "周杰伦", Album: "叶惠美"}
	songs := []Song{
		{Name: "晴天 (原唱 周杰伦)", Artists: []Artist{{Name: "RyaVocal"}}, Album: Album{Name: "晴天"}},
		{Name: "晴天(深情版)", Artists: []Artist{{Name: "Lucky小爱"}}, Album: Album{Name: "晴天(深情版)"}},
		{Name: "晴天", Artists: []Artist{{Name: "A-LNK"}}, Album: Album{Name: "不散"}},
		{Name: "晴天", Artists: []Artist{{Name: "周杰伦"}}, Album: Album{Name: "叶惠美"}},
	}

	best, score := Match(in, songs)
	if best == nil || best.Artists[0].Name != "周杰伦" {
		t.Fatalf("期望匹配到周杰伦原版, 实际: %+v", best)
	}
	if score != 120 {
		t.Fatalf("原版期望得分 120, 实际 %d", score)
	}
}

func TestMatchFallsBackToMarkedCover(t *testing.T) {
	in := MatchInput{Title: "晴天", Artist: "周杰伦"}
	songs := []Song{
		{Name: "晴天 (原唱 周杰伦)", Artists: []Artist{{Name: "RyaVocal"}}, Album: Album{Name: "晴天"}},
		{Name: "晴天", Artists: []Artist{{Name: "A-LNK"}}, Album: Album{Name: "不散"}},
	}

	best, _ := Match(in, songs)
	if best == nil || best.Artists[0].Name != "RyaVocal" {
		t.Fatalf("原版缺席时期望选标注原唱的翻唱, 实际: %+v", best)
	}
}

func TestMatchAlbum(t *testing.T) {
	albums := []AlbumResult{
		{ID: 1, Name: "叶惠美 (纪念版)", PicURL: "http://p1/x1.jpg"},
		{ID: 18905, Name: "叶惠美", PicURL: "http://p1/x.jpg"},
	}
	best, _ := MatchAlbum("叶惠美", "周杰伦", albums)
	if best == nil || best.ID != 18905 {
		t.Fatalf("期望精确专辑名胜出, 实际: %+v", best)
	}
}

func TestMatchArtist(t *testing.T) {
	artists := []ArtistResult{
		{ID: 1, Name: "周杰伦与朋友们"},
		{ID: 6452, Name: "周杰伦"},
	}
	best := MatchArtist("周杰伦", artists)
	if best == nil || best.ID != 6452 {
		t.Fatalf("期望精确歌手名胜出, 实际: %+v", best)
	}
}

func TestMergeTranslation(t *testing.T) {
	orig := "[00:01.00]晴天\n[00:05.00]no trans line\n[00:00.000] 作词 : 周杰伦\n"
	trans := "[00:01.00]Sunny\n"

	merged := MergeTranslation(orig, trans)
	want := "[00:01.00]晴天 Sunny\n[00:05.00]no trans line\n[00:00.000] 作词 : 周杰伦"
	if merged != want {
		t.Fatalf("合并结果不符:\n%q\nwant:\n%q", merged, want)
	}
}

func TestMergeTranslationNoTrans(t *testing.T) {
	orig := "[00:01.00]晴天"
	if got := MergeTranslation(orig, ""); got != orig {
		t.Fatalf("无翻译时原样返回, 实际 %q", got)
	}
}

func TestSplitLrcLineMetaTag(t *testing.T) {
	// 元数据标签不能当时间轴
	ts, text := splitLrcLine("[ti:晴天]")
	if ts != "" {
		t.Fatalf("元数据行应返回空时间戳, 实际 %q", ts)
	}
	_ = text
	ts, text = splitLrcLine("[00:12.340]文本")
	if ts != "[00:12.340]" || text != "文本" {
		t.Fatalf("时间轴解析错误: %q %q", ts, text)
	}
}
