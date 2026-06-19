package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/studiomodelconfig"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func studioModelRank(name string) float64 {
	lower := strings.ToLower(strings.TrimSpace(name))
	score := 0.0
	if m := regexp.MustCompile(`gpt[-_]?(\d+(?:\.\d+)?)`).FindStringSubmatch(lower); len(m) > 1 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			score += v * 100
		}
	}
	for _, m := range regexp.MustCompile(`(?:^|[-_])(\d+)(?:\.(\d+))?(?:[-_]|$)`).FindAllStringSubmatch(lower, -1) {
		if len(m) > 1 && m[1] != "" {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				score += v * 10
			}
		}
		if len(m) > 2 && m[2] != "" {
			if v, err := strconv.ParseFloat(m[2], 64); err == nil {
				score += v
			}
		}
	}
	if strings.Contains(lower, "latest") {
		score += 1000
	}
	if strings.Contains(lower, "preview") {
		score += 20
	}
	if strings.Contains(lower, "mini") {
		score -= 25
	}
	return score
}

func studioIsImageModel(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "image") || strings.Contains(lower, "dall-e") || strings.Contains(lower, "imagen") || strings.Contains(lower, "flux")
}

func studioIsTextModel(name string) bool {
	if studioIsImageModel(name) {
		return false
	}
	lower := strings.ToLower(name)
	for _, token := range []string{"embedding", "audio", "tts", "whisper", "moderation", "rerank"} {
		if strings.Contains(lower, token) {
			return false
		}
	}
	return strings.TrimSpace(name) != ""
}

func chooseLatestStudioModel(models []string, accept func(string) bool) string {
	best := ""
	bestScore := 0.0
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" || !accept(model) {
			continue
		}
		score := studioModelRank(model)
		if best == "" || score > bestScore || (score == bestScore && model > best) {
			best = model
			bestScore = score
		}
	}
	return best
}

func hasStudioModel(models []string, accept func(string) bool) bool {
	for _, model := range models {
		if accept(strings.TrimSpace(model)) {
			return true
		}
	}
	return false
}

