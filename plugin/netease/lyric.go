package netease

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Lyric 拉取 LRC 歌词,无歌词返回空字符串
func Lyric(id int) (string, error) {
	orig, _, err := LyricWithTranslation(id)
	return orig, err
}

// LyricWithTranslation 同时取原文与翻译(网易云 tv=1)
func LyricWithTranslation(id int) (string, string, error) {
	body, err := httpGet(fmt.Sprintf("https://music.163.com/api/song/lyric?id=%d&lv=1&kv=1&tv=1", id))
	if err != nil {
		return "", "", err
	}
	if body == nil {
		return "", "", nil
	}
	var result struct {
		Lrc struct {
			Lyric string `json:"lyric"`
		} `json:"lrc"`
		TLrc struct {
			Lyric string `json:"lyric"`
		} `json:"tlyric"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}
	return result.Lrc.Lyric, result.TLrc.Lyric, nil
}

// MergeTranslation 将翻译按时间戳合并进原文:同行显示 "原文 翻译"
func MergeTranslation(orig, trans string) string {
	if orig == "" || trans == "" {
		return orig
	}
	transMap := map[string]string{}
	for _, line := range strings.Split(trans, "\n") {
		ts, text := splitLrcLine(line)
		if ts != "" && text != "" {
			transMap[ts] = text
		}
	}
	var b strings.Builder
	for _, line := range strings.Split(orig, "\n") {
		ts, text := splitLrcLine(line)
		if ts != "" && text != "" {
			if t, ok := transMap[ts]; ok && t != text {
				b.WriteString(ts)
				b.WriteString(text)
				b.WriteString(" ")
				b.WriteString(t)
				b.WriteString("\n")
				continue
			}
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// lyricNoise 歌词噪声行关键词:版权水印/平台标识等,命中整行剔除
var lyricNoise = []string{
	"博纳", "正版授权", "授权方", "未经许可", "未经授权",
	"翻录必究", "版权所有", "著作权", "官方授权",
}

// FilterNoise 剔除歌词中的噪声行("博纳正版授权"等版权水印)
func FilterNoise(lyric string) string {
	lines := strings.Split(lyric, "\n")
	out := lines[:0]
	for _, l := range lines {
		noisy := false
		for _, k := range lyricNoise {
			if strings.Contains(l, k) {
				noisy = true
				break
			}
		}
		if !noisy {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// splitLrcLine 拆出 "[mm:ss.xx]" 时间戳与文本;非歌词行返回空时间戳
func splitLrcLine(line string) (string, string) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "[") {
		return "", line
	}
	end := strings.Index(line, "]")
	if end < 0 {
		return "", line
	}
	ts := line[:end+1]
	rest := strings.TrimSpace(line[end+1:])
	// 时间戳必须是数字冒号格式,避免 [00:00.000]词：xxx 之外的元数据标签被当时间轴
	if !isTimestamp(ts) {
		return "", line
	}
	return ts, rest
}

func isTimestamp(ts string) bool {
	inner := strings.TrimSuffix(strings.TrimPrefix(ts, "["), "]")
	if inner == "" {
		return false
	}
	digits := 0
	for _, c := range inner {
		if (c >= '0' && c <= '9') || c == ':' || c == '.' {
			if c >= '0' && c <= '9' {
				digits++
			}
			continue
		}
		return false
	}
	return digits >= 3 && strings.Contains(inner, ":")
}
