package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Text  string
	IsBot bool
}

type GeminiClient struct {
	apiKey  string
	httpCli *http.Client
}

func NewGeminiClient(apiKey string) *GeminiClient {
	if apiKey == "" {
		return nil
	}
	return &GeminiClient{
		apiKey:  apiKey,
		httpCli: &http.Client{Timeout: 30 * time.Second},
	}
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

func (c *GeminiClient) GenerateResponse(history []ChatMessage) (string, error) {
	if c == nil {
		return "", fmt.Errorf("gemini client not initialized")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", c.apiKey)

	var contents []geminiContent
	for _, msg := range history {
		role := "user"
		if msg.IsBot {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Text}},
		})
	}

	req := geminiRequest{
		Contents: contents,
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{
				Text: strings.Join([]string{
					"Ты — полезный ассистент в мессенджере",
					"Отвечай на русском языке",
					"Не используй Markdown-разметку",
					"Если тебя просят представиться — скажи, что ты Ассистент, господина создателя этого великого мессенджера.",
					"Отвечай дружелюбно. Не используй точки в конце",
				}, "\n"),
			}},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpCli.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result geminiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("gemini API error: %s", result.Error.Message)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned empty response")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
