package llm

import (
	"context"
	"fmt"

	"github.com/phibot/phibot/internal/config"
	"github.com/phibot/phibot/internal/logger"
	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
	model  string
	cfg    config.LLMConfig
}

func NewOpenAIProvider(cfg config.LLMConfig) (*OpenAIProvider, error) {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIProvider{
		client: client,
		model:  cfg.Model,
		cfg:    cfg,
	}, nil
}

func (p *OpenAIProvider) ModelName() string {
	return p.model
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		Temperature: float32(p.cfg.Temperature),
		MaxTokens:   p.cfg.MaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("openai chat error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return resp.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		Temperature: float32(p.cfg.Temperature),
		MaxTokens:   p.cfg.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("openai stream error: %w", err)
	}

	ch := make(chan StreamChunk)

	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					ch <- StreamChunk{Done: true}
					return
				}
				logger.Error("stream recv error", "err", err)
				ch <- StreamChunk{Err: err, Done: true}
				return
			}

			if len(response.Choices) > 0 {
				content := response.Choices[0].Delta.Content
				if content != "" {
					ch <- StreamChunk{Content: content}
				}

				if response.Choices[0].FinishReason != "" {
					ch <- StreamChunk{Done: true}
					return
				}
			}
		}
	}()

	return ch, nil
}
