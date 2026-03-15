package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/welife-os/welife-os/engine/internal/llm"
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

// isMaskedKey returns true if the value looks like a masked API key
// (contains consecutive asterisks), meaning the user didn't type a real key.
func isMaskedKey(s string) bool {
	return strings.Contains(s, "***")
}

var validProviders = map[string]struct{}{
	"ollama":            {},
	"openai-compatible": {},
}

func (s *Server) handleUpdateLLMConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Limit request body to 1 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

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
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "base_url must be a valid URL"})
			return
		}
	}

	currentSettings, err := s.store.GetSettings(ctx, llmSettingKeys)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read existing settings"})
		return
	}

	nextSettings := cloneSettings(currentSettings)
	persistedPatch := make(map[string]string, 5)

	assignPatchSetting(nextSettings, persistedPatch, "llm_provider", req.Provider)
	assignPatchSetting(nextSettings, persistedPatch, "llm_base_url", req.BaseURL)
	assignPatchSetting(nextSettings, persistedPatch, "llm_model", req.Model)
	assignPatchSetting(nextSettings, persistedPatch, "llm_embedding_model", req.EmbeddingModel)

	if req.APIKey != nil && *req.APIKey != "" && !isMaskedKey(*req.APIKey) {
		nextSettings["llm_api_key"] = *req.APIKey
		persistedPatch["llm_api_key"] = *req.APIKey
	}

	candidateCfg := applySettingsToLLMConfig(baseLLMConfig(s.config), nextSettings)
	newClient, err := llm.NewClient(candidateCfg)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid llm config: " + err.Error(),
		})
		return
	}

	if err := s.store.SaveSettings(ctx, persistedPatch); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save setting"})
		return
	}

	s.llmClient.Swap(newClient)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func cloneSettings(settings map[string]string) map[string]string {
	cloned := make(map[string]string, len(settings))
	for key, value := range settings {
		cloned[key] = value
	}
	return cloned
}

func assignPatchSetting(settings map[string]string, persistedPatch map[string]string, key string, value *string) {
	if value == nil {
		return
	}
	settings[key] = *value
	persistedPatch[key] = *value
}
