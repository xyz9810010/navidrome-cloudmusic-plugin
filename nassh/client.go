// Package nassh NAS 的 SSH 执行器(match 等工具共用):
// 查询歌曲真实路径、写 .lrc 伴生文件、写插件 KVStore 歌词缓存。
package nassh

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	conn *ssh.Client
}

func New(host, user, password string) (*Client, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, "22"), cfg)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() { _ = c.conn.Close() }

// Run 执行命令并要求退出码为 0
func (c *Client) Run(cmd string) (string, error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	var b strings.Builder
	sess.Stdout = &b
	sess.Stderr = &b
	if err := sess.Run(cmd); err != nil {
		return b.String(), fmt.Errorf("命令失败(%v): %s", err, lastLines(b.String(), 4))
	}
	return b.String(), nil
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// shQuote 单引号转义
func shQuote(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

// QuerySongPaths 用 Subsonic song id(=media_file.id)查真实相对路径。
// 该 fork 的 Subsonic API 返回虚拟层级路径,必须查库。
func (c *Client) QuerySongPaths(ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + id + "'"
	}
	out, err := c.Run(fmt.Sprintf(
		`docker exec navidrome-cn sqlite3 /data/navidrome.db "select id || '|' || path from media_file where id in (%s);"`,
		strings.Join(quoted, ",")))
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

// WriteLrcFile 把 lrc 文本写成音乐库里的伴生文件(base64 防转义)。
// 直接写宿主机路径,不经过容器——容器挂载可能被飞牛重建打回只读。
func (c *Client) WriteLrcFile(relPath, lrc string) error {
	outPath := strings.TrimSuffix(relPath, ".flac")
	outPath = strings.TrimSuffix(outPath, ".mp3") + ".lrc"
	cmd := fmt.Sprintf(
		`echo '%s' | base64 -d > '/vol1/1000/音乐/%s'`,
		hex.EncodeToString([]byte(lrc)), shQuote(outPath))
	_, err := c.Run(cmd)
	return err
}

// SetLyricCache 把手动指定的歌词写进 cloudmusic 插件的 KVStore,
// key 规则必须与插件 agent.go 的 cacheKey 完全一致(cm:lyric3 前缀)。
func (c *Client) SetLyricCache(artist, title, text string) error {
	value := fmt.Sprintf(`{"found":true,"text":%s}`, mustJSON(text))
	key := "cm:lyric3:" + strings.ReplaceAll(strings.ToLower(artist+title), " ", "")
	key = strings.ReplaceAll(key, "　", "")
	cmd := fmt.Sprintf(
		`docker exec navidrome-cn sqlite3 /data/plugins/cloudmusic/kvstore.db `+
			`"insert or replace into kvstore(key,value,size,created_at,updated_at) values('%s', x'%s', %d, datetime('now'), datetime('now'));"`,
		key, hex.EncodeToString([]byte(value)), len(value))
	_, err := c.Run(cmd)
	return err
}

func mustJSON(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
