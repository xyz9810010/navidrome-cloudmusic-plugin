package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHExec 飞牛 NAS 的 SSH 执行器:负责容器 rw/ro 切换与 ffmpeg 写标签
type SSHExec struct {
	Client *ssh.Client
}

func NewSSHExec(host, user, password string) (*SSHExec, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, err := ssh.Dial("tcp", net.JoinHostPort(host, "22"), cfg)
	if err != nil {
		return nil, err
	}
	return &SSHExec{Client: client}, nil
}

func (s *SSHExec) Close() { _ = s.Client.Close() }

// Run 执行命令并要求退出码为 0,返回输出
func (s *SSHExec) Run(cmd string) (string, error) {
	sess, err := s.Client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	var buf strings.Builder
	sess.Stdout = &buf
	sess.Stderr = &buf
	if err := sess.Run(cmd); err != nil {
		return buf.String(), fmt.Errorf("命令失败(%v): %s", err, lastLines(buf.String(), 5))
	}
	return buf.String(), nil
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// recreateContainer 重建 navidrome-cn 容器,musicRW 控制音乐目录挂载模式。
// 容器是无状态的(数据在卷里),重建是安全的;也用于修改 ND_* 环境变量。
func (s *SSHExec) recreateContainer(musicRW bool) error {
	mode := "ro"
	if musicRW {
		mode = "rw"
	}
	cmd := fmt.Sprintf(`docker rm -f navidrome-cn >/dev/null 2>&1; docker run -d --name navidrome-cn \
--restart unless-stopped \
-p 4535:4533 \
-v /vol2/1000/music:/music:%s \
-v /vol1/@appdata/navidrome-cn-data:/data \
-e ND_DEFAULTLANGUAGE=zh-Hans \
-e ND_SCANNER_EXTRACTOR=ffmpeg \
-e ND_COVERARTPRIORITY=external,embedded \
-e ND_AGENTS=cloudmusic,deezer,lastfm,listenbrainz,apple-music \
-e ND_LYRICSPRIORITY=embedded,.lrc,cloudmusic,nd-lyrics \
-e ND_MUSICFOLDER=/music \
-e ND_DATAFOLDER=/data \
-e ND_CONFIGFILE=/data/navidrome.toml \
-e ND_PORT=4533 \
navidrome-cn:test`, mode)
	out, err := s.Run(cmd)
	if err != nil {
		return err
	}
	if strings.Contains(out, "Error") || strings.Contains(out, "error") {
		return fmt.Errorf("docker run 输出异常: %s", lastLines(out, 3))
	}
	// 等容器就绪
	for i := 0; i < 15; i++ {
		time.Sleep(time.Second)
		out, _ := s.Run("docker ps --filter name=navidrome-cn --filter status=running -q | wc -l")
		if strings.TrimSpace(out) == "1" {
			return nil
		}
	}
	return fmt.Errorf("容器未在 15 秒内就绪")
}

// WriteGenreTag 用容器内 ffmpeg 给单个文件写 GENRE 标签(流拷贝,不重编码)。
// path 为相对 /music 的路径。先写临时文件再原子替换。
func (s *SSHExec) WriteGenreTag(path, genre string) error {
	ext := ".flac"
	if i := strings.LastIndex(path, "."); i >= 0 {
		ext = path[i:]
	}
	shell := fmt.Sprintf(
		`set -e; cd /music; f='%s'; ffmpeg -y -loglevel error -i "$f" -map 0 -c copy -metadata genre='%s' -f %s "$f.tagging.tmp" && mv "$f.tagging.tmp" "$f"`,
		shQuote(path), shQuote(genre), strings.TrimPrefix(ext, "."))
	_, err := s.Run("docker exec navidrome-cn sh -c '" + shQuote(shell) + "'")
	return err
}

// shQuote 单引号转义,防止路径注入
func shQuote(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

// QuerySongPaths 用 media_file.id(Subsonic song id)批量查真实相对路径。
// 必须查库:该 fork 的 Subsonic API 返回虚拟层级路径,与磁盘扁平结构不符。
func (s *SSHExec) QuerySongPaths(ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + id + "'"
	}
	cmd := fmt.Sprintf(
		`docker exec navidrome-cn sqlite3 /data/navidrome.db "select id || '|' || path from media_file where id in (%s);"`,
		strings.Join(quoted, ","))
	out, err := s.Run(cmd)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if i := strings.Index(line, "|"); i > 0 {
			m[line[:i]] = line[i+1:]
		}
	}
	return m, nil
}
