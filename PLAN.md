# music-dl-cn Go 重写计划

## 目标

将原 PHP/Laravel Zero 项目重写为 Go，产出：
- 跨平台单二进制 CLI（Linux / macOS / Windows）
- 兼容原有交互式功能（搜索、多选、下载）
- 新增 `--json` 输出模式，可直接输出搜索结果和 URL

## 架构

```
music-dl-cn/
├── cmd/
│   └── root.go              # cobra CLI 入口
├── internal/
│   ├── api/                 # 音乐平台 API 层
│   │   ├── interface.go     # MusicSource interface
│   │   ├── netease.go       # 网易云音乐
│   │   ├── kugou.go         # 酷狗音乐
│   │   └── tencent.go       # 腾讯音乐
│   ├── downloader/          # HTTP 下载 + 进度条
│   ├── ui/                  # 交互式 CLI 组件
│   └── models/              # Song, SearchResult 等结构体
├── main.go
├── go.mod
└── PLAN.md
```

## 依赖选型

| 包 | 用途 |
|----|------|
| `cobra` | CLI 框架（命令、flags） |
| `bubbletea` + `lipgloss` | 交互式 UI |
| `schollz/progressbar/v3` | 下载进度条 |
| `go-beeep/beep` | 桌面通知 |
| 标准库 `net/http`, `encoding/json`, `crypto/*` | HTTP + JSON + 加密 |

## 功能对照

| 功能 | PHP | Go |
|------|-----|----|
| 搜索（网易/酷狗/腾讯） | ✅ | ✅ |
| 交互式 source 选择 | ✅ | ✅ |
| 交互式多选歌曲 | ✅ | ✅ |
| 并发查询 | ✅ fork/process | ✅ goroutines |
| 下载 + 进度条 | ✅ | ✅ |
| JSON 输出 | ❌ | ✅ 新增 |
| 桌面通知 | ✅ | ✅ |
| 自更新 | ✅ | ✅ |
| 单二进制 | ❌ 需 PHP | ✅ |
| 跨平台 | 有限 | ✅ |

## JSON 输出示例

```bash
# 搜索，输出 JSON
music-dl search --keyword "刘德华" --sources netease --json

# 获取下载 URL
music-dl url --id 123456 --source netease --json
```

```json
{
  "songs": [
    {
      "id": "123456",
      "name": "忘情水",
      "artist": "刘德华",
      "album": "真我的风采",
      "source": "netease",
      "url": "https://...",
      "size": "8.2MB",
      "bitrate": 320
    }
  ]
}
```

## 实现阶段

### Phase 1：项目骨架 ✅
- Go module 初始化
- 目录结构
- models 定义

### Phase 2：网易云 API ✅（当前）
- search(keyword) → []Song
- url(id) → string
- AES+RSA 加密逻辑移植
- 测试脚本：搜索"刘德华"

### Phase 3：酷狗 + 腾讯 API
- 同 Phase 2，移植对应加密逻辑

### Phase 4：下载器
- HTTP streaming
- 进度条
- 403 自动重试

### Phase 5：交互式 CLI
- 完整交互流程
- --json 输出模式

### Phase 6：跨平台构建
- Makefile
- Linux / macOS / Windows 二进制
