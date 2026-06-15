package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
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
	require.Equal(t, "use this composition", fields["prompt"])
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
