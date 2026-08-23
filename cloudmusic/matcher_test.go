package cloudmusic

import "testing"

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
		t.Fatalf("原版期望得分 120 (歌名50+歌手50+专辑20), 实际 %d", score)
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

func TestCoverURLFallback(t *testing.T) {
	s := Song{Album: Album{Name: "叶惠美"}}
	if got := s.CoverURL(); got != "" {
		t.Fatalf("两个字段都为空时期望空串, 实际 %q", got)
	}

	s.Album.PicURL2 = "109951165566379710"
	if got := s.CoverURL(); got != "109951165566379710" {
		t.Fatalf("picUrl 为空时应用 pic_str, 实际 %q", got)
	}

	s.Album.PicURL = "https://p1.music.126.net/x/1.jpg"
	if got := s.CoverURL(); got != "https://p1.music.126.net/x/1.jpg" {
		t.Fatalf("picUrl 优先, 实际 %q", got)
	}
}
