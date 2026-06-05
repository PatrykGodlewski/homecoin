package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type BudgetAllocation struct {
	Category              string `json:"category"`
	RecommendedLimitCents int64  `json:"recommended_limit_cents"`
	Rationale             string `json:"rationale"`
}

type BudgetSuggestion struct {
	Allocations        []BudgetAllocation `json:"allocations"`
	SavingsTargetCents int64              `json:"savings_target_cents"`
	Summary            string             `json:"summary"`
}

type Client struct {
	client *openai.Client
	model  openai.ChatModel
}

func NewClient(apiKey, model string) *Client {
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &Client{
		client: openai.NewClient(opts...),
		model:  openai.ChatModel(model),
	}
}

func (c *Client) SuggestBudget(ctx context.Context, metadata json.RawMessage) (*BudgetSuggestion, int, error) {
	systemPrompt := `You are a household budgeting assistant. Given anonymized household financial metadata, respond ONLY with valid JSON matching this schema:
{"allocations":[{"category":"string","recommended_limit_cents":number,"rationale":"string"}],"savings_target_cents":number,"summary":"string"}
All amounts are in cents. Be practical and conservative.`

	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.F(c.model),
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(string(metadata)),
		}),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("openai completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, 0, fmt.Errorf("openai returned no choices")
	}

	content := resp.Choices[0].Message.Content
	var suggestion BudgetSuggestion
	if err := json.Unmarshal([]byte(content), &suggestion); err != nil {
		return nil, 0, fmt.Errorf("parse suggestion json: %w", err)
	}

	tokens := int(resp.Usage.TotalTokens)
	return &suggestion, tokens, nil
}
