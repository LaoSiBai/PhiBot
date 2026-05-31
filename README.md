# PhiBot

基于 Go 的 AI 聊天机器人，纯鼠标驱动的电影级极简 TUI。

## 特性

- **流式输出** - LLM 回复逐字流式显示
- **鼠标驱动 TUI** - 纯鼠标交互，无键盘导航
- **三标签页** - Dashboard / Local Chat / Settings
- **极简美学** - 暗色终端原生风格，无边框，极致留白
- **悬浮反馈** - Hover 高亮、3秒自动清除的错误吐司
- **温度滑块** - 鼠标拖拽实时调节 Temperature
- **实时 TPS** - Dashboard 显示真实 Tokens/Sec
- **OpenAI 兼容** - 支持所有 OpenAI 兼容 API
- **事件驱动** - 基于 EventBus 的架构设计

## 快速开始

### 环境要求

- Go 1.22+

### 安装

```bash
git clone https://github.com/LaoSiBai/PhiBot.git
cd PhiBot
```

### 配置

复制配置文件模板并填入你的 API Key：

```bash
cp configs/config.example.toml configs/config.toml
```

编辑 `configs/config.toml`：

```toml
[bot]
nickname = "PhiBot"
data_dir = "."

[llm]
provider = "openai"
base_url = "https://api.openai.com/v1"
api_key = "sk-your-api-key-here"
model = "gpt-4o-mini"
temperature = 0.7
max_tokens = 2048
```

### 运行

```bash
go run ./cmd/phibot
```

或编译后运行：

```bash
go build -o phibot ./cmd/phibot
./phibot
```

## 使用方法

- **鼠标点击** - 切换标签页、聚焦输入框、点击设置项
- **鼠标悬停** - 元素高亮为白色
- **输入消息** - 点击输入框后输入，按 Enter 发送
- **鼠标滚轮** - 在聊天页上下滚动消息
- **退出** - `Ctrl+C`

## 项目结构

```
phibot/
├── cmd/phibot/          # 程序入口 (AltScreen + MouseAllMotion)
├── configs/             # 配置文件模板
├── internal/
│   ├── bot/            # Bot 核心 + EventBus + Stats
│   ├── config/         # TOML 配置加载
│   ├── llm/            # LLM Provider (OpenAI 流式)
│   ├── logger/         # 日志 (写入 phibot.log)
│   └── platform/tui/   # Bubble Tea 界面
│       ├── app.go      # 主 Model (鼠标事件/标签/吐司/流式)
│       └── styles.go   # 极简调色板
└── plan.md             # 架构设计文档
```

## 开发计划

- [x] **P0** - 项目骨架 + 配置 + 日志 + OpenAI 对话
- [x] **a1** - 基础 TUI 聊天界面 + 调试面板 + 流式输出
- [x] **a2** - 鼠标驱动TUI + 3标签页 + Dashboard/Settings + 温度滑块 + 错误吐司
- [ ] **P1** - 会话管理 + 上下文窗口 + 记忆系统基础
- [ ] **P2** - OneBot 平台接入 + 消息路由
- [ ] **P3** - Prompt 模板 + 自然对话风格调优
- [ ] **P4** - MCP 工具调用
- [ ] **P5** - 插件系统 + 表情包
- [ ] **P6** - 用户画像 + 表达学习

详见 [plan.md](plan.md)

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.22+ |
| TUI | Bubble Tea + Lipgloss + Bubbles |
| LLM | OpenAI 兼容接口 (sashabaranov/go-openai) |
| 配置 | TOML (pelletier/go-toml) |
| 日志 | charmbracelet/log (写入文件) |

## License

MIT
