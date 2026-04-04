package api

import (
	"encoding/json"
	"net/http"

	"hacknu/backend/internal/llm"
)

func (h *Handlers) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.ai == nil || !h.ai.Enabled() {
		http.Error(w, "llm disabled: set GEMINI_API_KEY or OPENAI_API_KEY", http.StatusServiceUnavailable)
		return
	}

	var in llm.AnalyzeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	out, err := h.ai.Analyze(r.Context(), in)
	if err != nil {
		h.log.Error("ai analyze", "err", err)
		http.Error(w, "ai analyze failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handlers) handleAIStatus(w http.ResponseWriter, _ *http.Request) {
	enabled := h.ai != nil && h.ai.Enabled()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"enabled": enabled})
}
