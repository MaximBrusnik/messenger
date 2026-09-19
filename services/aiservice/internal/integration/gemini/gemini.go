package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const geminiSystemInstruction = "Ты — умный ассистент в мессенджере. Отвечай кратко и по делу, на русском языке. Не используй markdown."

type ChatMessage struct {
	Text  string `json:"text"`
	IsBot bool   `json:"is_bot"`
}

type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type Client struct {
	apiKey  string
	httpCli *http.Client
}

func NewClient(apiKey string) *Client {
	if apiKey == "" {
		return nil
	}
	return &Client{
		apiKey:  apiKey,
		httpCli: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) GenerateResponse(history []ChatMessage) (string, error) {
	if c == nil {
		return "", fmt.Errorf("gemini client is not configured")
	}

	contents := []geminiContent{}
	for _, m := range history {
		role := "user"
		if m.IsBot {
			role = "model"
		}
		contents = append(contents, geminiContent{Role: role, Parts: []geminiPart{{Text: m.Text}}})
	}

	payload := geminiRequest{Contents: contents}
	payload.SystemInstruction = &geminiContent{Role: "system", Parts: []geminiPart{{Text: geminiSystemInstruction}}}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent?key=" + c.apiKey
	resp, err := c.httpCli.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	var gres geminiResponse
	if err := json.Unmarshal(data, &gres); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if gres.Error != nil {
		return "", fmt.Errorf("gemini api error: %s", gres.Error.Message)
	}
	if len(gres.Candidates) == 0 || len(gres.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned empty response")
	}
	text := gres.Candidates[0].Content.Parts[0].Text
	return strings.TrimSpace(text), nil
}
