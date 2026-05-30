# PhiBot

一个基于 Go 的 AI 聊天机器人，使用 Bubble Tea 构建终端界面。

## 特性

- 🚀 **流式输出** - LLM 回复逐字显示
- 🎨 **终端 UI** - 基于 Bubble Tea 的美观界面
- 🔌 **OpenAI 兼容** - 支持所有 OpenAI 兼容 API
- 📊 **调试面板** - 实时查看事件流和日志
- 🎯 **事件驱动** - 基于 EventBus 的架构设计

## 快速开始

### 环境要求

- Go 1.22+

### 安装

```bash
git clone https://github.com/phibot/phibot.git
cd phibot
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
base_url = "https://api.openai.com/v1"  # 或其他兼容接口
api_key = "sk-your-api-key-here"
model = "gpt-4o-mini"
temperature = 0.7
max_tokens = 2048

[tui]
theme = "default"
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

- **输入消息** - 直接输入并按 Enter 发送
- **切换视图** - 按 `Tab` 在聊天和调试面板间切换
- **退出** - 按 `Ctrl+C`

## 项目结构

```
phibot/
├── cmd/phibot/          # 程序入口
├── configs/             # 配置文件
├── internal/
│   ├── bot/            # Bot 核心 + EventBus
│   ├── config/         # 配置加载
│   ├── llm/            # LLM Provider 接口
│   ├── logger/         # 日志封装
│   └── platform/tui/   # Bubble Tea 界面
└── plan.md             # 架构设计文档
```

## 开发计划

- [x] **P0** - 项目骨架 + 配置 + 日志 + OpenAI 对话 + TUI 基础界面
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
| TUI | Bubble Tea + Lipgloss |
| LLM | OpenAI 兼容接口 |
| 配置 | TOML |
| 日志 | charmbracelet/log |

## License

MIT
