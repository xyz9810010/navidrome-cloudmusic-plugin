package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"cloudmusic-plugin/netease"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lyrics"
	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

// agent 实现 Navidrome 的元数据与歌词插件接口
type agent struct{}

var (
	_ metadata.ArtistURLProvider       = (*agent)(nil)
	_ metadata.ArtistBiographyProvider = (*agent)(nil)
	_ metadata.ArtistImagesProvider    = (*agent)(nil)
	_ metadata.AlbumImagesProvider     = (*agent)(nil)
	_ metadata.AlbumInfoProvider       = (*agent)(nil)
	_ lyrics.Lyrics                    = (*agent)(nil)
)

// Init 注册全部能力
func Init() {
	a := &agent{}
	metadata.Register(a)
	lyrics.Register(a)
}

const cacheTTLSec = 7 * 24 * 3600 // ID 解析缓存 7 天

func imageRes() (string, int32) {
	v, ok := pdk.GetConfig("image_resolution")
	if !ok || v == "" {
		return "1200", 1200
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return "1200", 1200
	}
	return v, int32(n)
}

func cacheKey(kind, a, b string) string {
	k := kind + ":" + strings.ToLower(strings.ReplaceAll(a+b, " ", ""))
	return strings.ReplaceAll(k, "　", "")
}

// ---------- 歌手 ----------

type artistInfo struct {
	ID  int    `json:"id"`
	Pic string `json:"pic"`
}

func resolveArtist(name string) *artistInfo {
	key := cacheKey("cm:artist", name, "")
	if v, ok, _ := host.KVStoreGet(key); ok && len(v) > 0 {
		var info artistInfo
		if json.Unmarshal(v, &info) == nil && info.ID > 0 {
			return &info
		}
	}
	artists, err := netease.SearchArtist(name)
	if err != nil || len(artists) == 0 {
		return nil
	}
	best := netease.MatchArtist(name, artists)
	if best == nil {
		return nil
	}
	pic := best.Img1v1
	if pic == "" {
		pic = best.PicURL
	}
	info := artistInfo{ID: best.ID, Pic: pic}
	if b, err := json.Marshal(info); err == nil {
		// 注意:宿主 extism:host/user 未提供 kvstore_setwithttl,只能用无 TTL 版本
		_ = host.KVStoreSet(key, b)
	}
	pdk.Log(pdk.LogInfo, fmt.Sprintf("artist 匹配: %s -> id=%d", name, best.ID))
	return &info
}

func (a *agent) GetArtistURL(req metadata.ArtistRequest) (*metadata.ArtistURLResponse, error) {
	info := resolveArtist(req.Name)
	if info == nil {
		return nil, nil
	}
	return &metadata.ArtistURLResponse{URL: fmt.Sprintf("https://music.163.com/#/artist?id=%d", info.ID)}, nil
}

func (a *agent) GetArtistBiography(req metadata.ArtistRequest) (*metadata.ArtistBiographyResponse, error) {
	info := resolveArtist(req.Name)
	if info == nil {
		return nil, nil
	}
	brief, err := netease.ArtistDetail(info.ID)
	if err != nil || brief == "" {
		return nil, nil
	}
	return &metadata.ArtistBiographyResponse{Biography: strings.ReplaceAll(strings.TrimSpace(brief), "\n", "<br>")}, nil
}

func (a *agent) GetArtistImages(req metadata.ArtistRequest) (*metadata.ArtistImagesResponse, error) {
	info := resolveArtist(req.Name)
	if info == nil || info.Pic == "" {
		return nil, nil
	}
	res, size := imageRes()
	return &metadata.ArtistImagesResponse{Images: []metadata.ImageInfo{{URL: netease.PicWithRes(info.Pic, res), Size: size}}}, nil
}

// ---------- 专辑 ----------

type albumInfo struct {
	ID  int    `json:"id"`
	Pic string `json:"pic"`
}

