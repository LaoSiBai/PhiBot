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

	logFile, err := os.OpenFile("phibot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.SetOutput(logFile)
	} else {
		fmt.Fprintf(os.Stderr, "无法打开日志文件: %v\n", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	b, err := bot.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 Bot 失败: %v\n", err)
		os.Exit(1)
	}

	logger.Info("PhiBot 启动", "nickname", cfg.Bot.Nickname, "model", cfg.LLM.Model)

	m := tui.NewModel(b)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI 运行失败: %v\n", err)
		os.Exit(1)
	}
}
