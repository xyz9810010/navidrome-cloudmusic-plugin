// tagger 标签补全工具:给无流派的专辑写 GENRE 标签。
//
// 数据链:Navidrome(无流派专辑) -> 网易云匹配(专辑type + 匹配曲名) -> 规则推断标签
//        -> SSH 临时以 rw 挂载重建容器 -> ffmpeg 流拷贝写 FLAC/MP3 标签 -> 恢复 ro
//
// 用法:
//   go run ./tagger -limit 20            # dry-run,只输出计划
//   go run ./tagger -limit 20 -apply     # 实际写入
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"navidrome-cloudmusic-plugin/cloudmusic"
)

// localConfig 本地凭证配置(gitignore 的 config.json)
type localConfig struct {
	Server     string `json:"server"`
	User       string `json:"user"`
	Password   string `json:"password"`
	SSHHost    string `json:"ssh_host"`
	SSHUser    string `json:"ssh_user"`
	SSHPassword string `json:"ssh_password"`
}

func loadConfig() localConfig {
	var cfg localConfig
	if b, err := os.ReadFile("config.json"); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}
	return cfg
}

type planItem struct {
	album   Album
	songs   []Song
	tags    []string
	reason  string
}

func main() {
	var (
		server  = flag.String("server", "", "Navidrome 地址(默认读 config.json)")
		user    = flag.String("user", "", "Navidrome 用户")
		pass    = flag.String("pass", "", "Navidrome 密码")
		sshHost = flag.String("ssh-host", "", "NAS SSH 地址")
		sshUser = flag.String("ssh-user", "", "NAS SSH 用户")
		sshPass = flag.String("ssh-pass", "", "NAS SSH 密码")
		limit   = flag.Int("limit", 10, "处理数量上限")
		apply   = flag.Bool("apply", false, "实际写入(默认 dry-run)")
		fixTags = flag.Bool("fix-tags", false, "标签修复模式:解析特殊命名文件,写回歌名/歌手/专辑标签")
		renameN = flag.Bool("rename-numbered", false, "改名模式:仅序号开头的文件改为 '歌名 - 歌手.扩展名'")
	)
	flag.Parse()

	// 命令行未给的参数从 config.json 补
	cfg := loadConfig()
	if *server == "" {
		*server = cfg.Server
	}
	if *user == "" {
		*user = cfg.User
	}
	if *pass == "" {
		*pass = cfg.Password
	}
	if *sshHost == "" {
		*sshHost = cfg.SSHHost
	}
	if *sshUser == "" {
		*sshUser = cfg.SSHUser
	}
	if *sshPass == "" {
		*sshPass = cfg.SSHPassword
	}
	if *server == "" || *user == "" || *pass == "" {
		fatal("缺少 Navidrome 地址/账号:请复制 config.example.json 为 config.json 填写,或用 -server/-user/-pass 指定")
	}

	// 标签修复模式:只需 SSH,不走 Subsonic
	if *fixTags {
		if *sshHost == "" || *sshUser == "" || *sshPass == "" {
			fatal("fix-tags 需要 SSH 配置(config.json 或 -ssh-* 参数)")
		}
		os.Exit(runFixTags(*sshHost, *sshUser, *sshPass, *limit, *apply))
	}

	// 序号文件改名模式
	if *renameN {
		if *sshHost == "" || *sshUser == "" || *sshPass == "" {
			fatal("rename-numbered 需要 SSH 配置(config.json 或 -ssh-* 参数)")
		}
		os.Exit(runRenameNumbered(cfg, *limit, *apply))
	}

	sc := NewSubsonicClient(*server, *user, *pass)

	// 1. 拉专辑,筛出无流派的
	var targets []Album
	for offset := 0; len(targets) < *limit; offset += 100 {
		albums, err := sc.GetAlbums(offset, 100)
		if err != nil {
			fatal("拉取专辑失败: %v", err)
		}
		if len(albums) == 0 {
			break
		}
		for _, a := range albums {
			if strings.TrimSpace(a.Genre) == "" {
				targets = append(targets, a)
				if len(targets) >= *limit {
					break
				}
			}
		}
	}
	fmt.Printf("无流派专辑: 取 %d 张(上限 %d)\n", len(targets), *limit)
	if len(targets) == 0 {
		return
	}

	// 2. 逐张:网易云匹配 + 规则推断
	var plan []planItem
	for _, al := range targets {
		item := planItem{album: al}
		songs, err := sc.GetAlbumSongs(al.ID)
		if err != nil || len(songs) == 0 {
			continue
		}
		item.songs = songs

		// 专辑匹配 -> type
		ndAlbumType := ""
		if albums, err := cloudmusic.SearchAlbum(cloudmusic.CleanKeyword(al.Artist + " " + al.Name)); err == nil {
			for i := range albums {
				if normEq(albums[i].Name, al.Name) {
					ndAlbumType = albums[i].Type
					break
				}
			}
		}

		// 曲名匹配:用专辑第一首歌
		first := songs[0]
		matched := ""
		if results, err := cloudmusic.Search(cloudmusic.CleanKeyword(first.Artist + " " + first.Title)); err == nil {
			in := cloudmusic.MatchInput{Title: first.Title, Artist: first.Artist, Album: al.Name}
			if best, score := cloudmusic.Match(in, results); best != nil && score >= 50 {
				matched = best.Name
			}
		}

		item.tags = DeriveTags(al.Name, ndAlbumType, matched)
		item.reason = fmt.Sprintf("ndType=%q matchedSong=%q", ndAlbumType, matched)
		if len(item.tags) > 0 {
			plan = append(plan, item)
		} else {
			fmt.Printf("  [跳过] %s / %s (无法推断: %s)\n", al.Artist, al.Name, item.reason)
		}
	}

	fmt.Printf("\n=== 计划(共 %d 张)===\n", len(plan))
	for _, it := range plan {
		fmt.Printf("  [%s] %s / %s -> %s\n", it.album.ID, it.album.Artist, it.album.Name, strings.Join(it.tags, ";"))
	}
	if !*apply || len(plan) == 0 {
		if !*apply {
			fmt.Println("\n(dry-run,加 -apply 实际写入)")
		}
		return
	}

	// 3. 写入:r 容器 rw -> ffmpeg 写标签 -> 恢复 ro
	exec, err := NewSSHExec(*sshHost, *sshUser, *sshPass)
	if err != nil {
		fatal("SSH 连接失败: %v", err)
	}
	defer exec.Close()

	fmt.Println("\n重建容器(确保音乐目录 rw + 完整环境变量)...")
	if err := exec.recreateContainer(); err != nil {
		fatal("容器重建失败: %v", err)
	}

	done, failed := 0, 0
	for _, it := range plan {
		genre := MergeGenre(it.tags)
		ids := make([]string, 0, len(it.songs))
		for _, song := range it.songs {
			ids = append(ids, song.ID)
		}
		realPaths, err := exec.QuerySongPaths(ids)
		if err != nil {
			fmt.Printf("  [失败] %s / %s: 路径解析失败: %v\n", it.album.Artist, it.album.Name, err)
			failed++
			continue
		}
		okAll := true
		for _, song := range it.songs {
			if song.Suffix != "flac" && song.Suffix != "mp3" {
				continue // 只处理 flac/mp3
			}
			real, ok := realPaths[song.ID]
			if !ok {
				continue
			}
			if err := exec.WriteGenreTag(real, genre); err != nil {
				fmt.Printf("  [失败] %s: %v\n", real, err)
				okAll = false
				break
			}
		}
		if okAll {
			done++
			fmt.Printf("  [完成] %s / %s -> %s\n", it.album.Artist, it.album.Name, genre)
		} else {
			failed++
		}
	}

	fmt.Println("重建容器(恢复默认状态)...")
	if err := exec.recreateContainer(); err != nil {
		fmt.Printf("[警告] 重建失败: %v(请手动检查容器状态!)\n", err)
	}

	// 4. 等重扫后报告流派统计
	fmt.Printf("\n写入完成: %d 成功 / %d 失败,等待重扫...\n", done, failed)
	time.Sleep(20 * time.Second)
	genres, err := sc.GetGenres()
	if err == nil {
		var names []string
		for g, n := range genres {
			names = append(names, fmt.Sprintf("%s(%d)", g, n))
		}
		sort.Strings(names)
		fmt.Println("当前流派:", strings.Join(names, " "))
	}
}

func normEq(a, b string) bool {
	na, nb := cloudmusic.Norm(a), cloudmusic.Norm(b)
	return na == nb || strings.Contains(na, nb) || strings.Contains(nb, na)
}

func fatal(format string, args ...any) {
	fmt.Printf("[错误] "+format+"\n", args...)
	os.Exit(1)
}
