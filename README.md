# Navidrome 网易云增强插件

为 [Navidrome](https://www.navidrome.org) 音乐服务器补全网易云音乐元数据的插件(.ndp / WASM)+ 主机侧标签工具。

## 功能

| 功能 | 形式 | 说明 |
|---|---|---|
| 歌词 | 插件 | 智能评分匹配(歌名/歌手/原唱/专辑),LRC 带时间轴,自动合并网易云翻译 |
| 专辑封面 | 插件 | 匹配后经 song/detail 补全(搜索接口不返回封面) |
| 歌手头像/简介 | 插件 | 网易云歌手数据 |
| 专辑简介 | 插件 | 网易云专辑 Description |
| 流派标签 | tagger 工具 | 推断引擎(EP/单曲/精选集/现场/翻唱/DJ/原声带…),ffmpeg 流拷贝写入文件,不重编码 |

## 智能匹配评分

搜索结果按评分选最优,避免"晴天"匹配到一堆翻唱:

```
歌曲名完全一致 +50    歌手完全一致 +50
歌曲名包含     +20    歌手包含     +25
标注原唱的翻唱 +30     专辑匹配     +20/10
```

原版被网易云版权过滤时,自动回退到标注"原唱"的翻唱;歌词匹配低于 50 分不返回,宁缺毋滥。

## 构建

官方 Go 工具链即可(无需 TinyGo),要求 Go 1.25+:

```bash
cd plugin
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm .
python -m zipfile -c cloudmusic.ndp plugin.wasm manifest.json
```

或直接运行项目根目录的 `build.cmd`。

## 部署

1. 下载 [plugin/cloudmusic.ndp](plugin/cloudmusic.ndp)(仓库内直接提供最新构建;也可自己 `build.cmd` 构建),放入 Navidrome 的插件目录(`<DataFolder>/plugins/`)
2. 配置环境变量(容器):

```text
ND_AGENTS=cloudmusic,deezer,lastfm,listenbrainz,apple-music
ND_LYRICSPRIORITY=embedded,.lrc,cloudmusic,nd-lyrics
```

3. 重启后网页启用插件(或 `update plugin set enabled=1 where id='cloudmusic'`)

一键部署(自动上传+启用+验证,见 `deploy.py`):

```bash
cp config.example.json config.json   # 填入你的服务器与SSH凭证
python deploy.py
```

## 标签补全工具

给无流派的专辑推断并写入 GENRE 标签。默认 dry-run,`-apply` 实际写入:

```bash
go run ./tagger -limit 50          # 查看计划
go run ./tagger -limit 50 -apply   # 写入(临时 rw 挂载 -> ffmpeg 写标签 -> 恢复 ro)
```

标签修复(只处理库里 `[Unknown Artist]` 的文件,解析"编号.歌名 歌手 (DJ版)"等
特殊命名 → 网易云验证 → 写回歌名/歌手/专辑):

```bash
go run ./tagger -fix-tags -limit 1000 -apply
```

> 网易云无匿名流派接口,标签来自推断(专辑 type + 曲名/专辑名特征规则,含繁体变体);
> 无可靠数据源的普通专辑不打标。`tagger/rules.go` 可自行扩展规则。

## 手动匹配工具

自动匹配不了的歌(网易云搜不到/名字差太远),手动指定对应关系:

```bash
go run ./match -q "歌名 歌手"                # 交互:选本地歌 → 选网易云候选
go run ./match -q "黄昏" -local 1 -netease 2 # 非交互:第1首本地歌 ↔ 网易云第2条
```

选好后自动:写 `.lrc` 伴生文件(直写宿主机)+ 更新插件歌词缓存(立即生效)。

## 共用包

```
navid/    Subsonic API 客户端
nassh/    NAS SSH 执行器(真实路径查询/.lrc 写入/插件缓存写入)
parser/   文件名解析器链
```

## 目录结构

```
plugin/        .ndp 插件(WASM,运行在 Navidrome 内)
  netease/     网易云 API 客户端(host.HTTPSend 版)+ 匹配器 + 测试
  agent.go     实现 Lyrics/MetadataAgent 各 Provider 接口
tagger/        主机侧标签写入工具(含推断规则与测试)
cloudmusic/    独立的网易云客户端包(命令行/工具复用)
build.cmd      一键构建
deploy.py      一键部署
config.json    本地凭证(gitignore,勿提交)
```

## 缓存架构(哪些数据不会重复请求)

| 数据 | 缓存层 | 说明 |
|---|---|---|
| 封面/头像 | Navidrome 服务端 | 首次取回即持久缓存 |
| 歌手/专辑简介 | Navidrome 服务端 | 库缓存,含负缓存 |
| 标签 | 音频文件内 | tagger 直接写入,永久 |
| 歌词 | 三层 | ①文件内嵌/.lrc 优先命中(不发起请求) ②Navidrome 服务端缓存结果 ③插件 KVStore 缓存歌词全文与"确认无歌词"负缓存(网络失败不缓存,避免抖动被记住) |
| 网易云 ID 匹配 | 插件 KVStore | 歌手/专辑 ID 解析结果持久缓存 |

插件 KVStore 上限在 manifest 里声明(默认 16MB,约数千首歌词)。

## 配置持久化

飞牛等 NAS 的 Docker 管理界面重建容器会打回手工设置的环境变量。本项目:

- `navidrome.toml` 同步到数据卷 `/data`(容器重建不丢), LyricsPriority 等以文件为准
- `deploy.py` 每次部署自动检测环境变量缺失并重建容器修复

## 已知限制

- 插件 HTTP 走宿主 `extism:host/user`,函数集固定(如无 kvstore_setwithttl)
- 改 .ndp 后宿主会重置插件启用状态,需重新启用(deploy.py 已自动处理)
- FLAC/MP3 以外格式 tagger 跳过

## License

MIT
