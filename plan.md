# PhiBot 架构设计

## 技术选型

| 层面 | 技术 |
|------|------|
| **语言** | Go 1.22+ |
| **TUI** | Bubble Tea + Lipgloss + Bubbles |
| **LLM** | OpenAI 兼容接口（`openai-go` 官方 SDK） |
| **数据库** | SQLite（`modernc.org/sqlite` 纯 Go 驱动） |
| **向量检索** | SQLite + 余弦相似度（纯 Go，无外部依赖） |
| **消息平台** | TUI + OneBot 11（WebSocket，对接 NapCat） |
| **MCP** | `mark3labs/mcp-go` |
| **配置** | TOML（`pelletier/go-toml`） |
| **日志** | `charmbracelet/log`（与 Bubble Tea 生态一致） |

## 项目结构

```
phibot/
├── cmd/phibot/
│   └── main.go                    # 入口
├── internal/
│   ├── bot/                       # Bot 核心生命周期
│   │   ├── bot.go                 # 启动、事件循环、优雅关闭
│   │   └── event_bus.go           # 事件总线（发布/订阅）
│   │
│   ├── chat/                      # 聊天管理
│   │   ├── session.go             # 会话（群聊/私聊）
│   │   ├── manager.go             # 会话管理器
│   │   ├── message.go             # 统一消息模型
│   │   └── context.go             # 上下文窗口管理（滑动窗口 + 摘要）
│   │
│   ├── llm/                       # LLM 接入层
│   │   ├── provider.go            # Provider 接口
│   │   ├── openai.go              # OpenAI 兼容实现
│   │   ├── tool.go                # Function Calling / Tool Use
│   │   └── token.go               # Token 计数
│   │
│   ├── memory/                    # 记忆系统
│   │   ├── store.go               # 存储接口
│   │   ├── sqlite.go              # SQLite 实现
│   │   ├── embedding.go           # Embedding 服务
│   │   ├── vector.go              # 向量检索
│   │   ├── person.go              # 用户画像
│   │   └── migration.go           # 数据库迁移
│   │
│   ├── platform/                  # 平台抽象层
│   │   ├── platform.go            # Platform 接口
│   │   ├── router.go              # 消息路由
│   │   ├── tui/                   # Bubble Tea TUI (鼠标驱动)
│   │   │   ├── app.go             # 主 Model (鼠标/标签/流式/吐司/滑块)
│   │   │   └── styles.go          # 极简调色板 (白/暗灰/冰蓝)
│   │   └── onebot/                # OneBot 11 协议
│   │       ├── client.go          # WebSocket 客户端
│   │       ├── event.go           # 事件解析
│   │       └── api.go             # API 调用（发消息等）
│   │
│   ├── plugin/                    # 插件系统
│   │   ├── manager.go             # 插件生命周期
│   │   ├── plugin.go              # 插件接口定义
│   │   ├── hook.go                # Hook 点（消息前/后处理等）
│   │   └── registry.go            # 插件注册表
│   │
│   ├── mcp/                       # MCP 集成
│   │   ├── client.go              # MCP 客户端管理
│   │   └── bridge.go              # MCP Tool ↔ LLM Function Calling 桥接
│   │
│   ├── emoji/                     # 表情包系统
│   │   ├── manager.go             # 表情管理
│   │   └── matcher.go             # 表情匹配
│   │
│   ├── prompt/                    # Prompt 模板
│   │   ├── manager.go             # 模板管理
│   │   ├── system.go              # System prompt 构建
│   │   └── templates/             # 模板文件
│   │
│   └── config/                    # 配置
│       ├── config.go              # 配置结构体
│       └── defaults.go            # 默认值
│
├── pkg/                           # 可复用的公共包
│   └── msgutil/
│
├── configs/                       # 配置文件模板
│   └── config.example.toml
├── prompts/                       # Prompt 模板文件
├── go.mod
├── go.sum
└── Makefile
```

## 核心设计

### 1. 事件驱动架构

