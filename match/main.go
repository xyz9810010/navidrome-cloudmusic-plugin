// match 手动匹配工具:本地歌 ↔ 网易云歌曲人工指定对应关系。
//
// 自动匹配验证不过的歌(网易云搜不到/名字差太远),用这个手动指定:
// 选本地歌 → 选网易云候选 → 写入 .lrc 伴生文件 + 插件歌词缓存。
//
// 用法:
//
//	go run ./match -q "黄昏 周传雄"                  # 交互式,按提示选
//	go run ./match -q "黄昏" -local 1 -netease 2     # 非交互:第1首本地歌 ↔ 网易云第2条
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"navidrome-cloudmusic-plugin/cloudmusic"
	"navidrome-cloudmusic-plugin/nassh"
	"navidrome-cloudmusic-plugin/navid"
)

func main() {
	var (
		query   = flag.String("q", "", "搜索关键词(本地歌名/歌手)")
		localIx = flag.Int("local", 0, "直接选第 N 首本地歌(0=交互选择)")
		netIx   = flag.Int("netease", 0, "直接选第 N 条网易云结果(0=交互选择)")
		batch   = flag.Bool("batch", false, "批量模式:扫描无.lrc的歌自动匹配补词(≥50分才写)")
		limit   = flag.Int("limit", 0, "批量模式处理上限(0=全部)")
	)
	flag.Parse()

	cfg := loadConfig()
	if cfg.Server == "" || cfg.SSHHost == "" {
		fmt.Println("[错误] 缺少 config.json(复制 config.example.json 填写)")
		os.Exit(1)
	}

	// 批量模式
	if *batch {
		runBatch(cfg, *limit)
		return
	}

	if *query == "" {
		fmt.Println("用法:")
		fmt.Println("  go run ./match -q \"歌名 歌手\" [-local N -netease N]")
		fmt.Println("  go run ./match -batch [-limit N]   # 批量补无歌词的歌")
		os.Exit(1)
	}

	sc := navid.New(cfg.Server, cfg.User, cfg.Password)

	// 1. 本地歌
	songs, err := sc.SearchSongs(*query, 10)
	if err != nil {
		fatal("搜索本地库失败: %v", err)
	}
	if len(songs) == 0 {
		fatal("本地库没有匹配的歌")
	}
	fmt.Println("=== 本地歌曲 ===")
	for i, s := range songs {
		fmt.Printf("  %2d. %s | %s | %s\n", i+1, s.Title, s.Artist, s.Album)
	}
	li := pick(os.Stdin, *localIx, len(songs), "选本地歌")
	song := songs[li-1]

	// 2. 网易云候选
	keyword := cloudmusic.CleanKeyword(song.Artist + " " + song.Title)
	results, err := cloudmusic.Search(keyword)
	if err != nil {
		fatal("网易云搜索失败: %v", err)
	}
	in := cloudmusic.MatchInput{Title: song.Title, Artist: song.Artist, Album: song.Album}
	fmt.Printf("\n=== 网易云候选 (关键词: %s) ===\n", keyword)
	for i, r := range results {
		score := cloudmusic.Score(in, r)
		fmt.Printf("  %2d. %s | %s | %s | id=%d | 匹配%d分\n", i+1, r.Name, r.ArtistNames(), r.Album.Name, r.ID, score)
	}
	if len(results) == 0 {
		fatal("网易云没有结果,换个关键词试试")
	}
	ni := pick(os.Stdin, *netIx, len(results), "选网易云(回车=第一条)")
	picked := results[ni-1]

	// 3. 取词并落地
	orig, trans, err := cloudmusic.LyricWithTranslation(picked.ID)
	if err != nil || orig == "" {
		fatal("这首歌在网易云没有歌词(id=%d)", picked.ID)
	}
	text := cloudmusic.FilterNoise(cloudmusic.MergeTranslation(orig, trans))

	ssh, err := nassh.New(cfg.SSHHost, cfg.SSHUser, cfg.SSHPassword)
	if err != nil {
		fatal("SSH 连接失败: %v", err)
	}
	defer ssh.Close()

	// 3a. 写 .lrc 伴生文件
	paths, err := ssh.QuerySongPaths([]string{song.ID})
	if err != nil {
		fmt.Printf("[警告] 查真实路径失败: %v(跳过 .lrc)\n", err)
	} else if p, ok := paths[song.ID]; ok {
		if err := ssh.WriteLrcFile(p, text); err != nil {
			fmt.Printf("[警告] 写 .lrc 失败: %v\n", err)
		} else {
			fmt.Printf("[OK] .lrc 已写入: %s\n", p)
		}
	}

	// 3b. 写插件歌词缓存(立即生效,不用等重扫)
	if err := ssh.SetLyricCache(song.Artist, song.Title, text); err != nil {
		fmt.Printf("[警告] 写插件缓存失败: %v\n", err)
	} else {
		fmt.Printf("[OK] 插件歌词缓存已更新: %s - %s ↔ 网易云《%s》(id=%d)\n",
			song.Artist, song.Title, picked.Name, picked.ID)
	}
	fmt.Println("完成。播放这首歌即可看到指定歌词。")
}

func pick(r *os.File, preset, max int, prompt string) int {
	if preset >= 1 && preset <= max {
		return preset
	}
	fmt.Printf("\n%s [1-%d, 回车=1]: ", prompt, max)
	reader := bufio.NewReader(r)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return 1
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > max {
		fatal("无效选择: %s", line)
	}
	return n
}

type localConfig struct {
	Server      string `json:"server"`
	User        string `json:"user"`
	Password    string `json:"password"`
	SSHHost     string `json:"ssh_host"`
	SSHUser     string `json:"ssh_user"`
	SSHPassword string `json:"ssh_password"`
}

func loadConfig() localConfig {
	var cfg localConfig
	if b, err := os.ReadFile("config.json"); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	return cfg
}

func fatal(format string, args ...any) {
	fmt.Printf("[错误] "+format+"\n", args...)
	os.Exit(1)
}
