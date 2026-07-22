<p align="center">
  <img src="docs/assets/holt-logo.svg" alt="Holt" width="385">
</p>

<p align="center"><strong>一个独属于你的终端编码智能体。</strong></p>

<p align="center">
  <a href="LICENSE"><img alt="license" src="https://img.shields.io/badge/license-MIT-blue"></a>
  <img alt="Go 1.25+" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white">
  <img alt="25+ providers" src="https://img.shields.io/badge/providers-25+-34E2EA">
  <br>
  <a href="README.md">English</a> | <strong>中文</strong>
</p>

Holt 是一个用于本地终端的 AI 编码智能体。它可以检查仓库、编辑文件、运行命令、使用浏览器/终端辅助工具，并在你选择模型和权限级别的同时保持持久的本地会话。

```bash
holt
holt exec "修复 ./pkg 中失败的测试"
holt exec --output-format stream-json < turns.jsonl
```

## 为什么选择 Holt

- **使用你想要的模型。** 支持 OpenAI、Anthropic、Gemini、Groq、OpenRouter、DeepSeek、Mistral、xAI、Qwen、Kimi、GitHub Models、Ollama、LM Studio，或任何 OpenAI/Anthropic 兼容端点。
- **保持控制权。** 文件写入、Shell 命令、网络访问和工作区外写入都经过 Holt 的权限和沙箱策略。
- **在终端中工作。** TUI 具有模型/提供商选择器、图片输入、斜杠命令、实时计划/工具渲染、回滚滚动、主题以及恢复/分叉支持。
- **无 TUI 也能工作。** `holt exec` 可脚本化，支持文本/JSON/stream-JSON I/O、隔离的工作树、规范优先运行，以及用于 CI 的有意义的退出码。
- **保持上下文本地化。** 会话存储在磁盘上，可搜索、可恢复，且 Holt 不会作为遥测数据上传。
- **可扩展。** 使用 MCP 服务器、技能、插件、钩子和来自同一 CLI 的专业子智能体。

## 安装

### npm

```bash
npm install -g @askspecter/holt
holt
```

npm 包安装一个小型包装器以及与你平台匹配的 Holt 二进制文件（从 GitHub Releases 获取）。支持 Linux、macOS 和 Windows 的 x64 和 arm64。

### Bun

Bun 默认不运行依赖的生命周期脚本，因此获取 Holt 二进制文件的 `postinstall` 会被跳过，首次运行会失败并显示 `No native binary found next to the npm wrapper`。

使用 Bun 安装后，通过以下两种方式之一获取二进制文件：

```bash
# 选项 A：手动运行安装程序
bun add @askspecter/holt
node node_modules/@askspecter/holt/scripts/postinstall.mjs

# 选项 B：允许 postinstall 在安装时运行
# 添加到你的 package.json:  "trustedDependencies": ["@askspecter/holt"]
bun add @askspecter/holt
```

对于全局安装（`bun add -g @askspecter/holt`），请对全局安装路径使用选项 A，或使用下面的安装脚本。

### 安装脚本

Linux/macOS：

```bash
curl -fsSL https://raw.githubusercontent.com/askspecter/holt/main/scripts/install.sh | bash
```

Windows PowerShell：

```powershell
irm https://raw.githubusercontent.com/askspecter/holt/main/scripts/install.ps1 | iex
```

### 从源码构建

源码构建需要 Go 1.25+。

```bash
git clone https://github.com/askspecter/holt.git
cd holt
go run ./cmd/holt
```

发布安装器和 npm 包需要已发布的 GitHub Release 资源。如果你在首次公开发布之前进行测试，请从源码构建：

```bash
go build -o holt ./cmd/holt
```

在 Linux 上，如果你需要原生沙箱，还需要构建沙箱辅助程序：

```bash
go build -o holt-linux-sandbox ./cmd/holt-linux-sandbox
go build -o holt-seccomp ./cmd/holt-seccomp   # 可选的兼容性包装器
```

将 `holt` 和 `holt-linux-sandbox` 放在 `PATH` 上的同一目录中（`~/.local/bin` 是一个好的默认选择）。macOS 不需要额外的辅助二进制文件。Windows 源码构建可以使用主 `holt.exe` 作为沙箱辅助程序；发布包仍然附带独立的 Windows 辅助可执行文件。

更多安装细节：[docs/INSTALL.md](docs/INSTALL.md)。

## 首次运行

启动 TUI：

```bash
holt
```

设置向导帮助你选择提供商和模型。你也可以从命令行配置提供商：

```bash
holt setup
holt providers list
holt models list
holt doctor
```

对于 API 提供商，在设置之前设置匹配的环境变量或在向导中输入密钥：

```bash
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=...
export GEMINI_API_KEY=...
export LONGCAT_API_KEY=...
```

要直接配置美团 LongCat（LongCat-2.0），运行：

```bash
holt providers setup longcat --set-active
```

对于本地模型，运行 Ollama 或 LM Studio，然后使用 `holt setup` 或 `holt providers detect`。

## 日常使用

### 交互式 TUI

```bash
holt
```

常用控制：

| 控制 | 操作 |
|---|---|
| `Enter` | 发送提示 |
| `/` | 打开斜杠命令建议 |
| `Shift+Tab` | 切换权限模式 |
| `Ctrl+B` | 显示/隐藏侧边栏 |
| `Ctrl+C` | 取消或退出 |

常用斜杠命令：

