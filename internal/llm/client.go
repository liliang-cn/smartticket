package llm

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// maxEmbeddingBatch is the per-request input cap (Aliyun text-embedding-v4 = 10).
const maxEmbeddingBatch = 10

// ChatMessage is a single chat turn.
type ChatMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// Client talks to any OpenAI-compatible endpoint.
type Client struct {
	api openai.Client
}

// NewClient builds a client for the given base URL and API key.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		api: openai.NewClient(
			option.WithBaseURL(baseURL),
			option.WithAPIKey(apiKey),
		),
	}
}

// Chat sends messages and returns the assistant's text.
func (c *Client) Chat(ctx context.Context, model string, msgs []ChatMessage) (string, error) {
	oa := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "system":
			oa = append(oa, openai.SystemMessage(m.Content))
		case "assistant":
			oa = append(oa, openai.AssistantMessage(m.Content))
		default:
			oa = append(oa, openai.UserMessage(m.Content))
		}
	}
	resp, err := c.api.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    model,
		Messages: oa,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Message.Content, nil
}

// Embed returns one vector per input text, batching to maxEmbeddingBatch per
// request. dimensions is sent when > 0 (v3/v4 support it).
func (c *Client) Embed(ctx context.Context, model string, dimensions int, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += maxEmbeddingBatch {
		end := start + maxEmbeddingBatch
		if end > len(texts) {
			end = len(texts)
		}
		params := openai.EmbeddingNewParams{
			Model: model,
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: texts[start:end],
			},
		}
		if dimensions > 0 {
			params.Dimensions = openai.Int(int64(dimensions))
		}
		resp, err := c.api.Embeddings.New(ctx, params)
		if err != nil {
			return nil, err
		}
		for _, d := range resp.Data {
			v := make([]float32, len(d.Embedding))
			for i, f := range d.Embedding {
				v[i] = float32(f)
			}
			out = append(out, v)
		}
	}
	return out, nil
}
