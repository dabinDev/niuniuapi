package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/creationtask"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/ent/studiomodelconfig"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type studioModelAccountRepoStub struct {
	service.AccountRepository
	byGroupPlatform map[int64]map[string][]service.Account
	byID            map[int64]*service.Account
}

func (s *studioModelAccountRepoStub) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]service.Account, error) {
	byPlatform := s.byGroupPlatform[groupID]
	if byPlatform == nil {
		return nil, nil
	}
	accounts := byPlatform[platform]
	out := make([]service.Account, len(accounts))
	copy(out, accounts)
	return out, nil
}

func (s *studioModelAccountRepoStub) GetByID(ctx context.Context, id int64) (*service.Account, error) {
	if s.byID == nil {
		return nil, errors.New("account not found")
	}
	account := s.byID[id]
	if account == nil {
		return nil, errors.New("account not found")
	}
	copy := *account
	return &copy, nil
}

type studioModelsHTTPUpstream struct {
	lastReq       *http.Request
	modelsPayload string
	modelsStatus  int
}

func (u *studioModelsHTTPUpstream) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	u.lastReq = req
	if strings.HasSuffix(req.URL.Path, "/images/generations") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"aW1hZ2U="}]}`)),
		}, nil
	}
	return &http.Response{
		StatusCode: u.modelsStatusCode(),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(u.modelsResponse())),
	}, nil
}

func (u *studioModelsHTTPUpstream) modelsStatusCode() int {
	if u.modelsStatus != 0 {
		return u.modelsStatus
	}
	return http.StatusOK
}

func (u *studioModelsHTTPUpstream) modelsResponse() string {
	if strings.TrimSpace(u.modelsPayload) != "" {
		return u.modelsPayload
	}
	return `{"data":[{"id":"gpt-5.4"},{"id":"gpt-image-2"}]}`
}

func (u *studioModelsHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func newStudioHandlerTestClient(t *testing.T, name string) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

type studioCoverGatewayRequest struct {
	Path        string
	ContentType string
	Body        []byte
}

type studioCoverGatewayRecorder struct {
	mu       sync.Mutex
	requests []studioCoverGatewayRequest
}

func (r *studioCoverGatewayRecorder) add(req studioCoverGatewayRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
}

func (r *studioCoverGatewayRecorder) snapshot() []studioCoverGatewayRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]studioCoverGatewayRequest, len(r.requests))
	copy(out, r.requests)
	return out
}

func startStudioCoverGatewayRecorder(t *testing.T) *studioCoverGatewayRecorder {
	t.Helper()
	return startStudioCoverGatewayRecorderWithRelease(t, nil)
}

func startStudioChatGatewayRecorder(t *testing.T, content string) *studioCoverGatewayRecorder {
	t.Helper()
	recorder := &studioCoverGatewayRecorder{}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	t.Setenv("SERVER_PORT", port)

	payload, err := json.Marshal(gin.H{
		"choices": []gin.H{{
			"message": gin.H{"content": content},
		}},
	})
	require.NoError(t, err)

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		recorder.add(studioCoverGatewayRequest{
			Path:        r.URL.Path,
			ContentType: r.Header.Get("Content-Type"),
			Body:        body,
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	})}
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	return recorder
}

func startStudioCoverGatewayRecorderWithRelease(t *testing.T, release <-chan struct{}) *studioCoverGatewayRecorder {
	t.Helper()
	recorder := &studioCoverGatewayRecorder{}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	t.Setenv("SERVER_PORT", port)

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		recorder.add(studioCoverGatewayRequest{
			Path:        r.URL.Path,
			ContentType: r.Header.Get("Content-Type"),
			Body:        body,
		})
		if release != nil {
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"aW1hZ2U="}]}`))
	})}
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	return recorder
}

func newStudioCoverHandlerWithConfig(t *testing.T, name string, model string) (*StudioHandler, int64) {
	t.Helper()
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, name)

	user, err := client.User.Create().
		SetEmail(name + "@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName(name + "-openai").
		SetPlatform(service.PlatformOpenAI).
		Save(ctx)
	require.NoError(t, err)
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetName(name + "-key").
		SetKey("sk-" + name).
		SetGroupID(group.ID).
		Save(ctx)
	require.NoError(t, err)

	cfg, err := json.Marshal(studioModelConfigDTO{
		Image: &studioModelSlot{APIKeyID: apiKey.ID, Model: model},
	})
	require.NoError(t, err)
	_, err = client.StudioModelConfig.Create().
		SetUserID(user.ID).
		SetConfig(string(cfg)).
		Save(ctx)
	require.NoError(t, err)

	return &StudioHandler{client: client}, user.ID
}

func newStudioTextHandlerWithConfig(t *testing.T, name string, model string) (*StudioHandler, int64) {
	t.Helper()
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, name)

	user, err := client.User.Create().
		SetEmail(name + "@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName(name + "-openai").
		SetPlatform(service.PlatformOpenAI).
		Save(ctx)
	require.NoError(t, err)
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetName(name + "-key").
		SetKey("sk-" + name).
		SetGroupID(group.ID).
		Save(ctx)
	require.NoError(t, err)

	cfg, err := json.Marshal(studioModelConfigDTO{
		Text: &studioModelSlot{APIKeyID: apiKey.ID, Model: model},
	})
	require.NoError(t, err)
	_, err = client.StudioModelConfig.Create().
		SetUserID(user.ID).
		SetConfig(string(cfg)).
		Save(ctx)
	require.NoError(t, err)

	return &StudioHandler{client: client}, user.ID
}

func performStudioCoverRequest(t *testing.T, h *StudioHandler, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/studio/cover", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.GenerateCover(c)
	return rec
}

func performStudioCoverPromptPolishRequest(t *testing.T, h *StudioHandler, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/studio/cover/prompt-polish", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.PolishCoverPrompt(c)
	return rec
}

func performStudioCoverJobStart(t *testing.T, h *StudioHandler, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/studio/cover/jobs", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.StartCoverJob(c)
	return rec
}

func performStudioCoverJobGet(t *testing.T, h *StudioHandler, userID int64, jobID string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/studio/cover/jobs/"+jobID, nil)
	c.Params = gin.Params{{Key: "id", Value: jobID}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.GetCoverJob(c)
	return rec
}

func performStudioImportRequest(t *testing.T, h *StudioHandler, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/studio/import", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.ImportStudioContent(c)
	return rec
}

func performStudioFanqieRankRequest(t *testing.T, h *StudioHandler, userID int64, channel string) *httptest.ResponseRecorder {
	return performStudioFanqieRankRequestWithForce(t, h, userID, channel, false)
}

func performStudioFanqieRankRequestWithForce(t *testing.T, h *StudioHandler, userID int64, channel string, forceRefresh bool) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	target := "/api/v1/studio/fanqie/rank?channel=" + channel
	if forceRefresh {
		target += "&force_refresh=true"
	}
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.GetFanqieRank(c)
	return rec
}