func appendOpenAIStudioImageFallbackModels(models []string) []string {
	if hasStudioModel(models, studioIsImageModel) {
		return models
	}
	seen := make(map[string]struct{}, len(models)+len(openai.DefaultModels))
	out := make([]string, 0, len(models)+len(openai.DefaultModels))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		seen[model] = struct{}{}
		out = append(out, model)
	}
	for _, model := range openai.DefaultModels {
		id := strings.TrimSpace(model.ID)
		if id == "" || !studioIsImageModel(id) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func addStudioModelID(modelSet map[string]struct{}, model string) {
	model = strings.TrimSpace(model)
	if model != "" {
		modelSet[model] = struct{}{}
	}
}

func sortedStudioModelIDs(modelSet map[string]struct{}) []string {
	out := make([]string, 0, len(modelSet))
	for model := range modelSet {
		out = append(out, model)
	}
	sort.Strings(out)
	return out
}

func studioDefaultModelIDs(platform string) []string {
	switch platform {
	case service.PlatformOpenAI:
		return openai.DefaultModelIDs()
	case service.PlatformGemini:
		out := make([]string, 0, len(geminicli.DefaultModels))
		for _, model := range geminicli.DefaultModels {
			out = append(out, model.ID)
		}
		return out
	case service.PlatformAnthropic:
		return claude.DefaultModelIDs()
	case service.PlatformAntigravity:
		models := antigravity.DefaultModels()
		out := make([]string, 0, len(models))
		for _, model := range models {
			out = append(out, model.ID)
		}
		return out
	default:
		return nil
	}
}

func studioStaticModelCatalog(g *dbent.Group, accounts []service.Account) []string {
	modelSet := map[string]struct{}{}
	if g != nil && g.ModelsListConfig.Enabled && len(g.ModelsListConfig.Models) > 0 {
		for _, model := range g.ModelsListConfig.Models {
			addStudioModelID(modelSet, model)
		}
		return sortedStudioModelIDs(modelSet)
	}

	for i := range accounts {
		for model := range accounts[i].GetModelMapping() {
			addStudioModelID(modelSet, model)
		}
	}
	if len(modelSet) > 0 {
		return sortedStudioModelIDs(modelSet)
	}

	if g != nil {
		for _, model := range studioDefaultModelIDs(g.Platform) {
			addStudioModelID(modelSet, model)
		}
	}
	return sortedStudioModelIDs(modelSet)
}

func (h *StudioHandler) collectStudioModelsFromAccounts(ctx context.Context, g *dbent.Group, accounts []service.Account) ([]string, error) {
	modelSet := make(map[string]struct{})
	var lastErr error
	for i := range accounts {
		models, err := h.accountTestService.FetchUpstreamSupportedModels(ctx, &accounts[i])
		if err != nil {
			lastErr = err
			var syncErr *service.UpstreamModelSyncError
			if errors.As(err, &syncErr) {
				slog.Warn("studio_key_models_fetch_failed", "account_id", accounts[i].ID, "kind", syncErr.Kind)
			} else {
				slog.Warn("studio_key_models_fetch_failed", "account_id", accounts[i].ID)
			}
			continue
		}
		for _, model := range models {
			addStudioModelID(modelSet, model)
		}
	}
	if len(modelSet) > 0 {
		return sortedStudioModelIDs(modelSet), nil
	}

	fallback := studioStaticModelCatalog(g, accounts)
	if len(fallback) > 0 {
		if lastErr != nil && g != nil {
			slog.Warn("studio_key_models_static_catalog_fallback", "platform", g.Platform, "error", lastErr)
		}
		return fallback, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no upstream models are available for this api key")
}

func (h *StudioHandler) discoverStudioKeyModels(ctx context.Context, userID, keyID int64) ([]string, error) {
	if h.accountRepo == nil || h.accountTestService == nil {
		return nil, fmt.Errorf("studio model discovery is not configured")
	}

	ak, err := h.client.APIKey.Get(ctx, keyID)
	if err != nil || ak.UserID != userID {
		return nil, fmt.Errorf("api key not found")
	}
	if ak.GroupID == nil || *ak.GroupID <= 0 {
		return nil, fmt.Errorf("api key is not bound to a model group")
	}

	g, err := h.client.Group.Get(ctx, *ak.GroupID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, fmt.Errorf("api key model group is not available")
		}
		return nil, err
	}

	accounts, err := h.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, *ak.GroupID, g.Platform)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no schedulable upstream accounts are available for this api key")
	}

	return h.collectStudioModelsFromAccounts(ctx, g, accounts)
}

func (h *StudioHandler) autoConfigureStudioModelConfig(ctx context.Context, userID int64, current studioModelConfigDTO) (studioModelConfigDTO, error) {
	if current.Image != nil && current.Text != nil {
		return current, nil
	}

	ak, err := h.client.APIKey.Query().
		Where(apikey.UserIDEQ(userID), apikey.DeletedAtIsNil(), apikey.StatusEQ(service.StatusActive)).
		Order(dbent.Asc(apikey.FieldID)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return current, nil
		}
		return current, err
	}

	models, err := h.discoverStudioKeyModels(ctx, userID, ak.ID)
	if err != nil {
		return current, err
	}
	if ak.GroupID != nil && *ak.GroupID > 0 {
		if g, groupErr := h.client.Group.Get(ctx, *ak.GroupID); groupErr == nil && g.Platform == service.PlatformOpenAI {
			models = appendOpenAIStudioImageFallbackModels(models)
		}
	}

	next := current
	if next.Image == nil {
		if model := chooseLatestStudioModel(models, studioIsImageModel); model != "" {
			next.Image = &studioModelSlot{APIKeyID: ak.ID, Model: model}
		}
	}
	if next.Text == nil {
		if model := chooseLatestStudioModel(models, studioIsTextModel); model != "" {
			next.Text = &studioModelSlot{APIKeyID: ak.ID, Model: model}
		}
	}
	if next.Image == nil && next.Text == nil {
		return current, nil
	}

	raw, err := json.Marshal(next)
	if err != nil {
		return current, err
	}
	if err := h.client.StudioModelConfig.Create().
		SetUserID(userID).
		SetConfig(string(raw)).
		OnConflictColumns(studiomodelconfig.FieldUserID).
		UpdateNewValues().
		Exec(ctx); err != nil {
		return current, err
	}
	return next, nil
}