func resolveAlbum(name, artist string) *albumInfo {
	key := cacheKey("cm:album", name, artist)
	if v, ok, _ := host.KVStoreGet(key); ok && len(v) > 0 {
		var info albumInfo
		if json.Unmarshal(v, &info) == nil && info.ID > 0 {
			return &info
		}
	}
	keyword := strings.TrimSpace(artist + " " + name)
	albums, err := netease.SearchAlbum(keyword)
	if err != nil || len(albums) == 0 {
		return nil
	}
	best := netease.MatchAlbum(name, artist, albums)
	if best == nil {
		return nil
	}
	info := albumInfo{ID: best.ID, Pic: best.PicURL}
	if b, err := json.Marshal(info); err == nil {
		_ = host.KVStoreSet(key, b)
	}
	pdk.Log(pdk.LogInfo, fmt.Sprintf("album 匹配: %s/%s -> id=%d", artist, name, best.ID))
	return &info
}

func (a *agent) GetAlbumImages(req metadata.AlbumRequest) (*metadata.AlbumImagesResponse, error) {
	info := resolveAlbum(req.Name, req.Artist)
	if info == nil {
		return nil, nil
	}
	pic := info.Pic
	if pic == "" {
		// 专辑搜索没给图时,经歌曲详情兜底
		if songs, err := netease.SearchSong(strings.TrimSpace(req.Artist + " " + req.Name)); err == nil && len(songs) > 0 {
			if best, _ := netease.Match(netease.MatchInput{Title: req.Name, Artist: req.Artist}, songs); best != nil {
				if d, err := netease.SongDetail(best.ID); err == nil {
					pic = d.CoverURL()
				}
			}
		}
		if pic == "" {
			return nil, nil
		}
	}
	res, size := imageRes()
	return &metadata.AlbumImagesResponse{Images: []metadata.ImageInfo{{URL: netease.PicWithRes(pic, res), Size: size}}}, nil
}

func (a *agent) GetAlbumInfo(req metadata.AlbumRequest) (*metadata.AlbumInfoResponse, error) {
	info := resolveAlbum(req.Name, req.Artist)
	if info == nil {
		return nil, nil
	}
	name, desc, _, err := netease.AlbumDetail(info.ID)
	if err != nil || desc == "" {
		return nil, nil
	}
	return &metadata.AlbumInfoResponse{Name: name, Description: strings.ReplaceAll(strings.TrimSpace(desc), "\n", "<br>")}, nil
}

// ---------- 歌词 ----------

func (a *agent) GetLyrics(req lyrics.GetLyricsRequest) (lyrics.GetLyricsResponse, error) {
	track := req.Track
	pdk.Log(pdk.LogInfo, fmt.Sprintf("lyrics 入口: %s - %s (album=%s)", track.Artist, track.Title, track.Album))
	artist := track.Artist
	if len(track.Artists) > 0 && track.Artists[0].Name != "" {
		artist = track.Artists[0].Name
	}
	if artist == "" {
		artist = track.AlbumArtist
	}

	in := netease.MatchInput{Title: track.Title, Artist: artist, Album: track.Album}
	songs, err := netease.SearchSong(in.Keyword())
	if err != nil || len(songs) == 0 {
		return lyrics.GetLyricsResponse{}, nil
	}
	best, score := netease.Match(in, songs)
	// 低于 50 分(连歌名都不完全匹配)视为没找到,避免错误歌词
	if best == nil || score < 50 {
		pdk.Log(pdk.LogInfo, fmt.Sprintf("lyrics 无可靠匹配: %s - %s (best=%d)", artist, track.Title, score))
		return lyrics.GetLyricsResponse{}, nil
	}

	orig, trans, err := netease.LyricWithTranslation(best.ID)
	if err != nil || orig == "" {
		return lyrics.GetLyricsResponse{}, nil
	}
	pdk.Log(pdk.LogInfo, fmt.Sprintf("lyrics 匹配: %s - %s -> %s id=%d score=%d", artist, track.Title, best.Name, best.ID, score))
	text := netease.MergeTranslation(orig, trans)
	return lyrics.GetLyricsResponse{Lyrics: []lyrics.LyricsText{{Lang: "zh", Text: text}}}, nil
}
