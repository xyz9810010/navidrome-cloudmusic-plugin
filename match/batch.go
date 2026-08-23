package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"navidrome-cloudmusic-plugin/cloudmusic"
	"navidrome-cloudmusic-plugin/nassh"
	"navidrome-cloudmusic-plugin/parser"
)

// batchItem 一首待匹配的歌
type batchItem struct {
	relPath string // 相对 /music 的真实路径
	title   string // 库中标题
	artist  string // 库中歌手
	songID  string // Subsonic/库 id
}

// runBatch 批量补歌词:
// 扫描无 .lrc 伴生文件的歌 → 网易云搜索匹配(≥50分)→ 写 .lrc + 插件缓存。
// [Unknown Artist] 的文件先走 parser 拆"歌名 歌手"再匹配。
func runBatch(cfg localConfig, limit int) {
	ssh, err := nassh.New(cfg.SSHHost, cfg.SSHUser, cfg.SSHPassword)
	if err != nil {
		fatal("SSH 连接失败: %v", err)
	}
	defer ssh.Close()

	items, err := ssh.ListSongsMissingLrc()
	if err != nil {
		fatal("扫描无歌词文件失败: %v", err)
	}
	fmt.Printf("无 .lrc 伴生文件的歌: %d 首\n", len(items))
	if limit > 0 && len(items) > limit {
		items = items[:limit]
		fmt.Printf("本次处理: %d 首(限制)\n", limit)
	}

	done, skipped, failed := 0, 0, 0
	var remaining []string
	for i, it := range items {
		// 候选输入:有标签直接一条;未知歌手走 parser 拆
		inputs := []cloudmusic.MatchInput{{Title: it.Title, Artist: it.Artist}}
		if strings.Contains(it.Artist, "Unknown") || strings.TrimSpace(it.Artist) == "" {
			if cands, ok := parser.ParseCandidates(it.RelPath); ok {
				inputs = nil
				for _, c := range cands {
					inputs = append(inputs, cloudmusic.MatchInput{Title: c.Title, Artist: c.Artist})
				}
			}
		}

		var best *cloudmusic.Song
		var bestIn cloudmusic.MatchInput
		bestScore := 0
		for _, in := range inputs {
			results, err := cloudmusic.Search(cloudmusic.CleanKeyword(in.Keyword()))
			if err != nil {
				continue
			}
			if s, score := cloudmusic.Match(in, results); s != nil && score > bestScore {
				best, bestIn, bestScore = s, in, score
			}
			time.Sleep(250 * time.Millisecond)
		}

		if best == nil || bestScore < 50 {
			skipped++
			if len(remaining) < 2000 {
				remaining = append(remaining, fmt.Sprintf("%s | 库[%s/%s] 最佳候选 %d分", it.RelPath, it.Artist, it.Title, bestScore))
			}
		} else {
			orig, trans, err := cloudmusic.LyricWithTranslation(best.ID)
			if err != nil || orig == "" {
				skipped++
				continue
			}
			text := cloudmusic.FilterNoise(cloudmusic.MergeTranslation(orig, trans))
			if err := ssh.WriteLrcFile(it.RelPath, text); err != nil {
				failed++
				fmt.Printf("  [lrc失败] %s: %v\n", it.RelPath, err)
				continue
			}
			if err := ssh.SetLyricCache(bestIn.Artist, bestIn.Title, text); err != nil {
				fmt.Printf("  [缓存失败] %s: %v\n", it.RelPath, err)
			}
			done++
		}

		if (i+1)%25 == 0 {
			fmt.Printf("  进度 %d/%d (成功%d 跳过%d)\n", i+1, len(items), done, skipped)
		}
	}

	fmt.Printf("\n完成: 成功 %d | 跳过 %d | 失败 %d\n", done, skipped, failed)
	if len(remaining) > 0 {
		_ = writeFile("match-remaining.txt", strings.Join(remaining, "\n"))
		fmt.Printf("剩余 %d 首(候选<50分)已写入 match-remaining.txt\n", len(remaining))
	}
}

func writeFile(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
