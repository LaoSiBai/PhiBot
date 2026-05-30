package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/phibot/phibot/internal/config"
	"github.com/phibot/phibot/internal/llm"
	"github.com/phibot/phibot/internal/logger"
)

type BotStats struct {
	LastTPS float64
}

type Bot struct {
	Config   *config.Config
	EventBus *EventBus
	LLM      llm.Provider
	history  []llm.Message
	Stats    BotStats
}

func New(cfg *config.Config) (*Bot, error) {
	b := &Bot{
		Config:   cfg,
		EventBus: NewEventBus(),
		history: []llm.Message{
			{Role: llm.RoleSystem, Content: "你是一个友好的AI助手。"},
		},
		Stats: BotStats{},
	}

	_ = b.initLLM()

	b.EventBus.Subscribe(EventMessageReceive, b.handleMessage)

	return b, nil
}

func (b *Bot) initLLM() error {
	if b.Config.LLM.APIKey == "" || b.Config.LLM.APIKey == "sk-your-api-key-here" {
		b.LLM = nil
		return nil
	}
	provider, err := llm.NewOpenAIProvider(b.Config.LLM)
	if err != nil {
		b.LLM = nil
		return err
	}
	b.LLM = provider
	return nil
}

func (b *Bot) ReloadLLM() error {
	return b.initLLM()
}

func (b *Bot) handleMessage(event Event) {
	if event.Message == nil {
		return
	}

	msg := event.Message
	logger.Info("received message", "from", msg.SenderName, "content", msg.Content)

	if b.LLM == nil {
		b.EventBus.Publish(Event{
			Type: EventError,
			Data: fmt.Errorf("LLM 未配置，请先在设置页面填写 API Key"),
		})
		return
	}

	b.history = append(b.history, llm.Message{
		Role:    llm.RoleUser,
		Content: msg.Content,
	})

	ctx := context.Background()
	stream, err := b.LLM.ChatStream(ctx, b.history)
	if err != nil {
		logger.Error("LLM stream error", "err", err)
		b.EventBus.Publish(Event{
			Type: EventError,
			Data: err,
		})
		return
	}

	var fullResponse string
	var firstChunkTime time.Time
	var isFirst = true
	var charCount int

	for chunk := range stream {
		if chunk.Err != nil {
			logger.Error("stream chunk error", "err", chunk.Err)
			break
		}
		if chunk.Done {
			break
		}
		
		if isFirst && chunk.Content != "" {
			firstChunkTime = time.Now()
			isFirst = false
		}
		
		charCount += len([]rune(chunk.Content))
		fullResponse += chunk.Content
		
		b.EventBus.Publish(Event{
			Type: EventStreamChunk,
			Data: chunk.Content,
		})
	}

	if !isFirst {
		duration := time.Since(firstChunkTime).Seconds()
		if duration > 0 {
			// 粗略估算：平均 1 个字符约合 1.2 个 Token（中英文混合近似）
			tokens := float64(charCount) * 1.2
			b.Stats.LastTPS = tokens / duration
		}
	}

	b.history = append(b.history, llm.Message{
		Role:    llm.RoleAssistant,
		Content: fullResponse,
	})

	b.EventBus.Publish(Event{
		Type: EventStreamDone,
		Data: fullResponse,
	})

	b.EventBus.Publish(Event{
		Type: EventMessageSend,
		Message: &Message{
			ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
			SessionID:  msg.SessionID,
			SenderID:   "bot",
			SenderName: b.Config.Bot.Nickname,
			Content:    fullResponse,
			Type:       MessageTypeText,
			Timestamp:  time.Now(),
		},
	})
}

func (b *Bot) SendMessage(msg *Message) {
	b.EventBus.Publish(Event{
		Type:    EventMessageReceive,
		Message: msg,
	})
}

func (b *Bot) GetHistory() []llm.Message {
	return b.history
}

func (b *Bot) ClearHistory() {
	b.history = []llm.Message{
		{Role: llm.RoleSystem, Content: "你是一个友好的AI助手。"},
	}
}
