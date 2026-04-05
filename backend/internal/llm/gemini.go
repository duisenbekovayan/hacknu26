package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type geminiGenerateRequest struct {
	SystemInstruction *geminiContent `json:"systemInstruction,omitempty"`
	Contents          []geminiTurn   `json:"contents"`
	GenerationConfig  geminiGenCfg   `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiTurn struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenCfg struct {
	Temperature      float64 `json:"temperature"`
	ResponseMIMEType string  `json:"responseMimeType"`
}

type geminiGenerateResponse struct {
	PromptFeedback *struct {
		BlockReason       string `json:"blockReason"`
		BlockReasonMessage string `json:"blockReasonMessage"`
	} `json:"promptFeedback"`
	Candidates []struct {
		FinishReason string `json:"finishReason"`
		Content      struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func (s *Service) analyzeGemini(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	body := geminiGenerateRequest{
		SystemInstruction: &geminiContent{Parts: []geminiPart{{Text: systemPrompt}}},
		Contents: []geminiTurn{
			{Role: "user", Parts: []geminiPart{{Text: userPrompt}}},
		},
		GenerationConfig: geminiGenCfg{
			Temperature:      0.3,
			ResponseMIMEType: "application/json",
		},
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	u, err := url.Parse("https://generativelanguage.googleapis.com/v1beta/models/" + url.PathEscape(s.model) + ":generateContent")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("key", s.geminiAPIKey)
	u.RawQuery = q.Encode()

	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, u.String(), bytes.NewReader(rawBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("gemini api: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 {
		if parsed.PromptFeedback != nil && parsed.PromptFeedback.BlockReason != "" {
			msg := parsed.PromptFeedback.BlockReason
			if parsed.PromptFeedback.BlockReasonMessage != "" {
				msg += ": " + parsed.PromptFeedback.BlockReasonMessage
			}
			return "", fmt.Errorf("gemini blocked (%s)", msg)
		}
		return "", fmt.Errorf("gemini: пустой ответ (проверьте GEMINI_MODEL и что Generative Language API включён для ключа)")
	}
	if len(parsed.Candidates[0].Content.Parts) == 0 {
		fr := parsed.Candidates[0].FinishReason
		if fr != "" {
			return "", fmt.Errorf("gemini: нет текста в ответе (finishReason=%s)", fr)
		}
		return "", fmt.Errorf("gemini: нет текста в ответе")
	}
	var b strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		b.WriteString(p.Text)
	}
	return strings.TrimSpace(b.String()), nil
}
