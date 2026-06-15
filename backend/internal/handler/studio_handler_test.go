package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
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

type studioModelsHTTPUpstream struct {
	lastReq *http.Request
}

func (u *studioModelsHTTPUpstream) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	u.lastReq = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-5.4"},{"id":"gpt-image-2"}]}`)),
	}, nil
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
