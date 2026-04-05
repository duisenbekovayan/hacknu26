package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// handlePublicConfig — публичные настройки для фронта (ключи только для клиентских API с ограничением по referrer).
func (h *Handlers) handlePublicConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"google_maps_api_key": strings.TrimSpace(os.Getenv("GOOGLE_MAPS_API_KEY")),
	})
}
