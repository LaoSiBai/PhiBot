package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/phibot/phibot/internal/bot"
	"github.com/phibot/phibot/internal/config"
	"github.com/phibot/phibot/internal/logger"
	"github.com/phibot/phibot/internal/platform/tui"
)

func main() {
	configPath := flag.String("config", "configs/config.toml", "配置文件路径")
	debug := flag.Bool("debug", false, "启用调试日志")
	flag.Parse()

	if *debug {
		logger.Init(log.DebugLevel)
	} else {
		logger.Init(log.InfoLevel)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "sk-your-api-key-here" {
		fmt.Fprintf(os.Stderr, "请在配置文件中设置 LLM API Key: %s\n", *configPath)
		os.Exit(1)
	}

	b, err := bot.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 Bot 失败: %v\n", err)
		os.Exit(1)
	}

	logger.Info("PhiBot 启动", "nickname", cfg.Bot.Nickname, "model", cfg.LLM.Model)

	m := tui.NewModel(b)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI 运行失败: %v\n", err)
		os.Exit(1)
	}
}