| 命令 | 用途 |
|---|---|
| `/model`、`/provider` | 切换活动模型/提供商 |
| `/spec`、`/plan` | 在构建之前起草和审查计划 |
| `/image` | 为视觉模型附加图片 |
| `/resume`、`/rewind` | 继续或回滚本地会话 |
| `/compact`、`/context` | 管理上下文使用 |
| `/permissions`、`/tools` | 检查可用工具和策略 |
| `/add-dir` | 为此会话授予额外的写入目录 |
| `/theme`、`/doctor`、`/config` | 调整外观和检查设置 |

### 无头 `exec` 模式

```bash
holt exec "解释 internal/agent/loop.go"
holt exec --model claude-sonnet-4.5 "重构配置加载器"
holt exec --use-spec "为 API 客户端添加速率限制"
holt exec --worktree "在隔离的工作树中尝试迁移"
holt exec --resume
holt exec --fork <session-id> "尝试另一种方法"
```

编程使用：

```bash
holt exec --input-format stream-json --output-format stream-json < turns.jsonl
```

Stream-JSON 协议文档在 [docs/STREAM_JSON_PROTOCOL.md](docs/STREAM_JSON_PROTOCOL.md)。

## 安全模型

Holt 旨在使副作用可见。

- 工作区读取默认允许。
- 文件写入限制在工作区内，除非你授予其他目录。
- Shell 命令、网络访问、破坏性命令和提权操作需要权限授权。
- `--add-dir <path>` 和 `/add-dir <path>` 授予额外的写入根目录，而不会给智能体整个文件系统。
- 不安全/自主模式是显式选择加入。
- 在 Holt 控制的界面上，密钥会从工具输出和日志中被脱敏。

示例：

```bash
holt --add-dir ../docs-site
holt exec --add-dir ../shared "更新两个仓库"
```

沙箱行为可以通过以下方式检查：

```bash
holt sandbox policy
holt sandbox grants list
```

## Web 和本地控制

Holt 包含本地文件/搜索/编辑/Shell 工具、用于公共 URL 的 `web_fetch`，以及用于额外工具的 MCP 支持。

对于本地开发服务器，使用 Shell 命令（如通过 `exec_command` 的 `curl`），这样正常的沙箱和权限策略就会生效。长时间运行的命令保持附加到后台终端会话，可以从 TUI 中列出或停止。

npm 包还包含本地浏览器/终端工具使用的浏览器和终端辅助包。源码构建可以在它们位于 `PATH` 上或在 Holt 的本地控制设置中配置时使用相同的辅助工具。

## 常用命令

```text
holt                  交互式 TUI
holt exec             一次性或脚本化智能体运行
holt setup            首次运行提供商设置
holt auth             支持提供商的 OAuth/登录辅助
holt models           模型注册表和能力
holt providers        提供商配置和检测
holt doctor           设置、密钥和连接检查
holt context          上下文预算报告
holt repo-map         确定性仓库映射
holt repo-info        本地仓库摘要
holt search | find    搜索本地会话历史
holt sessions         检查、恢复、分叉和回滚会话
holt spec             管理规范模式草稿
holt specialist       管理专业子智能体
holt skills           管理 Markdown 指令技能
holt plugins          管理插件
holt hooks            管理生命周期钩子
holt mcp              管理 MCP 服务器和工具
holt serve --mcp      通过 MCP stdio 暴露 Holt 工具
holt sandbox          检查沙箱策略和授权
holt worktrees        准备隔离的 Git 工作树
holt verify           检测和运行本地验证检查
holt changes          检查和提交本地 Git 变更
holt usage            Token 使用量和估算成本
holt cron             定时智能体任务
holt update           检查更新版本
```

## 外观和无障碍

| 控制 | 效果 |
|---|---|
| `NO_COLOR=<任意值>` | 禁用颜色输出 |
| `HOLT_THEME=<名称>` | 选择启动主题（`auto`、`dark`、`light`，或颜色主题如 `dracula`、`nord`、`gruvbox`、`tokyo-night`、`catppuccin`、`one-dark`、`solarized-dark`、`rose-pine`、`everforest`、`solarized-light`） |
| `--theme <名称>` | 从 CLI 选择 TUI 主题（相同名称） |
| `/theme` | 在 TUI 中打开主题选择器（实时预览；`/theme <名称>` 直接切换） |
| `HOLT_NO_FADE=1` | 禁用流式淡入动画 |

含义不仅仅依赖于颜色；差异、权限和状态也使用文本或标记符号。

## 开发

```bash
go test ./...
go run ./cmd/holt-release build
go run ./cmd/holt-release smoke
go run ./cmd/holt-perf-bench
```

交叉编译示例：

```bash
go run ./cmd/holt-release build --goos linux --goarch amd64
go run ./cmd/holt-release build --goos windows --goarch amd64 --output dist/holt.exe
```

## 文档

- [安装](docs/INSTALL.md)
- [更新流程](docs/UPDATE.md)
- [Stream-JSON 协议](docs/STREAM_JSON_PROTOCOL.md)
- [专家](docs/SPECIALISTS.md)
- [GitHub Action](docs/GITHUB_ACTION.md)
- [基准测试](docs/BENCHMARK.md)
- [性能](docs/PERFORMANCE.md)
- [智能体评估](docs/AGENT_EVALS.md)

## 贡献

欢迎贡献。阅读 [CONTRIBUTING.md](CONTRIBUTING.md)，运行相关测试，然后提交一个聚焦的拉取请求。

安全报告应遵循 [SECURITY.md](SECURITY.md)。

## 许可证

Holt 基于 [MIT 许可证](LICENSE) 发布。
