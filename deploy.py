"""cloudmusic.ndp 一键部署到飞牛 NAS 的 navidrome-cn。

流程:SFTP 上传 -> docker cp 进插件目录 -> 重新启用(文件变更会重置 enabled)
     -> 重启容器 -> 验证插件加载。

用法: python deploy.py   (需先运行 build.cmd 生成 plugin/cloudmusic.ndp)
"""
import json
import sys
import time

import paramiko

sys.stdout.reconfigure(encoding="utf-8", errors="replace")


def load_config():
    """凭证从 gitignore 的 config.json 读取,避免入库泄露。"""
    try:
        with open("config.json", encoding="utf-8") as f:
            return json.load(f)
    except FileNotFoundError:
        print("[错误] 缺少 config.json:请复制 config.example.json 为 config.json 并填写")
        sys.exit(1)


CFG = load_config()
HOST = CFG["ssh_host"]
USER = CFG["ssh_user"]
PW = CFG["ssh_password"]
LOCAL_NDP = "plugin/cloudmusic.ndp"
REMOTE_TMP = "/tmp/cloudmusic.ndp"


def run(client, cmd, timeout=60):
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", "replace").strip()
    err = stderr.read().decode("utf-8", "replace").strip()
    return out, err


DOCKER_RUN_ENV = """-e ND_DEFAULTLANGUAGE=zh-Hans \
-e ND_SCANNER_EXTRACTOR=ffmpeg \
-e ND_COVERARTPRIORITY=external,embedded \
-e ND_AGENTS=cloudmusic,deezer,lastfm,listenbrainz,apple-music \
-e ND_LYRICSPRIORITY=embedded,.lrc,cloudmusic,nd-lyrics \
-e ND_MUSICFOLDER=/music \
-e ND_DATAFOLDER=/data \
-e ND_CONFIGFILE=/data/navidrome.toml \
-e ND_PORT=4533"""


def ensure_container_env(client):
    """环境变量被飞牛界面重建打回原样时,自动重建容器修复。"""
    out, _ = run(client,
        "docker inspect navidrome-cn --format '{{range .Config.Env}}{{println .}}{{end}}'")
    mounts, _ = run(client, "docker inspect navidrome-cn --format '{{json .HostConfig.Binds}}'")
    ok = ("cloudmusic" in out and "LYRICSPRIORITY" in out.upper()
          and ":rw" in mounts and "1000/音乐" in mounts)
    if ok:
        print("[env] 环境变量与 rw 挂载完整 ✓")
        return
    print("[env] 检测到环境变量缺失(可能被飞牛重建),自动重建容器...")
    out, err = run(client,
        "docker rm -f navidrome-cn >/dev/null 2>&1; "
        f"docker run -d --name navidrome-cn --restart unless-stopped -p 4535:4533 "
        f"-v '/vol1/1000/音乐':/music:rw -v /vol1/@appdata/navidrome-cn-data:/data "
        f"{DOCKER_RUN_ENV} navidrome-cn:test")
    if "error" in out.lower() or err:
        raise RuntimeError(f"容器重建失败: {out} {err}")
    time.sleep(10)
    print("[env] 容器已重建(含 cloudmusic AGENTS 与 LYRICSPRIORITY)")


def sync_toml(client):
    """同步 navidrome.toml 到数据卷(数据卷持久,容器重建不丢)。"""
    sftp = client.open_sftp()
    sftp.put("navidrome.toml", "/tmp/navidrome.toml")
    sftp.close()
    run(client,
        "docker cp /tmp/navidrome.toml navidrome-cn:/data/navidrome.toml && "
        "docker exec navidrome-cn chown root:root /data/navidrome.toml")
    print("[toml] navidrome.toml 已同步到 /data")


def main():
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PW, timeout=8,
                   allow_agent=False, look_for_keys=False)

    try:
        # 0. 环境自检修复 + 配置文件同步
        ensure_container_env(client)
        sync_toml(client)

        # 1. 上传
        sftp = client.open_sftp()
        sftp.put(LOCAL_NDP, REMOTE_TMP)
        sftp.close()

        # 2. 拷进容器 + 权限对齐(文件变更会触发宿主自动发现,但 enabled 会被重置)
        out, err = run(client,
            "docker cp /tmp/cloudmusic.ndp navidrome-cn:/data/plugins/cloudmusic.ndp && "
            "docker exec navidrome-cn chown 1000:0 /data/plugins/cloudmusic.ndp")
        if err:
            print("[失败] 安装:", err)
            return 1

        # 3. 轮询等宿主处理文件变更(它会把 enabled 重置为 0,必须等它处理完再改库;
        #    watcher 的触发时机不定,固定 sleep 不可靠)
        changed = False
        for _ in range(15):
            time.sleep(2)
            out, _ = run(client,
                "docker logs --since 2m navidrome-cn 2>&1 | "
                "grep 'Plugin file changed' | grep cloudmusic | tail -1")
            if out:
                changed = True
                break
        print(f"[{'已' if changed else '未'}检测到文件变更事件]")

        # 4. 启用(含媒体库授权+写权限,供插件写 .lrc)+ 重启 + 验证
        for attempt in (1, 2):
            out, err = run(client,
                "docker exec navidrome-cn sqlite3 /data/navidrome.db "
                "\"update plugin set enabled=1, all_libraries=1, allow_write_access=1 "
                "where id='cloudmusic';\" && "
                "docker restart navidrome-cn")
            if err:
                print("[失败] 启用/重启:", err)
                return 1
            time.sleep(18)
            out, _ = run(client,
                "docker logs --since 1m navidrome-cn 2>&1 | "
                "grep -E 'Loaded plugin.*cloudmusic' | tail -1")
            if out:
                print("[成功] 插件已加载:")
                print(out)
                return 0
            enabled, _ = run(client,
                "docker exec navidrome-cn sqlite3 /data/navidrome.db "
                "\"select enabled from plugin where id='cloudmusic';\"")
            print(f"[第{attempt}轮] 未加载 (enabled={enabled.strip()}),{'重试' if attempt == 1 else '放弃'}")
        print("[警告] 请手动检查: docker logs --since 2m navidrome-cn | grep cloudmusic")
        return 1
    finally:
        client.close()


if __name__ == "__main__":
    sys.exit(main())
