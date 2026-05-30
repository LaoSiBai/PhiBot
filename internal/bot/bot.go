package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/phibot/phibot/internal/config"
	"github.com/phibot/phibot/internal/llm"
	"github.com/phibot/phibot/internal/logger"
)

type Bot struct {
	Config   *config.Config
	EventBus *EventBus
	LLM      llm.Provider
	history  []llm.Message
}

func New(cfg *config.Config) (*Bot, error) {
	provider, err := llm.NewOpenAIProvider(cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	b := &Bot{
		Config:   cfg,
		EventBus: NewEventBus(),
		LLM:      provider,
		history: []llm.Message{
			{Role: llm.RoleSystem, Content: "你是一个友好的AI助手。"},
		},
	}

	b.EventBus.Subscribe(EventMessageReceive, b.handleMessage)

	return b, nil
}

func (b *Bot) handleMessage(event Event) {
	if event.Message == nil {
		return
	}

	msg := event.Message
	logger.Info("received message", "from", msg.SenderName, "content", msg.Content)

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
	for chunk := range stream {
		if chunk.Err != nil {
			logger.Error("stream chunk error", "err", chunk.Err)
			break
		}
		if chunk.Done {
			break
		}
		fullResponse += chunk.Content
		b.EventBus.Publish(Event{
			Type: EventStreamChunk,
			Data: chunk.Content,
		})
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
