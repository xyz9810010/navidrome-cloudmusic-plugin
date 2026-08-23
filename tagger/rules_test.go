package main

import "testing"

func TestDeriveTags(t *testing.T) {
	cases := []struct {
		name, album, ndType, song string
		want                      []string
	}{
		{"标注原唱的翻唱", "晴天 (女声版)", "Single", "晴天 (原唱 周杰伦)", []string{"单曲", "翻唱"}},
		{"Live", "十字路口 Live", "EP", "爱的代价 (Live)", []string{"EP", "现场"}},
		{"原声带", "仙剑奇侠传三 电视原声带", "专辑", "偏爱", []string{"原声带"}},
		{"精选集type", "滚石香港黄金十年 刘若英精选", "精选集", "后来", []string{"精选集"}},
		{"无法推断", "无特征专辑", "专辑", "普通歌名", nil},
		{"专辑默认不打标", "叶惠美", "专辑", "晴天", nil},
	}
	for _, c := range cases {
		got := DeriveTags(c.album, c.ndType, c.song)
		if !eqSlice(got, c.want) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func eqSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMergeGenre(t *testing.T) {
	if got := MergeGenre([]string{"单曲", "翻唱"}); got != "单曲;翻唱" {
		t.Fatalf("got %q", got)
	}
}

func TestShQuote(t *testing.T) {
	if got := shQuote("a'b"); got != `a'\''b` {
		t.Fatalf("got %q", got)
	}
}
