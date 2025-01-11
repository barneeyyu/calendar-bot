package utils

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v2"
)

//go:embed prompt/time_parser.yaml
var timeParserYAML []byte

type ParserPrompt struct {
	SystemPrompt string `yaml:"system_prompt"`
}

type TransferScheduleResponse struct {
	DateTime string `json:"dateTime"` // format: "2024-12-29 14:00"
	Task     string `json:"task"`     // example: "have a haircut"
	Valid    bool   `json:"valid"`    // true/false
}

type OpenaiHandler interface {
	TransferValidSchedule(inputMsg string) (*TransferScheduleResponse, error)
}

type OpenaiClient struct {
	client *openai.Client
}

func NewOpenAIClient(apiKey string, baseUrl string) (OpenaiHandler, error) {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseUrl
	client := openai.NewClientWithConfig(config)
	return &OpenaiClient{
		client: client,
	}, nil
}

func (c *OpenaiClient) TransferValidSchedule(inputMsg string) (*TransferScheduleResponse, error) {
	var prompt ParserPrompt
	err := yaml.Unmarshal(timeParserYAML, &prompt)
	if err != nil {
		return nil, fmt.Errorf("error parsing prompt yaml: %w", err)
	}
	systemPrompt := prompt.SystemPrompt + "\n當前時間：" + time.Now().Format("2006-01-02 15:04")

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: inputMsg,
				},
			},
			Temperature: 0.0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	var transferScheduleResponse TransferScheduleResponse
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &transferScheduleResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling openai API response: %w", err)
	}

	return &transferScheduleResponse, nil
}
