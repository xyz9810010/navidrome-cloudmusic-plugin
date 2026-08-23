package main

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"navidrome-cloudmusic-plugin/navid"
)

// numberedRe 序号开头:"0000." / "2375、" / 无分隔符零填充 "0000忘川"
var numberedRe = regexp.MustCompile(`^(\d{1,4}[.、．]|0\d{2,4}[^0-9])`)

// renameItem 一条改名计划
type renameItem struct {
	old, new string
}

// runRenameNumbered 仅对序号开头的文件改名:"歌名 - 歌手.扩展名"(与库中非序号文件惯例一致),
// 伴生 .lrc 同步改名。来源用 Navidrome 库里的标签(修复过的),未知歌手跳过。
func runRenameNumbered(cfg localConfig, limit int, apply bool) int {
	exec, err := NewSSHExec(cfg.SSHHost, cfg.SSHUser, cfg.SSHPassword)
	if err != nil {
		fatal("SSH 连接失败: %v", err)
	}
	defer exec.Close()

	rows, err := exec.QueryAllSongs()
	if err != nil {
		fatal("查询曲库失败: %v", err)
	}

	var plan []renameItem
	skipUnknown, skipSame, skipConflict := 0, 0, 0
	exists := map[string]bool{}
	for _, r := range rows {
		exists[r.Path] = true
	}
	for _, r := range rows {
		base := path.Base(r.Path)
		if !numberedRe.MatchString(base) {
			continue
		}
		if isUnknown(r.Artist) || strings.TrimSpace(r.Artist) == "" {
			skipUnknown++
			continue
		}
		ext := pathExt(r.Path)
		artist := strings.ReplaceAll(r.Artist, "/", ",") // 多歌手与库内惯例一致
		newBase := fmt.Sprintf("%s - %s%s", r.Title, artist, ext)
		if newBase == base {
			skipSame++
			continue
		}
		newRel := path.Join(path.Dir(r.Path), newBase)
		if exists[newRel] {
			skipConflict++
			continue
		}
		plan = append(plan, renameItem{old: r.Path, new: newRel})
	}

	fmt.Printf("序号开头文件: 计划改名 %d | 跳过(未知歌手 %d / 同名 %d / 冲突 %d)\n",
		len(plan), skipUnknown, skipSame, skipConflict)
	if limit > 0 && len(plan) > limit {
		plan = plan[:limit]
		fmt.Printf("本次处理: %d 个(限制)\n", limit)
	}
	for i, it := range plan {
		if i < 15 {
			fmt.Printf("  %s\n    → %s\n", it.old, path.Base(it.new))
		}
	}
	if len(plan) > 15 {
		fmt.Printf("  ... 共 %d 个\n", len(plan))
	}
	if !apply || len(plan) == 0 {
		if !apply {
			fmt.Println("\n(dry-run,加 -apply 实际改名)")
		}
		return 0
	}

	done, failed := 0, 0
	for _, it := range plan {
		if err := exec.RenameSong(it.old, it.new); err != nil {
			fmt.Printf("  [失败] %s: %v\n", it.old, err)
			failed++
		} else {
			done++
		}
	}
	fmt.Printf("\n改名完成: %d 成功 / %d 失败\n", done, failed)

	// 触发重扫让 Navidrome 更新路径
	sc := navid.New(cfg.Server, cfg.User, cfg.Password)
	_ = sc.StartScan()
	fmt.Println("已触发重扫,播放列表稍后自动更新")
	return 0
}

func isUnknown(artist string) bool {
	a := strings.ToLower(artist)
	return strings.Contains(a, "unknown") || strings.Contains(a, "未知")
}
