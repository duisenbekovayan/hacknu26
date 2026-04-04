package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// Service — LLM-интерпретация: OpenAI Responses / Chat Completions (в т.ч. OpenRouter sk-or-v1…) или Gemini (AIza…).
type Service struct {
	log           *slog.Logger
	provider      string // "openai" | "gemini"
	client        openai.Client
	geminiAPIKey  string
	model         string
	openaiUseChat bool // Chat Completions вместо Responses (OpenRouter, прокси)
	enabled       bool
}

func NewService(log *slog.Logger) *Service {
	geminiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	openaiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	// Ключи Google часто ошибочно кладут в OPENAI_API_KEY.
	if geminiKey == "" && strings.HasPrefix(openaiKey, "AIza") {
		geminiKey = openaiKey
		openaiKey = ""
	}

	s := &Service{log: log}

	if geminiKey != "" {
		s.provider = "gemini"
		s.geminiAPIKey = geminiKey
		s.model = strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
		if s.model == "" {
			s.model = "gemini-2.0-flash"
		}
		s.enabled = true
		log.Info("llm provider", "name", "gemini", "model", s.model)
		return s
	}

	if openaiKey != "" {
		s.provider = "openai"
		base := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
		s.openaiUseChat = os.Getenv("OPENAI_USE_CHAT") == "1" ||
			strings.HasPrefix(openaiKey, "sk-or-v1") ||
			strings.Contains(strings.ToLower(base), "openrouter")
		if s.openaiUseChat && base == "" && strings.HasPrefix(openaiKey, "sk-or-v1") {
			base = "https://openrouter.ai/api/v1"
		}
		s.model = strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
		if s.model == "" {
			if s.openaiUseChat && strings.HasPrefix(openaiKey, "sk-or-v1") {
				s.model = "openai/gpt-4o-mini"
			} else {
				s.model = "gpt-4o-mini"
			}
		}
		opts := []option.RequestOption{option.WithAPIKey(openaiKey)}
		if base != "" {
			opts = append(opts, option.WithBaseURL(base))
		}
		s.client = openai.NewClient(opts...)
		s.enabled = true
		mode := "responses"
		if s.openaiUseChat {
			mode = "chat"
		}
		log.Info("llm provider", "name", "openai", "model", s.model, "mode", mode, "base", base)
		return s
	}

	log.Warn("llm disabled: set GEMINI_API_KEY or OPENAI_API_KEY")
	return s
}

func (s *Service) Enabled() bool {
	return s.enabled
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func buildPrompts(in AnalyzeInput) (systemPrompt string, userPrompt string, err error) {
	payload, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return "", "", err
	}

	systemPrompt = `Ты — помощник машиниста локомотива.
Анализируй телеметрию кратко, строго и по делу.

Твоя задача:
1. Объяснить текущее состояние
2. Определить уровень опасности
3. Назвать вероятные причины
4. Дать конкретные рекомендации машинисту

Правила:
- Не выдумывай датчики, которых нет в snapshot
- Не пиши длинных объяснений
- Учитывай alerts и health_index
- Если ситуация безопасна — так и скажи
- Если есть риск — дай 2–4 конкретных действия
- Поле next_risk: кратко (1 предложение), что может случиться при сохранении тренда; если не уместно — пустая строка
- Отвечай ТОЛЬКО валидным JSON без markdown и без текста вокруг

Формат ответа:
{
  "summary": "краткое объяснение",
  "severity": "normal|warning|critical",
  "probable_causes": ["..."],
  "recommendations": ["..."],
  "affected_metrics": ["..."],
  "next_risk": ""
}`

	userPrompt = fmt.Sprintf("Проанализируй этот snapshot телеметрии:\n%s", string(payload))
	if strings.EqualFold(strings.TrimSpace(in.Mode), "actions") {
		userPrompt += "\n\nРежим «что делать»: сфокусируйся на конкретных немедленных действиях машиниста; summary — одно короткое предложение о срочности; recommendations — главное."
	}
	return systemPrompt, userPrompt, nil
}

func (s *Service) analyzeOpenAI(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if s.openaiUseChat {
		return s.analyzeOpenAIChat(ctx, systemPrompt, userPrompt)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	resp, err := s.client.Responses.New(reqCtx, responses.ResponseNewParams{
		Model: shared.ResponsesModel(s.model),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(systemPrompt, responses.EasyInputMessageRoleSystem),
				responses.ResponseInputItemParamOfMessage(userPrompt, responses.EasyInputMessageRoleUser),
			},
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.OutputText()), nil
}

func (s *Service) analyzeOpenAIChat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	resp, err := s.client.Chat.Completions.New(reqCtx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(s.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			{OfSystem: &openai.ChatCompletionSystemMessageParam{
				Content: openai.ChatCompletionSystemMessageParamContentUnion{
					OfString: openai.String(systemPrompt),
				},
			}},
			{OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfString: openai.String(userPrompt),
				},
			}},
		},
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai chat: empty choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func (s *Service) Analyze(ctx context.Context, in AnalyzeInput) (*AnalyzeOutput, error) {
	if !s.enabled {
		return nil, fmt.Errorf("llm disabled")
	}

	systemPrompt, userPrompt, err := buildPrompts(in)
	if err != nil {
		return nil, err
	}

	var raw string
	switch s.provider {
	case "gemini":
		raw, err = s.analyzeGemini(ctx, systemPrompt, userPrompt)
	default:
		raw, err = s.analyzeOpenAI(ctx, systemPrompt, userPrompt)
	}
	if err != nil {
		return nil, err
	}

	raw = extractJSONObject(raw)

	var out AnalyzeOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		s.log.Error("llm json parse failed", "raw", raw, "err", err)
		return nil, fmt.Errorf("invalid llm json: %w", err)
	}

	return &out, nil
}
