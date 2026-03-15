package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type llmConfigResponse struct {
	Provider       string `json:"provider"`
	BaseURL        string `json:"base_url"`
	Model          string `json:"model"`
	APIKey         string `json:"api_key"`
	EmbeddingModel string `json:"embedding_model"`
}

type llmConfigPatchRequest struct {
	Provider       *string `json:"provider"`
	BaseURL        *string `json:"base_url"`
	Model          *string `json:"model"`
	APIKey         *string `json:"api_key"`
	EmbeddingModel *string `json:"embedding_model"`
}

// maskAPIKey returns a masked version of the key for safe display.
// Example: "sk-abc123xyz" → "sk-****xyz"
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return key[:3] + "****" + key[len(key)-4:]
}

func (s *Server) handleGetLLMConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	settings, err := s.store.GetSettings(ctx, llmSettingKeys)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read settings"})
		return
	}

	// Start from current runtime values, override with DB values.
	resp := llmConfigResponse{
		Provider:       s.config.LLMProvider,
		BaseURL:        s.config.LLMBaseURL,
		Model:          s.config.LLMModel,
		APIKey:         maskAPIKey(s.config.LLMAPIKey),
		EmbeddingModel: s.config.EmbeddingModel,
	}

	if v, ok := settings["llm_provider"]; ok {
		resp.Provider = v
	}
	if v, ok := settings["llm_base_url"]; ok {
		resp.BaseURL = v
	}
	if v, ok := settings["llm_model"]; ok {
		resp.Model = v
	}
	if v, ok := settings["llm_api_key"]; ok {
		resp.APIKey = maskAPIKey(v)
	}
	if v, ok := settings["llm_embedding_model"]; ok {
		resp.EmbeddingModel = v
	}

	writeJSON(w, http.StatusOK, resp)
}

var validProviders = map[string]struct{}{
	"ollama":            {},
	"openai-compatible": {},
}

func (s *Server) handleUpdateLLMConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req llmConfigPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	// Validate provider.
	if req.Provider != nil {
		if _, ok := validProviders[*req.Provider]; !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "provider must be \"ollama\" or \"openai-compatible\"",
			})
			return
		}
	}

	// Validate base_url.
	if req.BaseURL != nil {
		parsed, err := url.Parse(*req.BaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "base_url must be a valid URL"})
			return
		}
	}

	// Save each provided field.
	saves := map[string]*string{
		"llm_provider":        req.Provider,
		"llm_base_url":        req.BaseURL,
		"llm_model":           req.Model,
		"llm_embedding_model": req.EmbeddingModel,
	}

	for key, val := range saves {
		if val != nil {
			if err := s.store.SaveSetting(ctx, key, *val); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save setting"})
				return
			}
		}
	}

	// Handle api_key separately: skip if empty or looks like a mask.
	if req.APIKey != nil && *req.APIKey != "" && !strings.Contains(*req.APIKey, "****") {
		if err := s.store.SaveSetting(ctx, "llm_api_key", *req.APIKey); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save api key"})
			return
		}
	}

	// Hot-swap the LLM client.
	if err := s.swapLLMClient(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "config saved but failed to apply: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
