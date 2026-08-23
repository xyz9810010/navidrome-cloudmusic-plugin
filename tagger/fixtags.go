package main

import (
	"fmt"
	"time"

	"navidrome-cloudmusic-plugin/cloudmusic"
	"navidrome-cloudmusic-plugin/parser"
)

// fixItem 一条标签修复计划
type fixItem struct {
	file       string // 相对 /music 的文件名
	res        parser.Result
	neteaseHit string // 网易云匹配到的曲名(验证依据)
	neteaseAlb string
	score      int
}

// runFixTags 标签修复:解析特殊命名文件("0000.歌名 歌手 (DJ版)"),经网易云验证后
// 把 歌名/歌手/专辑 写回文件标签,治好 Navidrome 里的 [Unknown Artist]。
func runFixTags(sshHost, sshUser, sshPass string, limit int, apply bool) int {
	exec, err := NewSSHExec(sshHost, sshUser, sshPass)
	if err != nil {
		fatal("SSH 连接失败: %v", err)
	}
	defer exec.Close()

	files, err := exec.ListMusicFiles()
	if err != nil {
		fatal("列目录失败: %v", err)
	}
	fmt.Printf("音乐目录文件总数: %d\n", len(files))

	// 解析 + 网易云验证(候选逐个试,取最高分)
	var plan []fixItem
	checked, noParse, noMatch := 0, 0, 0
	for _, f := range files {
		if checked >= limit {
			break
		}
		cands, ok := parser.ParseCandidates(f)
		if !ok {
			noParse++
			continue
		}
		checked++
		var best fixItem
		for _, c := range cands {
			in := cloudmusic.MatchInput{Title: c.Title, Artist: c.Artist}
			songs, err := cloudmusic.Search(cloudmusic.CleanKeyword(in.Keyword()))
			if err != nil {
				continue
			}
			if song, score := cloudmusic.Match(in, songs); song != nil && score > best.score {
				best = fixItem{file: f, res: c, neteaseHit: song.Name, neteaseAlb: song.Album.Name, score: score}
			}
			time.Sleep(150 * time.Millisecond) // 对网易云客气一点
		}
		if best.score >= 70 {
			plan = append(plan, best)
		} else {
			noMatch++
		}
	}

	fmt.Printf("解析失败(非特殊命名) %d | 验证不通过 %d | 计划修复 %d:\n", noParse, noMatch, len(plan))
	for _, it := range plan {
		title := it.res.Title
		if it.res.Version != "" {
			title += " (" + it.res.Version + ")"
		}
		fmt.Printf("  %s\n    → 歌名=%q 歌手=%q 专辑=%q (网易云: %s, %d分)\n",
			it.file, title, it.res.Artist, it.neteaseAlb, it.neteaseHit, it.score)
	}
	if !apply || len(plan) == 0 {
		if !apply {
			fmt.Println("\n(dry-run,加 -apply 实际写入)")
		}
		return 0
	}

	fmt.Println("\n重建容器(音乐目录 rw)...")
	if err := exec.recreateContainer(); err != nil {
		fatal("容器重建失败: %v", err)
	}
	done, failed := 0, 0
	for _, it := range plan {
		title := it.res.Title
		if it.res.Version != "" {
			title += " (" + it.res.Version + ")"
		}
		if err := exec.WriteTags(it.file, title, it.res.Artist, it.neteaseAlb); err != nil {
			fmt.Printf("  [失败] %s: %v\n", it.file, err)
			failed++
		} else {
			done++
			fmt.Printf("  [完成] %s → %s / %s\n", it.file, title, it.res.Artist)
		}
	}
	fmt.Println("重建容器(恢复默认状态)...")
	if err := exec.recreateContainer(); err != nil {
		fmt.Printf("[警告] 重建失败: %v(请手动检查容器状态!)\n", err)
	}
	fmt.Printf("\n标签修复完成: %d 成功 / %d 失败(Navidrome 会自动重扫)\n", done, failed)
	return 0
}