func createFanqieRankSnapshotTableForTest(t *testing.T, client *dbent.Client) {
	t.Helper()
	_, err := client.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS fanqie_rank_snapshots (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	channel TEXT NOT NULL UNIQUE,
	books TEXT NOT NULL DEFAULT '[]',
	source TEXT NOT NULL DEFAULT '',
	fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
	require.NoError(t, err)
}

func resetFanqieRankCacheForTest() {
	fanqieRankCache.Lock()
	fanqieRankCache.entries = map[fanqieRankChannel]fanqieRankCacheEntry{}
	fanqieRankCache.Unlock()
}

func performStudioFanqieCoverRequest(t *testing.T, h *StudioHandler, key string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/studio/fanqie/covers/"+key, nil)
	c.Params = gin.Params{{Key: "key", Value: key}}

	h.ServeFanqieCover(c)
	return rec
}

func resetFanqieCoverCachesForTest() {
	fanqieCoverSources.Lock()
	fanqieCoverSources.entries = map[string]fanqieCoverSourceEntry{}
	fanqieCoverSources.Unlock()

	fanqieCoverBytesCache.Lock()
	fanqieCoverBytesCache.entries = map[string]fanqieCoverCacheEntry{}
	fanqieCoverBytesCache.Unlock()
}

func registerFanqieCoverSourceForTest(rawURL string) (string, bool) {
	return registerFanqieCoverSource(rawURL)
}

func performStudioFanqieSearchRequest(t *testing.T, h *StudioHandler, userID int64, query string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/studio/fanqie/search?q="+query, nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.SearchFanqieBooks(c)
	return rec
}

func performStudioFanqieDownloadRequest(t *testing.T, h *StudioHandler, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/studio/fanqie/download", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.DownloadFanqieBook(c)
	return rec
}

func performStudioFanqieAnalyzeRequest(t *testing.T, h *StudioHandler, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/studio/fanqie/analyze", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	h.AnalyzeFanqieBook(c)
	return rec
}

func TestStudioKeyModelsFetchesLiveUpstreamModelsForSelectedUserKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_key_models_live_upstream")

	user, err := client.User.Create().
		SetEmail("studio-models@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName("openai-studio-models").
		SetPlatform(service.PlatformOpenAI).
		Save(ctx)
	require.NoError(t, err)
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetName("studio-key").
		SetKey("sk-studio-test").
		SetGroupID(group.ID).
		Save(ctx)
	require.NoError(t, err)

	upstream := &studioModelsHTTPUpstream{}
	repo := &studioModelAccountRepoStub{byGroupPlatform: map[int64]map[string][]service.Account{
		group.ID: {
			service.PlatformOpenAI: {
				{
					ID:       9,
					Name:     "mapped-only-locally",
					Platform: service.PlatformOpenAI,
					Type:     service.AccountTypeAPIKey,
					Credentials: map[string]any{
						"api_key":       "upstream-secret",
						"base_url":      "https://upstream.example.com/v1",
						"model_mapping": map[string]any{"gpt-5-mini": "gpt-5-mini", "gpt-image-1": "gpt-image-1"},
					},
					Concurrency: 1,
				},
			},
		},
	}}
	h := &StudioHandler{
		client:      client,
		accountRepo: repo,
		accountTestService: service.NewAccountTestService(
			repo,
			nil,
			nil,
			nil,
			upstream,
			&config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
			nil,
		),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/studio/keys/"+strconv.FormatInt(apiKey.ID, 10)+"/models", nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(apiKey.ID, 10)}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	h.ListKeyModels(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int      `json:"code"`
		Data []string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.ElementsMatch(t, []string{"gpt-5.4", "gpt-image-2"}, resp.Data)
	require.NotContains(t, resp.Data, "gpt-5-mini")
	require.NotContains(t, resp.Data, "gpt-image-1")
	require.Equal(t, "https://upstream.example.com/v1/models", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer upstream-secret", upstream.lastReq.Header.Get("Authorization"))
}

func TestStudioKeyModelsFallsBackToStaticCatalogWhenUpstreamModelListUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_key_models_upstream_unauthorized_fallback")

	user, err := client.User.Create().
		SetEmail("studio-models-unauthorized@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName("openai-studio-models-unauthorized").
		SetPlatform(service.PlatformOpenAI).
		Save(ctx)
	require.NoError(t, err)
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetName("studio-key").
		SetKey("sk-studio-test").
		SetGroupID(group.ID).
		Save(ctx)
	require.NoError(t, err)

	upstream := &studioModelsHTTPUpstream{
		modelsStatus:  http.StatusUnauthorized,
		modelsPayload: `{"error":"expired upstream token"}`,
	}
	repo := &studioModelAccountRepoStub{byGroupPlatform: map[int64]map[string][]service.Account{
		group.ID: {
			service.PlatformOpenAI: {
				{
					ID:       19,
					Name:     "unauthorized-model-list",
					Platform: service.PlatformOpenAI,
					Type:     service.AccountTypeAPIKey,
					Credentials: map[string]any{
						"api_key":  "upstream-secret",
						"base_url": "https://upstream.example.com/v1",
					},
					Concurrency: 1,
				},
			},
		},
	}}
	h := &StudioHandler{
		client:      client,
		accountRepo: repo,
		accountTestService: service.NewAccountTestService(
			repo,
			nil,
			nil,
			nil,
			upstream,
			&config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
			nil,
		),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/studio/keys/"+strconv.FormatInt(apiKey.ID, 10)+"/models", nil)
	c.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(apiKey.ID, 10)}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	h.ListKeyModels(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int      `json:"code"`
		Data []string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Contains(t, resp.Data, "gpt-5.5")
	require.Contains(t, resp.Data, "gpt-image-2")
	require.NotContains(t, rec.Body.String(), "HTTP 401")
	require.NotContains(t, rec.Body.String(), "expired upstream token")
}

func TestStudioModelTestUsesUpstreamAccountDirectly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_model_test_upstream")

	user, err := client.User.Create().
		SetEmail("studio-model-test@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName("openai-studio-model-test").
		SetPlatform(service.PlatformOpenAI).
		Save(ctx)
	require.NoError(t, err)
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetName("studio-key").
		SetKey("sk-user-key-without-balance").
		SetGroupID(group.ID).
		Save(ctx)
	require.NoError(t, err)

	account := service.Account{
		ID:       9,
		Name:     "upstream-image-account",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "upstream-secret",
			"base_url": "https://upstream.example.com/v1",
		},
		Concurrency: 1,
	}
	upstream := &studioModelsHTTPUpstream{}
	repo := &studioModelAccountRepoStub{
		byID: map[int64]*service.Account{account.ID: &account},
		byGroupPlatform: map[int64]map[string][]service.Account{
			group.ID: {
				service.PlatformOpenAI: {account},
			},
		},
	}
	h := &StudioHandler{
		client:      client,
		accountRepo: repo,
		accountTestService: service.NewAccountTestService(
			repo,
			nil,
			nil,
			nil,
			upstream,
			&config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
			nil,
		),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/studio/model-test", strings.NewReader(`{
		"type":"image",
		"api_key_id":`+strconv.FormatInt(apiKey.ID, 10)+`,
		"model":"gpt-image-2"
	}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	h.TestModel(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "https://upstream.example.com/v1/images/generations", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer upstream-secret", upstream.lastReq.Header.Get("Authorization"))
	var payload map[string]any
	body, err := io.ReadAll(upstream.lastReq.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "gpt-image-2", payload["model"])
	require.Equal(t, "b64_json", payload["response_format"])
}

func TestGetModelConfigAutoConfiguresFirstKeyWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_model_config_auto_first_key")

	user, err := client.User.Create().
		SetEmail("studio-auto-config@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName("openai-auto-config").
		SetPlatform(service.PlatformOpenAI).
		Save(ctx)
	require.NoError(t, err)
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetName("first-key").
		SetKey("sk-auto-config").
		SetGroupID(group.ID).
		Save(ctx)
	require.NoError(t, err)

	upstream := &studioModelsHTTPUpstream{}
	repo := &studioModelAccountRepoStub{byGroupPlatform: map[int64]map[string][]service.Account{
		group.ID: {
			service.PlatformOpenAI: {
				{
					ID:       11,
					Name:     "auto-config-account",
					Platform: service.PlatformOpenAI,
					Type:     service.AccountTypeAPIKey,
					Credentials: map[string]any{
						"api_key":  "upstream-secret",
						"base_url": "https://upstream.example.com/v1",
					},
					Concurrency: 1,
				},
			},
		},
	}}
	h := &StudioHandler{
		client:      client,
		accountRepo: repo,
		accountTestService: service.NewAccountTestService(
			repo,
			nil,
			nil,
			nil,
			upstream,
			&config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
			nil,
		),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/studio/model-config", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	h.GetModelConfig(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int                  `json:"code"`
		Data studioModelConfigDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data.Image)
	require.NotNil(t, resp.Data.Text)
	require.Equal(t, apiKey.ID, resp.Data.Image.APIKeyID)
	require.Equal(t, apiKey.ID, resp.Data.Text.APIKeyID)
	require.Equal(t, "gpt-image-2", resp.Data.Image.Model)
	require.Equal(t, "gpt-5.4", resp.Data.Text.Model)

	row, err := client.StudioModelConfig.Query().
		Where(studiomodelconfig.UserIDEQ(user.ID)).
		Only(ctx)
	require.NoError(t, err)
	require.Contains(t, row.Config, "gpt-image-2")
	require.Contains(t, row.Config, "gpt-5.4")
}

func TestGetModelConfigAutoConfiguresOpenAIImageFallbackWhenModelsEndpointOmitsImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_model_config_auto_openai_image_fallback")

	user, err := client.User.Create().
		SetEmail("studio-auto-image-fallback@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName("openai-auto-image-fallback").
		SetPlatform(service.PlatformOpenAI).
		Save(ctx)
	require.NoError(t, err)
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetName("first-key").
		SetKey("sk-auto-image-fallback").
		SetGroupID(group.ID).
		Save(ctx)
	require.NoError(t, err)

	existing, err := json.Marshal(studioModelConfigDTO{
		Text: &studioModelSlot{APIKeyID: apiKey.ID, Model: "gpt-5.5"},
	})
	require.NoError(t, err)
	_, err = client.StudioModelConfig.Create().
		SetUserID(user.ID).
		SetConfig(string(existing)).
		Save(ctx)
	require.NoError(t, err)

	upstream := &studioModelsHTTPUpstream{
		modelsPayload: `{"data":[{"id":"gpt-5.4"},{"id":"gpt-5.5"}]}`,
	}
	repo := &studioModelAccountRepoStub{byGroupPlatform: map[int64]map[string][]service.Account{
		group.ID: {
			service.PlatformOpenAI: {
				{
					ID:       12,
					Name:     "openai-text-only-model-list",
					Platform: service.PlatformOpenAI,
					Type:     service.AccountTypeAPIKey,
					Credentials: map[string]any{
						"api_key":  "upstream-secret",
						"base_url": "https://upstream.example.com/v1",
					},
					Concurrency: 1,
				},
			},
		},
	}}
	h := &StudioHandler{
		client:      client,
		accountRepo: repo,
		accountTestService: service.NewAccountTestService(
			repo,
			nil,
			nil,
			nil,
			upstream,
			&config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
			nil,
		),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/studio/model-config", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID})

	h.GetModelConfig(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int                  `json:"code"`
		Data studioModelConfigDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data.Image)
	require.NotNil(t, resp.Data.Text)
	require.Equal(t, apiKey.ID, resp.Data.Image.APIKeyID)
	require.Equal(t, "gpt-image-2", resp.Data.Image.Model)
	require.Equal(t, "gpt-5.5", resp.Data.Text.Model)
}

func TestBuildCoverPromptEmphasizesNovelCoverTypography(t *testing.T) {
	prompt := buildCoverPrompt(coverRequest{
		Mode:        "novel",
		Title:       "北境书塔",
		Synopsis:    "少年在雨夜进入一座会吞掉书名的书塔。",
		Protagonist: "红斗篷装订师",
		Genre:       "玄幻 / 悬疑",
		CoverTitle:  "北境书塔",
	})

	for _, kw := range []string{"小说封面", "封面文字", "书名", "北境书塔"} {
		require.Contains(t, prompt, kw)
	}
	require.NotContains(t, prompt, "不要出现任何文字")
}

func TestPolishCoverPromptCustomUsesTextModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := startStudioChatGatewayRecorder(t, `{"prompt":"竖版网络小说封面，雨夜赛博城市，主标题区醒目，封面文字清晰有设计感。"}`)
	h, userID := newStudioTextHandlerWithConfig(t, "studio_cover_polish_custom", "gpt-5.5")

	rec := performStudioCoverPromptPolishRequest(t, h, userID, `{
		"mode":"custom",
		"prompt":"赛博朋克雨夜街道"
	}`)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Prompt string `json:"prompt"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Contains(t, resp.Data.Prompt, "竖版网络小说封面")
	require.Contains(t, resp.Data.Prompt, "封面文字")

	requests := recorder.snapshot()
	require.Len(t, requests, 1)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(requests[0].Body, &payload))
	require.Equal(t, "gpt-5.5", payload["model"])
	instructions, _ := payload["instructions"].(string)
	require.NotEmpty(t, strings.TrimSpace(instructions))
	require.Contains(t, string(requests[0].Body), "赛博朋克雨夜街道")
	require.Contains(t, string(requests[0].Body), "小说封面")
}

func TestPolishCoverPromptNovelCompletesFieldsFromSynopsis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	startStudioChatGatewayRecorder(t, `{"protagonist":"林见微，冷感书塔修复师，红斗篷","genre":"玄幻 / 悬疑","mood":"冷色悬疑，雨夜压迫感","key_scene":"雨夜书塔门前，书页化作群鸟","cover_title":"北境书塔"}`)
	h, userID := newStudioTextHandlerWithConfig(t, "studio_cover_polish_novel", "gpt-5.5")

	rec := performStudioCoverPromptPolishRequest(t, h, userID, `{
		"mode":"novel",
		"title":"北境书塔",
		"synopsis":"少年在雨夜进入一座会吞掉书名的书塔。"
	}`)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Protagonist string `json:"protagonist"`
			Genre       string `json:"genre"`
			Mood        string `json:"mood"`
			KeyScene    string `json:"key_scene"`
			CoverTitle  string `json:"cover_title"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Contains(t, resp.Data.Protagonist, "林见微")
	require.Contains(t, resp.Data.Genre, "玄幻")
	require.Contains(t, resp.Data.Mood, "雨夜")
	require.Contains(t, resp.Data.KeyScene, "书塔")
	require.Equal(t, "北境书塔", resp.Data.CoverTitle)
}

func TestGenerateCoverQueuesGPTImageRequestsAsSingleImageJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := startStudioCoverGatewayRecorder(t)
	h, userID := newStudioCoverHandlerWithConfig(t, "studio_cover_queue", "gpt-image-2")

	rec := performStudioCoverRequest(t, h, userID, `{
		"mode":"novel",
		"synopsis":"a rain city that eats titles",
		"protagonist":"a bookbinder in a red coat",
		"size":"1024x1536",
		"count":3
	}`)

	require.Equal(t, http.StatusOK, rec.Code)
	requests := recorder.snapshot()
	require.Len(t, requests, 3)
	for _, req := range requests {
		require.Equal(t, "/v1/images/generations", req.Path)
		require.Contains(t, req.ContentType, "application/json")
		var payload map[string]any
		require.NoError(t, json.Unmarshal(req.Body, &payload))
		require.Equal(t, "gpt-image-2", payload["model"])
		require.Equal(t, float64(1), payload["n"])
		require.Equal(t, "1024x1536", payload["size"])
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Covers []struct {
				ID  string `json:"id"`
				URL string `json:"url"`
			} `json:"covers"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Covers, 3)
}

func TestGenerateCoverWithReferenceImageUsesMultipartEditsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := startStudioCoverGatewayRecorder(t)
	h, userID := newStudioCoverHandlerWithConfig(t, "studio_cover_reference_upload", "gpt-image-2")
	refImage := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("reference-bytes"))

	rec := performStudioCoverRequest(t, h, userID, `{
		"mode":"custom",
		"prompt":"use this composition",
		"ref_image":"`+refImage+`",
		"size":"1024x1024",
		"count":1
	}`)

	require.Equal(t, http.StatusOK, rec.Code)
	requests := recorder.snapshot()
	require.Len(t, requests, 1)
	req := requests[0]
	require.Equal(t, "/v1/images/edits", req.Path)
	mediaType, params, err := mime.ParseMediaType(req.ContentType)
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mediaType)

	reader := multipart.NewReader(bytes.NewReader(req.Body), params["boundary"])
	fields := map[string]string{}
	var upload []byte
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(part)
		require.NoError(t, err)
		if part.FileName() != "" {
			require.Equal(t, "image", part.FormName())
			require.Equal(t, "reference.png", part.FileName())
			require.Equal(t, "image/png", part.Header.Get("Content-Type"))
			upload = data
			continue
		}
		fields[part.FormName()] = string(data)
	}

	require.Equal(t, "gpt-image-2", fields["model"])
	require.Contains(t, fields["prompt"], "use this composition")
	require.Contains(t, fields["prompt"], "小说封面")
	require.Contains(t, fields["prompt"], "封面文字")
	require.Equal(t, "1", fields["n"])
	require.Equal(t, "1024x1024", fields["size"])
	require.Equal(t, []byte("reference-bytes"), upload)
}

func TestStartCoverJobReturnsRunningAndPollsUntilSucceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
	})
	startStudioCoverGatewayRecorderWithRelease(t, release)
	h, userID := newStudioCoverHandlerWithConfig(t, "studio_cover_async_job", "gpt-image-2")

	rec := performStudioCoverJobStart(t, h, userID, `{
		"mode":"novel",
		"synopsis":"a rain city that eats titles",
		"protagonist":"a bookbinder in a red coat",
		"size":"1024x1536",
		"count":1
	}`)

	require.Equal(t, http.StatusAccepted, rec.Code)
	var startResp struct {
		Code int `json:"code"`
		Data struct {
			JobID    string `json:"job_id"`
			Status   string `json:"status"`
			Progress int    `json:"progress"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &startResp))
	require.Equal(t, 0, startResp.Code)
	require.NotEmpty(t, startResp.Data.JobID)
	require.Equal(t, "running", startResp.Data.Status)

	pending := performStudioCoverJobGet(t, h, userID, startResp.Data.JobID)
	require.Equal(t, http.StatusOK, pending.Code)
	require.Contains(t, pending.Body.String(), `"status":"running"`)

	releaseOnce.Do(func() { close(release) })
	require.Eventually(t, func() bool {
		polled := performStudioCoverJobGet(t, h, userID, startResp.Data.JobID)
		if polled.Code != http.StatusOK {
			return false
		}
		var pollResp struct {
			Data struct {
				Status string `json:"status"`
				Result struct {
					Covers []struct {
						URL string `json:"url"`
					} `json:"covers"`
				} `json:"result"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(polled.Body.Bytes(), &pollResp))
		return pollResp.Data.Status == "succeeded" &&
			len(pollResp.Data.Result.Covers) == 1 &&
			strings.HasPrefix(pollResp.Data.Result.Covers[0].URL, "data:image/png;base64,")
	}, time.Second, 10*time.Millisecond)
}

func TestStudioCoverErrorMessageLocalizesGatewayBalanceError(t *testing.T) {
	err := &studioCoverHTTPError{
		StatusCode: http.StatusForbidden,
		Body:       []byte(`{"error":{"message":"Insufficient account balance"}}`),
	}

	require.Equal(t, "账户余额不足，请先充值或联系管理员增加余额", studioCoverErrorMessage(err))
}

func TestBuildTeardownPromptIncludesContentAndTone(t *testing.T) {
	system, user := buildTeardownPrompt(teardownRequest{
		Content: "少年得到一座会生长城市的书塔",
		Title:   "北境书塔",
		Genre:   "玄幻",
		Tone:    "savage",
	})
	if !strings.Contains(system, "JSON") {
		t.Fatalf("system prompt should require JSON output: %q", system)
	}
	for _, kw := range []string{"北境书塔", "玄幻", "少年得到一座会生长城市的书塔", "毒舌"} {
		if !strings.Contains(user, kw) {
			t.Fatalf("user prompt missing %q: %s", kw, user)
		}
	}
}

func TestTeardownToneHint(t *testing.T) {
	if !strings.Contains(teardownToneHint("gentle"), "鼓励") {
		t.Fatal("gentle tone should be encouraging")
	}
	if !strings.Contains(teardownToneHint("savage"), "毒舌") {
		t.Fatal("savage tone should be 毒舌")
	}
}

func TestBuildScriptPromptByForm(t *testing.T) {
	_, user := buildScriptPrompt(scriptRequest{Content: "主角推开书塔大门", Form: "storyboard", Episodes: 3})
	for _, kw := range []string{"分镜", "主角推开书塔大门", "约 3 集"} {
		if !strings.Contains(user, kw) {
			t.Fatalf("script prompt missing %q: %s", kw, user)
		}
	}
	_, short := buildScriptPrompt(scriptRequest{Content: "x", Form: "short"})
	if !strings.Contains(short, "短剧") {
		t.Fatalf("short form should mention 短剧: %s", short)
	}
}

func TestBuildHotspotPromptIncludesBenchmarkAndJSON(t *testing.T) {
	system, user := buildHotspotPrompt(hotspotRequest{
		Title:     "北境书塔",
		Genre:     "玄幻",
		Goal:      "new-book",
		Content:   "主角被家族放逐后得到旧神书塔",
		Benchmark: "同题材爆款前三章：压迫、反杀、资源差",
	})
	if !strings.Contains(system, "JSON") {
		t.Fatalf("system prompt should require JSON output: %q", system)
	}
	for _, kw := range []string{"北境书塔", "玄幻", "旧神书塔", "同题材爆款", "market_score", "radar", "actions"} {
		if !strings.Contains(user, kw) {
			t.Fatalf("hotspot prompt missing %q: %s", kw, user)
		}
	}
}

func TestBuildCreativePromptByMode(t *testing.T) {
	_, outline := buildCreativePrompt(creativeRequest{Mode: "outline", Content: "主角得到旧神书塔", Brief: "做成三卷大纲"})
	for _, kw := range []string{"大纲", "三卷大纲", "sections"} {
		if !strings.Contains(outline, kw) {
			t.Fatalf("outline prompt missing %q: %s", kw, outline)
		}
	}
	_, rewrite := buildCreativePrompt(creativeRequest{Mode: "rewrite", Content: "这一章节奏太平"})
	if !strings.Contains(rewrite, "改写") {
		t.Fatalf("rewrite mode should mention 改写: %s", rewrite)
	}
}

func TestImportStudioContentRecordsManualChapters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_import_manual")
	user, err := client.User.Create().
		SetEmail("studio-import@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	h := &StudioHandler{client: client}

	rec := performStudioImportRequest(t, h, user.ID, `{
		"source":"manual",
		"title":"北境书塔",
		"content":"第 1 章 雨夜入塔\n正文一\n\n第 2 章 旧约\n正文二",
		"consent":true
	}`)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "第 1 章 雨夜入塔")
	tasks, err := client.CreationTask.Query().Where(creationtask.UserIDEQ(user.ID), creationtask.TypeEQ("import")).All(ctx)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "北境书塔导入", tasks[0].Title)
	require.Equal(t, "local-importer", tasks[0].Model)
	require.Contains(t, tasks[0].Output, `"content":"正文一"`)
	require.Contains(t, tasks[0].Output, `"content":"正文二"`)
}

func TestStudioFanqieRankReturnsThirtyBooks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_rank")
	user, err := client.User.Create().
		SetEmail("studio-fanqie-rank@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	h := &StudioHandler{client: client}

	rec := performStudioFanqieRankRequest(t, h, user.ID, "male")

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Channel string `json:"channel"`
			Books   []struct {
				Rank  int    `json:"rank"`
				Title string `json:"title"`
			} `json:"books"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "male", resp.Data.Channel)
	require.Len(t, resp.Data.Books, 30)
	require.Equal(t, 1, resp.Data.Books[0].Rank)
	require.NotEmpty(t, resp.Data.Books[0].Title)
}

func TestStudioFanqieRankUsesPersistedSnapshotWithoutOfficialFetch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetFanqieRankCacheForTest()
	t.Cleanup(resetFanqieRankCacheForTest)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_rank_db_cache")
	createFanqieRankSnapshotTableForTest(t, client)
	user, err := client.User.Create().
		SetEmail("studio-fanqie-rank-db-cache@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)

	cachedBooks := []fanqieBook{{
		ID:          "db-1001",
		Rank:        1,
		Title:       "数据库缓存小说",
		Author:      "缓存作者",
		Category:    "玄幻",
		Status:      "连载中",
		WordCount:   "100万字",
		Score:       "热度 999",
		Description: "来自数据库快照",
		SourceURL:   "https://fanqienovel.com/page/1001",
		Tags:        []string{"数据库缓存"},
	}}
	rawBooks, err := json.Marshal(cachedBooks)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, `
INSERT INTO fanqie_rank_snapshots (channel, books, source, fetched_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		string(fanqieRankHot), string(rawBooks), "fanqie-official", time.Now().UTC(), time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	oldFetcher := fetchFanqieRankBooksFromOfficialFunc
	fetchFanqieRankBooksFromOfficialFunc = func(context.Context, fanqieRankChannel) ([]fanqieBook, error) {
		t.Fatal("official fetch should not run when a fresh database snapshot exists")
		return nil, nil
	}
	t.Cleanup(func() { fetchFanqieRankBooksFromOfficialFunc = oldFetcher })

	h := &StudioHandler{client: client}
	rec := performStudioFanqieRankRequest(t, h, user.ID, "hot")

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Source string       `json:"source"`
			Books  []fanqieBook `json:"books"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "fanqie-db-cache", resp.Data.Source)
	require.Len(t, resp.Data.Books, 1)
	require.Equal(t, "数据库缓存小说", resp.Data.Books[0].Title)
}

func TestStudioFanqieRankBackfillsPersistedSnapshotCovers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetFanqieRankCacheForTest()
	t.Cleanup(resetFanqieRankCacheForTest)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_rank_db_cover_backfill")
	createFanqieRankSnapshotTableForTest(t, client)
	user, err := client.User.Create().
		SetEmail("studio-fanqie-rank-db-cover-backfill@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)

	cachedBooks := []fanqieBook{{
		ID:          "db-no-cover-1001",
		Rank:        1,
		Title:       "旧缓存无封面小说",
		Author:      "缓存作者",
		Category:    "玄幻",
		Status:      "连载中",
		WordCount:   "100万字",
		Score:       "热度 999",
		Description: "来自旧数据库快照",
		SourceURL:   "https://fanqienovel.com/page/1001",
		Tags:        []string{"数据库缓存"},
	}}
	rawBooks, err := json.Marshal(cachedBooks)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, `
INSERT INTO fanqie_rank_snapshots (channel, books, source, fetched_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		string(fanqieRankHot), string(rawBooks), "fanqie-rank-cache", time.Now().UTC(), time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err)

	oldFetcher := fetchFanqieRankBooksFromOfficialFunc
	fetchFanqieRankBooksFromOfficialFunc = func(context.Context, fanqieRankChannel) ([]fanqieBook, error) {
		t.Fatal("official fetch should not run when a fresh database snapshot exists")
		return nil, nil
	}
	t.Cleanup(func() { fetchFanqieRankBooksFromOfficialFunc = oldFetcher })

	h := &StudioHandler{client: client}
	rec := performStudioFanqieRankRequest(t, h, user.ID, "hot")

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Source string       `json:"source"`
			Books  []fanqieBook `json:"books"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "fanqie-db-cache", resp.Data.Source)
	require.Len(t, resp.Data.Books, 1)
	require.Equal(t, "旧缓存无封面小说", resp.Data.Books[0].Title)
	require.Empty(t, resp.Data.Books[0].CoverURL)
}

func TestStudioFanqieRankForceRefreshPersistsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetFanqieRankCacheForTest()
	t.Cleanup(resetFanqieRankCacheForTest)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_rank_force_refresh")
	createFanqieRankSnapshotTableForTest(t, client)
	user, err := client.User.Create().
		SetEmail("studio-fanqie-rank-force-refresh@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)

	fetchedBooks := make([]fanqieBook, 30)
	for i := range fetchedBooks {
		fetchedBooks[i] = fanqieBook{
			ID:          fmt.Sprintf("official-%02d", i+1),
			Rank:        i + 1,
			Title:       fmt.Sprintf("官方刷新小说 %02d", i+1),
			Author:      "官方作者",
			Category:    "都市",
			Status:      "连载中",
			WordCount:   "80万字",
			Score:       "热度 900",
			Description: "来自强制刷新",
			SourceURL:   fmt.Sprintf("https://fanqienovel.com/page/%d", 9000+i),
			Tags:        []string{"官方刷新"},
		}
	}
	oldFetcher := fetchFanqieRankBooksFromOfficialFunc
	fetchFanqieRankBooksFromOfficialFunc = func(context.Context, fanqieRankChannel) ([]fanqieBook, error) {
		return fetchedBooks, nil
	}
	t.Cleanup(func() { fetchFanqieRankBooksFromOfficialFunc = oldFetcher })

	h := &StudioHandler{client: client}
	rec := performStudioFanqieRankRequestWithForce(t, h, user.ID, "hot", true)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Source string       `json:"source"`
			Books  []fanqieBook `json:"books"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "fanqie-official", resp.Data.Source)
	require.Len(t, resp.Data.Books, 30)
	require.Equal(t, "官方刷新小说 01", resp.Data.Books[0].Title)

	rows, err := client.QueryContext(ctx, `SELECT books, source FROM fanqie_rank_snapshots WHERE channel = ?`, string(fanqieRankHot))
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next())
	var storedBooksRaw string
	var storedSource string
	require.NoError(t, rows.Scan(&storedBooksRaw, &storedSource))
	require.Equal(t, "fanqie-official", storedSource)
	require.Contains(t, storedBooksRaw, "官方刷新小说 01")
}

func TestStudioFanqieRankPersistsFallbackWhenOfficialUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetFanqieRankCacheForTest()
	t.Cleanup(resetFanqieRankCacheForTest)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_rank_fallback_persist")
	createFanqieRankSnapshotTableForTest(t, client)
	user, err := client.User.Create().
		SetEmail("studio-fanqie-rank-fallback-persist@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)

	oldFetcher := fetchFanqieRankBooksFromOfficialFunc
	fetchFanqieRankBooksFromOfficialFunc = func(context.Context, fanqieRankChannel) ([]fanqieBook, error) {
		return nil, errors.New("official unavailable")
	}
	t.Cleanup(func() { fetchFanqieRankBooksFromOfficialFunc = oldFetcher })

	h := &StudioHandler{client: client}
	rec := performStudioFanqieRankRequest(t, h, user.ID, "hot")

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Source string       `json:"source"`
			Books  []fanqieBook `json:"books"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "fanqie-rank-cache", resp.Data.Source)
	require.Len(t, resp.Data.Books, 30)
	require.Empty(t, resp.Data.Books[0].CoverURL)

	rows, err := client.QueryContext(ctx, `SELECT books, source FROM fanqie_rank_snapshots WHERE channel = ?`, string(fanqieRankHot))
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next())
	var storedBooksRaw string
	var storedSource string
	require.NoError(t, rows.Scan(&storedBooksRaw, &storedSource))
	require.Equal(t, "fanqie-rank-cache", storedSource)
	require.Contains(t, storedBooksRaw, "热榜样本")
}

func TestLiveFanqieRankBooksFallsBackWhenOfficialSourceIsSlow(t *testing.T) {
	oldFetcher := fetchFanqieRankBooksFromOfficialFunc
	oldTimeout := fanqieRankOfficialFetchTimeout
	fanqieRankOfficialFetchTimeout = 80 * time.Millisecond
	fetchFanqieRankBooksFromOfficialFunc = func(ctx context.Context, _ fanqieRankChannel) ([]fanqieBook, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	t.Cleanup(func() {
		fetchFanqieRankBooksFromOfficialFunc = oldFetcher
		fanqieRankOfficialFetchTimeout = oldTimeout
		resetFanqieRankCacheForTest()
	})

	resetFanqieRankCacheForTest()

	started := time.Now()
	books, source, _ := liveFanqieRankBooks(context.Background(), nil, fanqieRankHot, false)

	require.Less(t, time.Since(started), 2*time.Second)
	require.Equal(t, "fanqie-rank-cache", source)
	require.Len(t, books, 30)
}

func TestLiveFanqieRankBooksAllowsNormalOfficialLatency(t *testing.T) {
	oldFetcher := fetchFanqieRankBooksFromOfficialFunc
	oldTimeout := fanqieRankOfficialFetchTimeout
	fanqieRankOfficialFetchTimeout = 3 * time.Second
	fetchFanqieRankBooksFromOfficialFunc = func(ctx context.Context, _ fanqieRankChannel) ([]fanqieBook, error) {
		select {
		case <-time.After(1700 * time.Millisecond):
			books := make([]fanqieBook, 30)
			for i := range books {
				books[i] = fanqieBook{
					ID:          fmt.Sprintf("latency-%02d", i+1),
					Rank:        i + 1,
					Title:       fmt.Sprintf("官方延迟榜单 %02d", i+1),
					Author:      "官方作者",
					Category:    "都市脑洞",
					Status:      "连载中",
					WordCount:   "88万字",
					Score:       "官方实时榜",
					Description: "正常网络延迟内返回",
					CoverURL:    "https://p3-reading-sign.fqnovelpic.com/novel-pic/p2o123~tplv.jpg",
					SourceURL:   fmt.Sprintf("https://fanqienovel.com/page/%d", 7000+i),
				}
			}
			return books, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	t.Cleanup(func() {
		fetchFanqieRankBooksFromOfficialFunc = oldFetcher
		fanqieRankOfficialFetchTimeout = oldTimeout
		resetFanqieRankCacheForTest()
	})

	resetFanqieRankCacheForTest()

	books, source, _ := liveFanqieRankBooks(context.Background(), nil, fanqieRankHot, false)

	require.Equal(t, "fanqie-official", source)
	require.Len(t, books, 30)
	require.Equal(t, "官方延迟榜单 01", books[0].Title)
	require.NotEmpty(t, books[0].CoverURL)
}

func TestStudioFanqieRankRewritesOfficialCoverURLsToLocalCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetFanqieCoverCachesForTest()
	resetFanqieRankCacheForTest()

	oldFetcher := fetchFanqieRankBooksFromOfficialFunc
	fetchFanqieRankBooksFromOfficialFunc = func(context.Context, fanqieRankChannel) ([]fanqieBook, error) {
		books := make([]fanqieBook, 30)
		for i := range books {
			books[i] = fanqieBook{
				ID:          fmt.Sprintf("100%d", i),
				Rank:        i + 1,
				Title:       fmt.Sprintf("缓存封面样本 %d", i+1),
				Author:      "作者",
				Category:    "都市脑洞",
				Status:      "连载中",
				WordCount:   "88万字",
				Score:       "官方实时榜",
				Description: "封面需要走本地缓存",
				CoverURL:    "https://p3-reading-sign.fqnovelpic.com/novel-pic/p2o123~tplv.jpg",
				SourceURL:   "https://fanqienovel.com/page/1001",
				Tags:        []string{"官方实时榜"},
			}
		}
		return books, nil
	}
	t.Cleanup(func() {
		fetchFanqieRankBooksFromOfficialFunc = oldFetcher
		resetFanqieCoverCachesForTest()
		resetFanqieRankCacheForTest()
	})

	client := newStudioHandlerTestClient(t, "studio_fanqie_cover_rewrite")
	user, err := client.User.Create().
		SetEmail("studio-fanqie-cover-rewrite@example.com").
		SetPasswordHash("hash").
		Save(context.Background())
	require.NoError(t, err)
	h := &StudioHandler{client: client}

	rec := performStudioFanqieRankRequest(t, h, user.ID, "hot")

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Books []struct {
				CoverURL string `json:"cover_url"`
			} `json:"books"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotEmpty(t, resp.Data.Books)
	require.True(t, strings.HasPrefix(resp.Data.Books[0].CoverURL, "/api/v1/studio/fanqie/covers/"), resp.Data.Books[0].CoverURL)
	require.NotContains(t, resp.Data.Books[0].CoverURL, "fqnovelpic.com")
}

func TestRegisterFanqieCoverSourceRejectsGeneratedFallbackCoverScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetFanqieCoverCachesForTest()
	t.Cleanup(resetFanqieCoverCachesForTest)

	key, ok := registerFanqieCoverSourceForTest("fanqie-generated-cover://local/cover.svg?title=fake")

	require.False(t, ok)
	require.Empty(t, key)
}

func TestServeFanqieCoverCachesImageBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetFanqieCoverCachesForTest()
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		require.Equal(t, "/cover.jpg", r.URL.Path)
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("cover-bytes"))
	}))
	defer upstream.Close()
	t.Cleanup(resetFanqieCoverCachesForTest)

	key, ok := registerFanqieCoverSourceForTest(upstream.URL + "/cover.jpg")
	require.True(t, ok)
	h := &StudioHandler{}

	first := performStudioFanqieCoverRequest(t, h, key)
	second := performStudioFanqieCoverRequest(t, h, key)

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, "image/jpeg", first.Header().Get("Content-Type"))
	require.Equal(t, "cover-bytes", first.Body.String())
	require.Equal(t, "miss", first.Header().Get("X-Fanqie-Cover-Cache"))
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, "cover-bytes", second.Body.String())
	require.Equal(t, "hit", second.Header().Get("X-Fanqie-Cover-Cache"))
	require.Equal(t, 1, hits)
}

func TestServeFanqieCoverRejectsRedirectToUntrustedHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetFanqieCoverCachesForTest()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.com/cover.jpg", http.StatusFound)
	}))
	defer upstream.Close()
	t.Cleanup(resetFanqieCoverCachesForTest)

	key, ok := registerFanqieCoverSourceForTest(upstream.URL + "/cover.jpg")
	require.True(t, ok)
	h := &StudioHandler{}

	rec := performStudioFanqieCoverRequest(t, h, key)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Empty(t, rec.Header().Get("X-Fanqie-Cover-Cache"))
	_, ok = getCachedFanqieCoverBytes(key)
	require.False(t, ok)
}

func TestParseFanqieRankHTMLReturnsOfficialBooks(t *testing.T) {
	html := `<script>window.__INITIAL_STATE__={"rank":{"book_list":[{"bookId":"1001","bookName":"新鲜热榜书","author":"作者A","abstract":"开篇强钩子","categoryV2":"都市脑洞","creationStatus":"1","wordNumber":"880000","thumbUri":"https:\/\/img.example.com\/cover.jpg","currentPos":1,"read_count":"12345"},{"bookId":"1002","bookName":"第二本","author":"作者B","abstract":"反转密集","categoryV2":"悬疑脑洞","creationStatus":"0","wordNumber":"1200000","currentPos":2,"read_count":"999"}]}};</script>`

	books, err := parseFanqieRankHTML(fanqieRankHot, html)

	require.NoError(t, err)
	require.Len(t, books, 2)
	require.Equal(t, "1001", books[0].ID)
	require.Equal(t, "新鲜热榜书", books[0].Title)
	require.Equal(t, "作者A", books[0].Author)
	require.Equal(t, "连载中", books[0].Status)
	require.Equal(t, "https://fanqienovel.com/page/1001", books[0].SourceURL)
	require.Equal(t, "阅读 12345", books[0].Score)
}

func TestParseFanqieRankAPIResponseReturnsLiveBooks(t *testing.T) {
	body := []byte(`{"code":0,"data":{"book_list":[{"bookId":"1001","bookName":"Live Book","author":"Author A","abstract":"Live intro","categoryV2":"Urban","creationStatus":"1","wordNumber":"880000","thumbUri":"https://img.example.com/cover.jpg","currentPos":1,"read_count":"12345"},{"bookId":"1002","bookName":"Second Book","author":"Author B","abstract":"Second intro","category":"","creationStatus":"0","wordNumber":"1200000","currentPos":2,"read_count":"999"}]}}`)

	books, err := parseFanqieRankAPIResponse(fanqieRankHot, "Urban", body)

	require.NoError(t, err)
	require.Len(t, books, 2)
	require.Equal(t, "1001", books[0].ID)
	require.Equal(t, "Live Book", books[0].Title)
	require.Equal(t, "Urban", books[0].Category)
	require.Equal(t, "连载中", books[0].Status)
	require.Equal(t, "88万字", books[0].WordCount)
	require.Equal(t, "在读 12345", books[0].Score)
	require.Equal(t, "https://fanqienovel.com/page/1001", books[0].SourceURL)
	require.Equal(t, "Urban", books[1].Category)
}

func TestEnrichFanqieObfuscatedRankBooksUsesBookSummary(t *testing.T) {
	oldSummary := fetchFanqieBookSummary
	fetchFanqieBookSummary = func(_ context.Context, book fanqieBook) (fanqieBook, error) {
		require.Equal(t, "7143038691944959011", book.ID)
		return fanqieBook{
			ID:          book.ID,
			Title:       "十日终焉",
			Author:      "杀虫队队员",
			Category:    "悬疑脑洞 / 推理",
			Status:      "已完结",
			WordCount:   "320万字",
			Score:       "官方详情",
			Description: "死亡游戏与规则怪谈。",
			CoverURL:    "https://example.com/cover.jpg",
			SourceURL:   "https://fanqienovel.com/page/" + book.ID,
			Tags:        []string{"官方详情", "悬疑脑洞"},
		}, nil
	}
	t.Cleanup(func() { fetchFanqieBookSummary = oldSummary })

	books := enrichFanqieObfuscatedRankBooks(context.Background(), []fanqieBook{{
		ID:          "7143038691944959011",
		Rank:        7,
		Title:       "\ue41f\ue475终焉",
		Author:      "杀虫队队\ue4d9",
		Category:    "悬疑脑洞",
		Status:      "连载中",
		WordCount:   "320万字",
		Score:       "在读 2164597",
		Description: "\ue246\ue3f1年度榜单作品",
		SourceURL:   "https://fanqienovel.com/page/7143038691944959011",
		Tags:        []string{"热榜", "官方实时榜"},
	}})

	require.Len(t, books, 1)
	require.Equal(t, 7, books[0].Rank)
	require.Equal(t, "在读 2164597", books[0].Score)
	require.Equal(t, "十日终焉", books[0].Title)
	require.Equal(t, "杀虫队队员", books[0].Author)
	require.Equal(t, "死亡游戏与规则怪谈。", books[0].Description)
	require.False(t, containsPrivateUseRune(books[0].Title+books[0].Author+books[0].Description))
}

func TestEnrichFanqieObfuscatedRankBooksRemovesPrivateUseTextWhenSummaryFails(t *testing.T) {
	oldSummary := fetchFanqieBookSummary
	fetchFanqieBookSummary = func(_ context.Context, book fanqieBook) (fanqieBook, error) {
		return fanqieBook{}, fmt.Errorf("official detail unavailable")
	}
	t.Cleanup(func() { fetchFanqieBookSummary = oldSummary })

	books := enrichFanqieObfuscatedRankBooks(context.Background(), []fanqieBook{{
		ID:          "7143038691944959011",
		Rank:        28,
		Title:       "糟糕！\ue121\ue42c\ue44d鬼包围\ue436",
		Author:      "\ue3fc\ue166\ue417巴",
		Category:    "悬疑脑洞",
		Status:      "连载中",
		WordCount:   "-",
		Score:       "在读 358634",
		Description: "\ue246\ue3f1官方摘要",
		SourceURL:   "https://fanqienovel.com/page/7143038691944959011",
		Tags:        []string{"热榜", "\ue123官方实时榜"},
	}})

	require.Len(t, books, 1)
	require.Equal(t, 28, books[0].Rank)
	require.Equal(t, "番茄作品 7143038691944959011", books[0].Title)
	require.Equal(t, "番茄小说", books[0].Author)
	require.Equal(t, "悬疑脑洞", books[0].Category)
	require.Equal(t, "官方详情暂时不可用，已隐藏番茄加密字体字段；可打开作品页或稍后刷新获取完整信息。", books[0].Description)
	require.False(t, fanqieBookHasPrivateUseText(books[0]))
}
func TestParseFanqieBookInfoAPIResponseParsesStringifiedCategoryList(t *testing.T) {
	body := []byte(`{"code":0,"data":{"bookId":"7143038691944959011","bookName":"十日终焉","author":"杀虫队队员","abstract":"死亡游戏与规则怪谈。","categoryV2":"[{\"Name\":\"悬疑脑洞\",\"MainCategory\":true},{\"Name\":\"推理\"}]","creationStatus":"0","wordNumber":"3201288","thumbUrl":"https://example.com/cover.jpg","readCount":"2175190"}}`)

	book, err := parseFanqieBookInfoAPIResponse(body, fanqieBook{})

	require.NoError(t, err)
	require.Equal(t, "十日终焉", book.Title)
	require.Equal(t, "悬疑脑洞 / 推理", book.Category)
	require.Equal(t, "320万字", book.WordCount)
	require.Equal(t, "https://example.com/cover.jpg", book.CoverURL)
}

func TestParseFanqieBookInfoAPIResponseDoesNotTruncateSignedCoverURL(t *testing.T) {
	cover := "https://p9-novel-sign.byteimg.com/novel-pic/4900f950c7af7f82fdc14cf528e0e288~tplv-resize:225:300.image?lk3s=191c1ecc&x-expires=1781773667&x-signature=q1WdTShPZ29y5QFD0NLiqf3JPDQ%3D"
	body := []byte(`{"code":0,"data":{"bookId":"7143038691944959011","bookName":"Live Book","thumbUrl":"` + cover + `"}}`)

	book, err := parseFanqieBookInfoAPIResponse(body, fanqieBook{})

	require.NoError(t, err)
	require.Equal(t, cover, book.CoverURL)
}

func TestFilterFanqieBooksKeepsOfficialSearchResultsDownloadable(t *testing.T) {
	books := []fanqieBook{
		{ID: "1001", Rank: 1, Title: "Live Book", Author: "Author A", Category: "Urban", SourceURL: "https://fanqienovel.com/page/1001"},
		{ID: "1002", Rank: 2, Title: "Second Book", Author: "Author B", Category: "Mystery", SourceURL: "https://fanqienovel.com/page/1002"},
	}

	results := filterFanqieBooks("author a", books, 20)

	require.Len(t, results, 1)
	require.Equal(t, 1, results[0].Rank)
	require.Equal(t, "Live Book", results[0].Title)
	require.Equal(t, "搜索命中", results[0].Score)
	require.Equal(t, "https://fanqienovel.com/page/1001", results[0].SourceURL)
}

func TestParseFanqieReaderHTMLExtractsChapterContent(t *testing.T) {
	html := `<script>window.__INITIAL_STATE__={"reader":{"chapterData":{"itemId":"c1","title":"Chapter 1","content":"<p>First paragraph</p><p>Second&nbsp;paragraph</p>"}}};</script>`

	chapter, err := parseFanqieReaderHTML(html, "https://fanqienovel.com/reader/c1")

	require.NoError(t, err)
	require.Equal(t, "c1", chapter.ItemID)
	require.Equal(t, "Chapter 1", chapter.Title)
	require.Equal(t, "First paragraph\nSecond paragraph", chapter.Content)
	require.True(t, chapter.Readable)
	require.Equal(t, "readable", chapter.DecodeStatus)
}

func TestParseFanqieReaderHTMLDetectsCaptchaPage(t *testing.T) {
	html := `<html><head><title>验证码中间页</title><script src="https://example.com/captcha/index.js"></script></head></html>`

	_, err := parseFanqieReaderHTML(html, "https://fanqienovel.com/reader/c1")

	require.Error(t, err)
	require.Contains(t, err.Error(), "验证码")
}

func TestParseFanqieReaderHTMLParsesNormalReaderPageWithCaptchaBootstrap(t *testing.T) {
	html := `<html><head><script src="https://example.com/captcha/index.js"></script></head><body><script>window.__INITIAL_STATE__={"reader":{"chapterData":{"itemId":"c1","title":"Chapter 1","content":"<p>Visible body</p>"}}};</script></body></html>`

	chapter, err := parseFanqieReaderHTML(html, "https://fanqienovel.com/reader/c1")

	require.NoError(t, err)
	require.Equal(t, "Visible body", chapter.Content)
	require.Equal(t, "readable", chapter.DecodeStatus)
}

func TestParseFanqiePageHTMLSkipsLockedChapters(t *testing.T) {
	html := `<script>window.__INITIAL_STATE__={"page":{"bookId":"1001","bookName":"Live Book","chapterListWithVolume":[[{"itemId":"c1","title":"Locked Chapter","realChapterOrder":"1","isChapterLock":true}]]}};</script>`

	detail, err := parseFanqiePageHTML(html, "https://fanqienovel.com/page/1001", fanqieBook{})

	require.NoError(t, err)
	require.Len(t, detail.Chapters, 1)
	require.Equal(t, "https://fanqienovel.com/reader/c1", detail.Chapters[0].SourceURL)
	require.NotEmpty(t, detail.Chapters[0].DecodeStatus)
}

func TestPopulateFanqieChapterContentsReportsCaptchaBlocked(t *testing.T) {
	detail := fanqieBookDetail{
		Book: fanqieBook{ID: "1001", Title: "Live Book", SourceURL: "https://fanqienovel.com/page/1001"},
		Chapters: []fanqieChapter{
			{Index: 1, ItemID: "c1", Title: "Chapter 1", SourceURL: "https://fanqienovel.com/reader/c1"},
		},
		Source: "fixture",
	}
	fetcher := func(_ context.Context, _ string) (string, error) {
		return `<html><head><title>验证码中间页</title><script src="https://example.com/captcha/index.js"></script></head></html>`, nil
	}

	result := populateFanqieChapterContents(context.Background(), detail, fetcher)
	download := buildFanqieDownloadResult(result)

	require.False(t, result.ReadableContent)
	require.Equal(t, "browser_required", download.DecodeStatus)
	require.Contains(t, strings.Join(download.Notes, "\n"), "验证码")
}

func TestPopulateFanqieChapterContentsStopsWhenReaderRequiresBrowserVerification(t *testing.T) {
	detail := fanqieBookDetail{
		Book: fanqieBook{ID: "1001", Title: "Live Book", SourceURL: "https://fanqienovel.com/page/1001"},
		Chapters: []fanqieChapter{
			{Index: 1, ItemID: "c1", Title: "Chapter 1", SourceURL: "https://fanqienovel.com/reader/c1"},
			{Index: 2, ItemID: "c2", Title: "Chapter 2", SourceURL: "https://fanqienovel.com/reader/c2"},
			{Index: 3, ItemID: "c3", Title: "Chapter 3", SourceURL: "https://fanqienovel.com/reader/c3"},
		},
		Source: "fixture",
	}
	fetches := 0
	fetcher := func(_ context.Context, _ string) (string, error) {
		fetches++
		return `<html><head><title>验证码中间页</title><script src="https://example.com/captcha/index.js"></script></head></html>`, nil
	}

	result := populateFanqieChapterContents(context.Background(), detail, fetcher)
	download := buildFanqieDownloadResult(result)

	require.Equal(t, 1, fetches)
	require.False(t, result.ReadableContent)
	require.True(t, result.Blocked)
	require.Equal(t, "browser_required", download.DecodeStatus)
	require.Contains(t, strings.Join(download.Notes, "\n"), "浏览器安全校验")
	for _, chapter := range result.Chapters {
		require.Contains(t, chapter.DecodeStatus, "浏览器安全校验")
	}
}

func TestPopulateFanqieChapterContentsFetchesReaderPages(t *testing.T) {
	detail := fanqieBookDetail{
		Book: fanqieBook{ID: "1001", Title: "Live Book", SourceURL: "https://fanqienovel.com/page/1001"},
		Chapters: []fanqieChapter{
			{Index: 1, ItemID: "c1", Title: "Chapter 1", SourceURL: "https://fanqienovel.com/reader/c1"},
			{Index: 2, ItemID: "c2", Title: "Chapter 2", SourceURL: "https://fanqienovel.com/reader/c2"},
		},
		Source: "fixture",
	}
	fetcher := func(_ context.Context, rawURL string) (string, error) {
		switch rawURL {
		case "https://fanqienovel.com/reader/c1":
			return `<script>window.__INITIAL_STATE__={"reader":{"chapterData":{"itemId":"c1","title":"Chapter 1","content":"<p>First body</p>"}}};</script>`, nil
		case "https://fanqienovel.com/reader/c2":
			return `<script>window.__INITIAL_STATE__={"reader":{"chapterData":{"itemId":"c2","title":"Chapter 2","content":"<p>Second body</p>"}}};</script>`, nil
		default:
			return "", fmt.Errorf("unexpected url %s", rawURL)
		}
	}

	result := populateFanqieChapterContents(context.Background(), detail, fetcher)
	download := buildFanqieDownloadResult(result)

	require.True(t, result.ReadableContent)
	require.Equal(t, "readable", download.DecodeStatus)
	require.Contains(t, download.Text, "First body")
	require.Contains(t, download.Text, "Second body")
	require.Equal(t, 10, download.Chapters[0].WordCount)
	require.Equal(t, "First body", download.Chapters[0].Content)
	require.Equal(t, "Second body", download.Chapters[1].Content)
}

func TestPopulateFanqieChapterContentsMarksLockedPreviewContent(t *testing.T) {
	detail := fanqieBookDetail{
		Book: fanqieBook{ID: "1001", Title: "Live Book", SourceURL: "https://fanqienovel.com/page/1001"},
		Chapters: []fanqieChapter{
			{Index: 1, ItemID: "c1", Title: "Chapter 1", SourceURL: "https://fanqienovel.com/reader/c1"},
		},
		Source: "fixture",
	}
	fetcher := func(_ context.Context, _ string) (string, error) {
		return `<script>window.__INITIAL_STATE__={"reader":{"chapterData":{"itemId":"c1","title":"Chapter 1","chapterWordNumber":"2000","isChapterLock":true,"content":"<p>Short preview</p><p"}}};</script>`, nil
	}

	result := populateFanqieChapterContents(context.Background(), detail, fetcher)
	download := buildFanqieDownloadResult(result)

	require.True(t, result.PartialContent)
	require.Equal(t, "web_preview", download.DecodeStatus)
	require.Contains(t, strings.Join(download.Notes, "\n"), "网页预览")
}

func TestStudioFanqieSearchFindsNamedNovel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_search")
	user, err := client.User.Create().
		SetEmail("studio-fanqie-search@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	h := &StudioHandler{client: client}

	oldSearch := fetchFanqieSearchBooks
	fetchFanqieSearchBooks = func(_ context.Context, query string) ([]fanqieBook, error) {
		require.Equal(t, "十日终焉", query)
		return []fanqieBook{{
			ID:          "7143038691944959011",
			Rank:        1,
			Title:       "十日终焉",
			Author:      "杀虫队队员",
			Category:    "悬疑脑洞",
			Status:      "已完结",
			WordCount:   "320万字",
			Score:       "官方搜索",
			Description: "死亡游戏与规则怪谈。",
			CoverURL:    "https://example.com/cover.jpg",
			SourceURL:   "https://fanqienovel.com/page/7143038691944959011",
			Tags:        []string{"官方搜索"},
		}}, nil
	}
	t.Cleanup(func() { fetchFanqieSearchBooks = oldSearch })

	rec := performStudioFanqieSearchRequest(t, h, user.ID, "十日终焉")

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "十日终焉")
	var resp struct {
		Data struct {
			Books []struct {
				Title string `json:"title"`
			} `json:"books"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.Books)
}

func TestStudioFanqieSearchFallsBackWhenOfficialSearchFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_search_fallback")
	user, err := client.User.Create().
		SetEmail("studio-fanqie-search-fallback@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	h := &StudioHandler{client: client}

	oldSearch := fetchFanqieSearchBooks
	fetchFanqieSearchBooks = func(_ context.Context, query string) ([]fanqieBook, error) {
		require.Equal(t, "十日终焉", query)
		return nil, errors.New("official fanqie search unavailable")
	}
	t.Cleanup(func() { fetchFanqieSearchBooks = oldSearch })

	rec := performStudioFanqieSearchRequest(t, h, user.ID, "十日终焉")

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data struct {
			Books []struct {
				Title     string `json:"title"`
				SourceURL string `json:"source_url"`
				Score     string `json:"score"`
			} `json:"books"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.Books)
	require.Equal(t, "十日终焉", resp.Data.Books[0].Title)
	require.Contains(t, resp.Data.Books[0].SourceURL, "fanqienovel.com")
	require.Equal(t, "搜索命中", resp.Data.Books[0].Score)
}

func TestSearchLiveFanqieBooksEnrichesDirectPageQuery(t *testing.T) {
	oldSearch := fetchFanqieSearchBooks
	oldSummary := fetchFanqieBookSummary
	fetchFanqieSearchBooks = func(context.Context, string) ([]fanqieBook, error) {
		t.Fatal("direct page query should not call the official search endpoint")
		return nil, nil
	}
	fetchFanqieBookSummary = func(_ context.Context, book fanqieBook) (fanqieBook, error) {
		require.Equal(t, "7143038691944959011", book.ID)
		return fanqieBook{
			ID:          "7143038691944959011",
			Rank:        1,
			Title:       "十日终焉",
			Author:      "杀虫队队员",
			Category:    "悬疑脑洞",
			Status:      "已完结",
			WordCount:   "320万字",
			Score:       "官方详情",
			Description: "死亡游戏与规则怪谈。",
			CoverURL:    "https://example.com/cover.jpg",
			SourceURL:   "https://fanqienovel.com/page/7143038691944959011",
			Tags:        []string{"官方详情"},
		}, nil
	}
	t.Cleanup(func() {
		fetchFanqieSearchBooks = oldSearch
		fetchFanqieBookSummary = oldSummary
	})

	books, err := searchLiveFanqieBooks(context.Background(), "https://fanqienovel.com/page/7143038691944959011")

	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "十日终焉", books[0].Title)
	require.Equal(t, "https://example.com/cover.jpg", books[0].CoverURL)
	require.Equal(t, "死亡游戏与规则怪谈。", books[0].Description)
}

func TestStudioFanqieDownloadRequiresConsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_download_consent")
	user, err := client.User.Create().
		SetEmail("studio-fanqie-download-consent@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	h := &StudioHandler{client: client}

	rec := performStudioFanqieDownloadRequest(t, h, user.ID, `{
		"book":{"id":"7143038691944959011","title":"十日终焉","source_url":"https://fanqienovel.com/page/7143038691944959011"},
		"consent":false
	}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "有权")
}

func TestStudioFanqieDownloadBuildsAuthorizedImportPackage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	client := newStudioHandlerTestClient(t, "studio_fanqie_download_package")
	user, err := client.User.Create().
		SetEmail("studio-fanqie-download@example.com").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	h := &StudioHandler{client: client}

	oldFetch := fetchFanqieBookDetail
	fetchFanqieBookDetail = func(context.Context, fanqieBook) (fanqieBookDetail, error) {
		return fanqieBookDetail{
			Book: fanqieBook{
				ID:          "7143038691944959011",
				Title:       "十日终焉",
				Author:      "杀虫队队员",
				Category:    "悬疑脑洞",
				Status:      "已完结",
				WordCount:   "320万字",
				Description: "规则怪谈与群像博弈。",
				SourceURL:   "https://fanqienovel.com/page/7143038691944959011",
				Tags:        []string{"规则怪谈"},
			},
			Chapters: []fanqieChapter{
				{Index: 1, ItemID: "c1", Title: "第1章 空屋", Content: "封闭房间里醒来。", Readable: true},
				{Index: 2, ItemID: "c2", Title: "第2章 说谎", Content: "游戏规则开始出现。", Readable: true},
			},
			ReadableContent: true,
			Source:          "fixture",
		}, nil
	}
	t.Cleanup(func() { fetchFanqieBookDetail = oldFetch })

	rec := performStudioFanqieDownloadRequest(t, h, user.ID, `{
		"book":{"id":"7143038691944959011","title":"十日终焉","source_url":"https://fanqienovel.com/page/7143038691944959011"},
		"consent":true
	}`)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "十日终焉.txt")
	require.Contains(t, rec.Body.String(), "第1章 空屋")
	require.Contains(t, rec.Body.String(), "封闭房间里醒来。")
	require.Contains(t, rec.Body.String(), "仅限个人备份")
	tasks, err := client.CreationTask.Query().Where(creationtask.UserIDEQ(user.ID), creationtask.TypeEQ("import")).All(ctx)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "十日终焉导入", tasks[0].Title)
	require.Contains(t, tasks[0].Output, `"content":"封闭房间里醒来。"`)
}

