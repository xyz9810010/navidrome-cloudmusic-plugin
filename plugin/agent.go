package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	best, score := netease.MatchAlbum(name, artist, albums)
	// 低于 40 分视为不可信,不缓存(防止垃圾专辑信息进库)
	if best == nil || score < 40 {
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

// lyricCache 歌词缓存(插件 KVStore,免去重复搜索网易云)
type lyricCache struct {
	Found bool   `json:"found"`
	Text  string `json:"text,omitempty"`
	Ts    int64  `json:"ts,omitempty"` // 写入时间戳;负缓存过期后可重新搜索
}

// 负缓存有效期:30 天后允许重新搜索(网易云上新/曲库改名的歌有机会再命中)
const negativeCacheTTL = 30 * 24 * 3600

func setLyricCache(key string, found bool, text string) {
	if b, err := json.Marshal(lyricCache{Found: found, Text: text, Ts: time.Now().Unix()}); err == nil {
		// 注意:宿主未提供 kvstore_setwithttl,只能无 TTL 存储
		_ = host.KVStoreSet(key, b)
	}
}

// stashMatchCaches 歌词匹配成功时,顺手把 歌手/专辑 的网易云 ID 写进解析缓存:
// 之后打开歌手页/专辑页零搜索。图片链接拿不到就不写(避免空 pic 覆盖好值)。
func stashMatchCaches(in netease.MatchInput, best *netease.Song) {
	if best == nil {
		return
	}
	// 专辑:ID 必存(封面可再经 song/detail 兜底)
	if in.Album != "" && best.Album.ID > 0 {
		if b, err := json.Marshal(albumInfo{ID: best.Album.ID, Pic: best.Album.PicURL}); err == nil {
			_ = host.KVStoreSet(cacheKey("cm:album", in.Album, in.Artist), b)
		}
	}
	// 歌手:仅有图时才存,避免空 pic 永久遮蔽头像查询
	for _, ar := range best.Artists {
		pic := ar.Img1v1
		if pic == "" {
			pic = ar.PicURL
		}
		if ar.ID > 0 && pic != "" && in.Artist != "" {
			if b, err := json.Marshal(artistInfo{ID: ar.ID, Pic: pic}); err == nil {
				_ = host.KVStoreSet(cacheKey("cm:artist", in.Artist, ""), b)
			}
			break
		}
	}
}

func (a *agent) GetLyrics(req lyrics.GetLyricsRequest) (lyrics.GetLyricsResponse, error) {
	track := req.Track
	artist := track.Artist
	if len(track.Artists) > 0 && track.Artists[0].Name != "" {
		artist = track.Artists[0].Name
	}
	if artist == "" {
		artist = track.AlbumArtist
	}

	// 缓存命中直接返回(含负缓存:之前确认过没有的不再搜)
	// v3: 新增无标签文件的"曲名内拆歌手"识别,旧负缓存不再拦截
	key := cacheKey("cm:lyric3", artist, track.Title)
	if v, ok, _ := host.KVStoreGet(key); ok && len(v) > 0 {
		var lc lyricCache
		if json.Unmarshal(v, &lc) == nil {
			if lc.Found {
				pdk.Log(pdk.LogInfo, fmt.Sprintf("lyrics 缓存命中: %s - %s", artist, track.Title))
				writeLrcSidecar(track.Path, lc.Text)
				return lyrics.GetLyricsResponse{Lyrics: []lyrics.LyricsText{{Lang: "zh", Text: lc.Text}}}, nil
			}
			// 负缓存过期则重新搜索(老条目无时间戳视为永久)
			if lc.Ts == 0 || time.Now().Unix()-lc.Ts < negativeCacheTTL {
				return lyrics.GetLyricsResponse{}, nil
			}
			pdk.Log(pdk.LogInfo, fmt.Sprintf("lyrics 负缓存过期,重新搜索: %s - %s", artist, track.Title))
		}
	}

	pdk.Log(pdk.LogInfo, fmt.Sprintf("lyrics 入口: %s - %s (album=%s)", artist, track.Title, track.Album))

	in := netease.MatchInput{Title: track.Title, Artist: artist, Album: track.Album}
	best, via := resolveTrack(in)
	if best == nil {
		pdk.Log(pdk.LogInfo, fmt.Sprintf("lyrics 无可靠匹配(三级兜底后): %s - %s", artist, track.Title))
		setLyricCache(key, false, "")
		return lyrics.GetLyricsResponse{}, nil
	}
	stashMatchCaches(in, best)

	orig, trans, err := netease.LyricWithTranslation(best.ID)
	if err != nil {
		// 歌词接口失败可能是网络抖动,不写负缓存
		return lyrics.GetLyricsResponse{}, nil
	}
	if orig == "" {
		setLyricCache(key, false, "")
		return lyrics.GetLyricsResponse{}, nil
	}
	pdk.Log(pdk.LogInfo, fmt.Sprintf("lyrics 匹配[%s]: %s - %s -> %s id=%d", via, artist, track.Title, best.Name, best.ID))
	text := netease.MergeTranslation(orig, trans)
	setLyricCache(key, true, text)
	writeLrcSidecar(track.Path, text)
	return lyrics.GetLyricsResponse{Lyrics: []lyrics.LyricsText{{Lang: "zh", Text: text}}}, nil
}

// isUnknownArtist 占位歌手:"[Unknown Artist]" / "未知歌手" 等
func isUnknownArtist(artist string) bool {
	a := strings.ToLower(strings.TrimSpace(artist))
	return a == "" || strings.Contains(a, "unknown") || strings.Contains(a, "未知")
}

// normalizeInput 处理无标签文件:歌手未知时,曲名里往往粘着歌手
// ("0000.一曲相思 半阳 （DJ版）"),按空格拆出 "末段=歌手" 变体,逐个尝试
func normalizeInput(in netease.MatchInput) []netease.MatchInput {
	if !isUnknownArtist(in.Artist) {
		return []netease.MatchInput{in}
	}
	in.Artist = ""
	core := netease.SearchTitle(in.Title)
	fields := strings.Fields(core)
	if len(fields) < 2 {
		return []netease.MatchInput{in}
	}
	v1 := in // 末段为歌手
	v1.Title = strings.Join(fields[:len(fields)-1], " ")
	v1.Artist = fields[len(fields)-1]
	v2 := in // 全部当曲名,歌手空
	v2.Title = core
	return []netease.MatchInput{v1, v2}
}

// resolveTrack 三级匹配,应对"歌名写法不一致":
//  1. 直搜歌曲(关键词去标点) —— 解决 "下个,路口,见" vs "下个路口见"
//  2. 专辑曲目反查 —— 专辑名通常更稳定,拿网易云专辑曲名表再匹配
//  3. 歌手热门歌曲 —— 曲名乱/专辑错但歌手对时,从热门里捞
//
// 无标签文件先拆出"曲名+歌手"变体,逐变体走三级;任一命中(≥50分)返回
func resolveTrack(in netease.MatchInput) (*netease.Song, string) {
	for _, v := range normalizeInput(in) {
		if best, via := resolveOne(v); best != nil {
			return best, via
		}
	}
	return nil, ""
}

func resolveOne(in netease.MatchInput) (*netease.Song, string) {
	// 1. 直搜
	if songs, err := netease.SearchSong(netease.CleanKeyword(in.Keyword())); err == nil && len(songs) > 0 {
		if best, score := netease.Match(in, songs); best != nil && score >= 50 {
			return best, fmt.Sprintf("直搜%d分", score)
		}
	}
	// 2. 专辑曲目(专辑门槛 40 分:内部曲目匹配仍要求 ≥50,错误成本只是一次请求)
	if in.Album != "" && !strings.Contains(strings.ToLower(in.Album), "unknown") {
		albums, err := netease.SearchAlbum(netease.CleanKeyword(in.Artist + " " + in.Album))
		if err == nil && len(albums) > 0 {
			if al, score := netease.MatchAlbum(in.Album, in.Artist, albums); al != nil && score >= 40 {
				if tracks, err := netease.AlbumTracks(al.ID); err == nil && len(tracks) > 0 {
					if best, ts := netease.Match(in, tracks); best != nil && ts >= 50 {
						return best, "专辑曲目"
					}
				}
			}
		}
	}
	// 3. 歌手热门
	if in.Artist != "" {
		artists, err := netease.SearchArtist(in.Artist)
		if err == nil && len(artists) > 0 {
			if ar := netease.MatchArtist(in.Artist, artists); ar != nil {
				if hots, err := netease.ArtistHotSongs(ar.ID); err == nil && len(hots) > 0 {
					if best, ts := netease.Match(in, hots); best != nil && ts >= 50 {
						return best, "歌手热门"
					}
				}
			}
		}
	}
	return nil, ""
}

// writeLrcSidecar 把歌词写成音轨旁的 .lrc 伴生文件。
// 需要 library+filesystem 权限(track.Path 才会有值)且容器音乐目录 rw 挂载;
// 失败只记日志,不影响歌词返回。Navidrome 的 LyricsPriority 里 .lrc 排在插件前,
// 写好之后这首歌就走文件、零网络请求。
func writeLrcSidecar(relPath, text string) {
	if relPath == "" || text == "" {
		return
	}
	libs, err := host.LibraryGetAllLibraries()
	if err != nil || len(libs) == 0 {
		pdk.Log(pdk.LogWarn, "lrc 写入跳过: 拿不到媒体库信息")
		return
	}
	// WASI 预开目录优先用 MountPoint,空则退回 Path
	root := libs[0].MountPoint
	if root == "" {
		root = libs[0].Path
	}
	full := strings.TrimSuffix(root, "/") + "/" + strings.TrimPrefix(relPath, "/")
	lrc := strings.TrimSuffix(full, filepath.Ext(full)) + ".lrc"
	// 已存在则跳过:避免每次播放重复写盘、也避免触发 Navidrome 文件监听重扫
	if st, err := os.Stat(lrc); err == nil && st.Size() > 0 {
		return
	}
	if err := os.WriteFile(lrc, []byte(text), 0644); err != nil {
		pdk.Log(pdk.LogWarn, "lrc 写入失败("+lrc+"): "+err.Error())
		return
	}
	pdk.Log(pdk.LogInfo, "lrc 已写入: "+lrc)
}