```
Platform(TUI/OneBot) → EventBus → ChatManager → LLM → Response → Platform
                                  ↓
                            Memory Store
```

- `EventBus` 是核心中枢，所有消息/事件通过它分发
- Hook 点：`OnMessageReceive` → `OnPreProcess` → `OnLLMRequest` → `OnResponse` → `OnMessageSend`

### 2. 平台抽象

```go
type Platform interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    SendMessage(ctx context.Context, msg Message) error
    Events() <-chan Event  // 消息事件流
}
```

TUI 和 OneBot 都实现这个接口，`Router` 负责将平台事件转为统一消息格式。

### 3. Bubble Tea TUI 布局

**鼠标驱动极简界面** — 无边框、无状态栏、极致留白：

```
        P H I B O T

    Dashboard    Local Chat    Settings

    ─────────────────────────────────────

    Engine        openai
    Memory        2048 Tokens
    Uptime        ● Running (00:05:23)    
    Tokens/Sec    ■■■■░░░░░░░░░░░ 42.3

    ● Engine Active  │  Mouse Driver Mode
```

- **纯鼠标交互**：点击切换标签页、聚焦输入、拖拽滑块、滚轮滚动
- **悬浮反馈**：鼠标悬停元素高亮为白色，离开恢复暗灰
- **错误吐司**：错误信息短暂显示在底栏，3秒后自动清除
- **无键盘导航**：仅 Enter 发送消息、Ctrl+C 退出
- 调试面板已移除，日志写入 `phibot.log` 文件

### 4. 记忆系统

- **短期记忆**：当前会话的上下文窗口（滑动窗口 + 超限时摘要压缩）
- **长期记忆**：SQLite 存储 + Embedding 向量检索
- **用户画像**：逐步积累用户偏好、性格特征

### 5. 插件系统

```go
type Plugin interface {
    Name() string
    Version() string
    Init(bot *bot.Bot) error
    Hooks() []Hook
    Start(ctx context.Context) error
    Stop() error
}
```

插件方案：**外部进程 + JSON-RPC**

- 插件作为独立进程运行，通过 JSON-RPC 与主程序通信
- 语言无关，最灵活，类似 MaiBot 方案

### 6. MCP 工具调用

通过 `mark3labs/mcp-go` 接入 MCP Server，将 MCP Tool 自动桥接为 LLM Function Calling 参数。

## 开发路线

| 阶段 | 内容 | 优先级 |
|------|------|--------|
| **P0** | 项目骨架 + 配置 + 日志 + OpenAI 对话 | 最高 |
| **a1** | 基础TUI聊天界面 + 调试面板 + 流式输出 | ✓ |
| **a2** | 鼠标驱动TUI + 3标签页 + Dashboard/Settings + 温度滑块 + 错误吐司 | ✓ |
| **P1** | 会话管理 + 上下文窗口 + 记忆系统基础 | 高 |
| **P2** | OneBot 平台接入 + 消息路由 | 高 |
| **P3** | Prompt 模板 + 自然对话风格调优 | 中 |
| **P4** | MCP 工具调用 | 中 |
| **P5** | 插件系统 + 表情包 | 中 |
| **P6** | 用户画像 + 表达学习 | 低 |

## 已确认决策

1. **插件方案**：外部进程 + JSON-RPC
2. **向量检索**：SQLite + 余弦相似度（纯 Go，无外部依赖）
3. **TUI 交互**：纯鼠标驱动，无键盘导航（仅 Enter 发送、Ctrl+C 退出）
4. **极简美学**：暗色终端原生风格，白/暗灰/冰蓝调色板，无边框无状态栏，极致留白
5. **IME 光标**：绝不将 `textinput.View()` 包在 `lipgloss.Style.Render()` 里，用 `PromptStyle`/`TextStyle` 代替
6. **日志输出**：写入 `phibot.log` 文件，避免 stderr 污染 AltScreen
7. **错误显示**：底栏临时吐司（3秒自动清除），不使用模态弹窗