func TestStudioFanqieAnalyzeUsesIntroAndFirstTenChapters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := startStudioChatGatewayRecorder(t, `{"title":"十日终焉开篇分析","summary":"封闭空间开局建立规则压力","hooks":["空屋醒来"],"chapter_notes":[{"title":"第1章 空屋","note":"快速建立异常场景"}],"actions":["强化规则代价"]}`)
	h, userID := newStudioTextHandlerWithConfig(t, "studio_fanqie_analyze", "gpt-5.5")

	oldDownloadFetch := fetchFanqieBookDetail
	oldAnalysisFetch := fetchFanqieBookAnalysisDetail
	fetchFanqieBookDetail = func(context.Context, fanqieBook) (fanqieBookDetail, error) {
		t.Fatal("analysis should use the first-ten-chapter fetcher, not the full download fetcher")
		return fanqieBookDetail{}, nil
	}
	fetchFanqieBookAnalysisDetail = func(context.Context, fanqieBook) (fanqieBookDetail, error) {
		chapters := make([]fanqieChapter, 0, 10)
		for i := 1; i <= 10; i++ {
			chapters = append(chapters, fanqieChapter{
				Index:    i,
				ItemID:   fmt.Sprintf("c%d", i),
				Title:    fmt.Sprintf("第%d章 样章", i),
				Content:  fmt.Sprintf("第%d章可读正文片段", i),
				Readable: true,
			})
		}
		return fanqieBookDetail{
			Book: fanqieBook{
				ID:          "7143038691944959011",
				Title:       "十日终焉",
				Author:      "杀虫队队员",
				Category:    "悬疑脑洞",
				Status:      "已完结",
				WordCount:   "320万字",
				Description: "首页简介：死亡游戏与规则怪谈。",
				SourceURL:   "https://fanqienovel.com/page/7143038691944959011",
			},
			Chapters:        chapters,
			ReadableContent: true,
			Source:          "fixture",
		}, nil
	}
	t.Cleanup(func() {
		fetchFanqieBookDetail = oldDownloadFetch
		fetchFanqieBookAnalysisDetail = oldAnalysisFetch
	})

	rec := performStudioFanqieAnalyzeRequest(t, h, userID, `{
		"book":{"id":"7143038691944959011","title":"十日终焉","source_url":"https://fanqienovel.com/page/7143038691944959011"}
	}`)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "十日终焉开篇分析")
	requests := recorder.snapshot()
	require.Len(t, requests, 1)
	require.Contains(t, string(requests[0].Body), "首页简介")
	require.Contains(t, string(requests[0].Body), "第10章 样章")
	require.NotContains(t, string(requests[0].Body), "第11章 样章")
}

func TestExtractJSONObject(t *testing.T) {
	cases := map[string]string{
		`{"a":1}`:                      `{"a":1}`,
		"```json\n{\"a\":1}\n```":      `{"a":1}`,
		"前缀 {\"a\":1} 后缀":              `{"a":1}`,
		"```\n{\"x\": {\"y\":2}}\n```": `{"x": {"y":2}}`,
	}
	for in, want := range cases {
		if got := extractJSONObject(in); got != want {
			t.Fatalf("extractJSONObject(%q) = %q, want %q", in, got, want)
		}
	}
}
