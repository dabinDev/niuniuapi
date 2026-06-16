package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/creationtask"
	"github.com/Wei-Shaw/sub2api/ent/studiomodelconfig"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// recordCreation 把一次创作产出落库（失败不影响主流程）。
func (h *StudioHandler) recordCreation(ctx context.Context, userID int64, typ, title string, input, output any, model string) {
	inB, _ := json.Marshal(input)
	outB, _ := json.Marshal(output)
	_ = h.client.CreationTask.Create().
		SetUserID(userID).
		SetType(typ).
		SetTitle(title).
		SetInput(string(inB)).
		SetOutput(string(outB)).
		SetModel(model).
		Exec(ctx)
}

func titleOr(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	if len([]rune(s)) > 60 {
		return string([]rune(s)[:60])
	}
	return s
}

type workItem struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
}

// ListWorks 列出当前用户的创作产出（仅元数据，不含大体积 output）。
func (h *StudioHandler) ListWorks(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	q := h.client.CreationTask.Query().Where(creationtask.UserID(subject.UserID))
	if t := c.Query("type"); t != "" {
		q = q.Where(creationtask.TypeEQ(t))
	}
	rows, err := q.Order(dbent.Desc(creationtask.FieldCreatedAt)).Limit(100).All(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]workItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, workItem{ID: r.ID, Type: r.Type, Title: r.Title, Model: r.Model, CreatedAt: r.CreatedAt})
	}
	response.Success(c, gin.H{"works": items})
}

// GetWork 返回单条创作产出（含完整 output）。
func (h *StudioHandler) GetWork(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	r, err := h.client.CreationTask.Get(c.Request.Context(), id)
	if err != nil || r.UserID != subject.UserID {
		response.BadRequest(c, "作品不存在")
		return
	}
	out := gin.H{"id": r.ID, "type": r.Type, "title": r.Title, "model": r.Model, "created_at": r.CreatedAt}
	if r.Output != "" {
		out["output"] = json.RawMessage(r.Output)
	}
	if r.Input != "" {
		out["input"] = json.RawMessage(r.Input)
	}
	response.Success(c, out)
}

// StudioHandler 处理创作台：模型配置（生图 / 文案模型）与封面生成。
type StudioHandler struct {
	client             *dbent.Client
	accountRepo        service.AccountRepository
	accountTestService *service.AccountTestService
}

// NewStudioHandler 构造 StudioHandler。
func NewStudioHandler(client *dbent.Client) *StudioHandler {
	return &StudioHandler{client: client}
}

// NewStudioHandlerWithDeps constructs a StudioHandler with model discovery dependencies.
func NewStudioHandlerWithDeps(client *dbent.Client, accountRepo service.AccountRepository, accountTestService *service.AccountTestService) *StudioHandler {
	return &StudioHandler{
		client:             client,
		accountRepo:        accountRepo,
		accountTestService: accountTestService,
	}
}

type studioModelSlot struct {
	APIKeyID int64  `json:"api_key_id"`
	Model    string `json:"model"`
}

type studioModelConfigDTO struct {
	Image *studioModelSlot `json:"image,omitempty"`
	Text  *studioModelSlot `json:"text,omitempty"`
}

type studioModelTestRequest struct {
	Type     string `json:"type"`
	APIKeyID int64  `json:"api_key_id"`
	Model    string `json:"model"`
}

// GetModelConfig 读取当前用户的创作模型配置（无则返回 null）。
func (h *StudioHandler) GetModelConfig(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	row, err := h.client.StudioModelConfig.Query().
		Where(studiomodelconfig.UserID(subject.UserID)).
		Only(c.Request.Context())
	if err != nil {
		if dbent.IsNotFound(err) {
			response.Success(c, nil)
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	var dto studioModelConfigDTO
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &dto)
	}
	response.Success(c, dto)
}

// SaveModelConfig 保存当前用户的创作模型配置（upsert）。
func (h *StudioHandler) SaveModelConfig(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req studioModelConfigDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	raw, err := json.Marshal(req)
	if err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.client.StudioModelConfig.Create().
		SetUserID(subject.UserID).
		SetConfig(string(raw)).
		OnConflictColumns(studiomodelconfig.FieldUserID).
		UpdateNewValues().
		Exec(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, req)
}

// ListKeyModels reads live model IDs from the upstream account(s) behind a user API key.
func (h *StudioHandler) ListKeyModels(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || keyID <= 0 {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	if h.accountRepo == nil || h.accountTestService == nil {
		response.InternalError(c, "Studio model discovery is not configured")
		return
	}

	ak, err := h.client.APIKey.Get(c.Request.Context(), keyID)
	if err != nil || ak.UserID != subject.UserID {
		response.NotFound(c, "API key not found")
		return
	}
	if ak.GroupID == nil || *ak.GroupID <= 0 {
		response.BadRequest(c, "API key is not bound to a model group")
		return
	}

	g, err := h.client.Group.Get(c.Request.Context(), *ak.GroupID)
	if err != nil {
		if dbent.IsNotFound(err) {
			response.BadRequest(c, "API key model group is not available")
			return
		}
		response.ErrorFrom(c, err)
		return
	}

	accounts, err := h.accountRepo.ListSchedulableByGroupIDAndPlatform(c.Request.Context(), *ak.GroupID, g.Platform)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if len(accounts) == 0 {
		response.BadRequest(c, "No schedulable upstream accounts are available for this API key")
		return
	}

	modelSet := make(map[string]struct{})
	var lastErr error
	for i := range accounts {
		models, err := h.accountTestService.FetchUpstreamSupportedModels(c.Request.Context(), &accounts[i])
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
			model = strings.TrimSpace(model)
			if model != "" {
				modelSet[model] = struct{}{}
			}
		}
	}

	if len(modelSet) == 0 {
		writeStudioModelDiscoveryError(c, lastErr)
		return
	}

	out := make([]string, 0, len(modelSet))
	for model := range modelSet {
		out = append(out, model)
	}
	sort.Strings(out)
	response.Success(c, out)
}

func writeStudioModelDiscoveryError(c *gin.Context, err error) {
	if err == nil {
		response.BadRequest(c, "No upstream models are available for this API key")
		return
	}
	var syncErr *service.UpstreamModelSyncError
	if errors.As(err, &syncErr) {
		switch syncErr.Kind {
		case service.UpstreamModelSyncErrorConfiguration, service.UpstreamModelSyncErrorUnsupported:
			response.BadRequest(c, syncErr.SafeMessage())
		default:
			response.Error(c, http.StatusBadGateway, syncErr.SafeMessage())
		}
		return
	}
	response.Error(c, http.StatusBadGateway, "Failed to fetch upstream models")
}

// TestModel probes the selected upstream model through backend-owned account
// testing, instead of asking the browser to call the user API gateway directly.
func (h *StudioHandler) TestModel(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.accountRepo == nil || h.accountTestService == nil {
		response.InternalError(c, "Studio model test is not configured")
		return
	}

	var req studioModelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	req.Model = strings.TrimSpace(req.Model)
	if req.Type != "image" && req.Type != "text" {
		response.BadRequest(c, "Invalid model test type")
		return
	}
	if req.APIKeyID <= 0 || req.Model == "" {
		response.BadRequest(c, "API key and model are required")
		return
	}

	ak, err := h.client.APIKey.Get(c.Request.Context(), req.APIKeyID)
	if err != nil || ak.UserID != subject.UserID {
		response.NotFound(c, "API key not found")
		return
	}
	if ak.GroupID == nil || *ak.GroupID <= 0 {
		response.BadRequest(c, "API key is not bound to a model group")
		return
	}

	g, err := h.client.Group.Get(c.Request.Context(), *ak.GroupID)
	if err != nil {
		if dbent.IsNotFound(err) {
			response.BadRequest(c, "API key model group is not available")
			return
		}
		response.ErrorFrom(c, err)
		return
	}

	accounts, err := h.accountRepo.ListSchedulableByGroupIDAndPlatform(c.Request.Context(), *ak.GroupID, g.Platform)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if len(accounts) == 0 {
		response.BadRequest(c, "No schedulable upstream accounts are available for this API key")
		return
	}

	var lastErr string
	for i := range accounts {
		result, err := h.accountTestService.RunTestBackground(c.Request.Context(), accounts[i].ID, req.Model)
		if err != nil {
			lastErr = err.Error()
			continue
		}
		if result != nil && result.Status == "success" {
			response.Success(c, gin.H{"ok": true})
			return
		}
		if result != nil && strings.TrimSpace(result.ErrorMessage) != "" {
			lastErr = result.ErrorMessage
		}
	}

	if lastErr == "" {
		lastErr = "No upstream account passed the model test"
	}
	response.Error(c, http.StatusBadGateway, lastErr)
}

type coverRequest struct {
	Mode        string `json:"mode"`
	Prompt      string `json:"prompt"`
	RefImage    string `json:"ref_image"`
	Title       string `json:"title"`
	Synopsis    string `json:"synopsis"`
	Protagonist string `json:"protagonist"`
	Genre       string `json:"genre"`
	Mood        string `json:"mood"`
	KeyScene    string `json:"key_scene"`
	CoverTitle  string `json:"cover_title"`
	Size        string `json:"size"`
	Count       int    `json:"count"`
}

const studioCoverJobTimeout = 15 * time.Minute

type studioCoverJobStatus string

const (
	studioCoverJobRunning   studioCoverJobStatus = "running"
	studioCoverJobSucceeded studioCoverJobStatus = "succeeded"
	studioCoverJobFailed    studioCoverJobStatus = "failed"
)

type studioCoverJob struct {
	ID        string
	UserID    int64
	Status    studioCoverJobStatus
	Progress  int
	Message   string
	Error     string
	Result    any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type studioCoverJobResponse struct {
	JobID     string               `json:"job_id"`
	Status    studioCoverJobStatus `json:"status"`
	Progress  int                  `json:"progress"`
	Message   string               `json:"message,omitempty"`
	Error     string               `json:"error,omitempty"`
	Result    any                  `json:"result,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
}

var studioCoverJobs = struct {
	sync.RWMutex
	jobs map[string]*studioCoverJob
}{jobs: map[string]*studioCoverJob{}}

type studioCoverGenerationPlan struct {
	Request coverRequest
	Model   string
	APIKey  string
	Prompt  string
	Size    string
	Count   int
}

type studioCoverHTTPError struct {
	StatusCode int
	Message    string
	Body       []byte
}

func (e *studioCoverHTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if len(e.Body) > 0 {
		return string(e.Body)
	}
	return http.StatusText(e.StatusCode)
}

func newStudioCoverBadRequest(message string) error {
	return &studioCoverHTTPError{StatusCode: http.StatusBadRequest, Message: message}
}

func newStudioCoverJobID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "cover_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("cover_%d", time.Now().UnixNano())
}

func snapshotStudioCoverJob(job *studioCoverJob) studioCoverJobResponse {
	return studioCoverJobResponse{
		JobID:     job.ID,
		Status:    job.Status,
		Progress:  job.Progress,
		Message:   job.Message,
		Error:     job.Error,
		Result:    job.Result,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}
}

func storeStudioCoverJob(job *studioCoverJob) studioCoverJobResponse {
	studioCoverJobs.Lock()
	defer studioCoverJobs.Unlock()
	studioCoverJobs.jobs[job.ID] = job
	return snapshotStudioCoverJob(job)
}

func updateStudioCoverJob(id string, update func(*studioCoverJob)) (studioCoverJobResponse, bool) {
	studioCoverJobs.Lock()
	defer studioCoverJobs.Unlock()
	job, ok := studioCoverJobs.jobs[id]
	if !ok {
		return studioCoverJobResponse{}, false
	}
	update(job)
	job.UpdatedAt = time.Now()
	return snapshotStudioCoverJob(job), true
}

func getStudioCoverJobSnapshot(id string) (studioCoverJobResponse, int64, bool) {
	studioCoverJobs.RLock()
	defer studioCoverJobs.RUnlock()
	job, ok := studioCoverJobs.jobs[id]
	if !ok {
		return studioCoverJobResponse{}, 0, false
	}
	return snapshotStudioCoverJob(job), job.UserID, true
}

func studioCoverErrorMessage(err error) string {
	var httpErr *studioCoverHTTPError
	if errors.As(err, &httpErr) {
		if httpErr.Message != "" {
			return httpErr.Message
		}
		var parsed struct {
			Message string `json:"message"`
			Detail  string `json:"detail"`
			Error   any    `json:"error"`
		}
		if len(httpErr.Body) > 0 && json.Unmarshal(httpErr.Body, &parsed) == nil {
			if parsed.Message != "" {
				return parsed.Message
			}
			if parsed.Detail != "" {
				return parsed.Detail
			}
			switch v := parsed.Error.(type) {
			case string:
				return v
			case map[string]any:
				if msg, _ := v["message"].(string); msg != "" {
					return msg
				}
			}
		}
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func writeStudioCoverGenerationError(c *gin.Context, err error) {
	var httpErr *studioCoverHTTPError
	if errors.As(err, &httpErr) {
		statusCode := httpErr.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		if len(httpErr.Body) > 0 && httpErr.Message == "" {
			c.Data(statusCode, "application/json; charset=utf-8", httpErr.Body)
			return
		}
		response.Error(c, statusCode, studioCoverErrorMessage(err))
		return
	}
	response.ErrorFrom(c, err)
}

func buildCoverPrompt(req coverRequest) string {
	if req.Mode == "custom" {
		return strings.TrimSpace(req.Prompt)
	}
	var b strings.Builder
	b.WriteString("为一本小说设计一张精美的竖版书籍封面插画。")
	if req.Genre != "" {
		b.WriteString("题材与画风：" + req.Genre + "。")
	}
	if req.Mood != "" {
		b.WriteString("情绪基调：" + req.Mood + "。")
	}
	if req.Protagonist != "" {
		b.WriteString("主角形象：" + req.Protagonist + "。")
	}
	if req.KeyScene != "" {
		b.WriteString("关键场景与意象：" + req.KeyScene + "。")
	}
	if req.Synopsis != "" {
		b.WriteString("故事简介：" + req.Synopsis + "。")
	}
	b.WriteString("竖版书封构图，画面精致、细节丰富、有氛围感，画面中不要出现任何文字。")
	return b.String()
}

const studioCoverMaxReferenceImageBytes = 20 << 20

func isStudioGPTImageModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gpt-image")
}

func normalizeStudioCoverCount(count int) int {
	if count < 1 {
		return 1
	}
	if count > 4 {
		return 4
	}
	return count
}

func buildStudioCoverGatewayPayload(model, prompt, size string, count int, refImage string) (string, []byte, string, error) {
	refImage = strings.TrimSpace(refImage)
	if refImage == "" {
		payload, err := json.Marshal(map[string]any{
			"model":  model,
			"prompt": prompt,
			"n":      count,
			"size":   size,
		})
		return "/v1/images/generations", payload, "application/json", err
	}

	imageBytes, mimeType, fileName, err := decodeStudioCoverReferenceImage(refImage)
	if err != nil {
		return "", nil, "", err
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("model", model)
	_ = writer.WriteField("prompt", prompt)
	_ = writer.WriteField("n", strconv.Itoa(count))
	_ = writer.WriteField("size", size)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="image"; filename="`+fileName+`"`)
	header.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(header)
	if err != nil {
		_ = writer.Close()
		return "", nil, "", err
	}
	if _, err := part.Write(imageBytes); err != nil {
		_ = writer.Close()
		return "", nil, "", err
	}
	if err := writer.Close(); err != nil {
		return "", nil, "", err
	}
	return "/v1/images/edits", buf.Bytes(), writer.FormDataContentType(), nil
}

func decodeStudioCoverReferenceImage(raw string) ([]byte, string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", "", fmt.Errorf("参考图为空")
	}
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return nil, "", "", fmt.Errorf("参考图必须是上传后的图片数据")
	}
	header, encoded, ok := strings.Cut(raw, ",")
	if !ok || strings.TrimSpace(encoded) == "" {
		return nil, "", "", fmt.Errorf("参考图格式无效")
	}
	if !strings.Contains(strings.ToLower(header), ";base64") {
		return nil, "", "", fmt.Errorf("参考图必须使用 base64 数据")
	}

	mimeType := "image/png"
	mediaType := strings.TrimPrefix(header, "data:")
	if idx := strings.Index(mediaType, ";"); idx >= 0 {
		mediaType = mediaType[:idx]
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if strings.HasPrefix(mediaType, "image/") {
		mimeType = mediaType
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", "", fmt.Errorf("参考图必须是图片文件")
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "", "", fmt.Errorf("参考图解析失败")
	}
	if len(decoded) == 0 {
		return nil, "", "", fmt.Errorf("参考图为空")
	}
	if len(decoded) > studioCoverMaxReferenceImageBytes {
		return nil, "", "", fmt.Errorf("参考图不能超过 20MB")
	}

	fileName := "reference.png"
	switch mimeType {
	case "image/jpeg", "image/jpg":
		fileName = "reference.jpg"
	case "image/webp":
		fileName = "reference.webp"
	case "image/png":
		fileName = "reference.png"
	}
	return decoded, mimeType, fileName, nil
}

func (h *StudioHandler) prepareStudioCoverGeneration(ctx context.Context, userID int64, req coverRequest) (*studioCoverGenerationPlan, error) {
	row, err := h.client.StudioModelConfig.Query().
		Where(studiomodelconfig.UserID(userID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, newStudioCoverBadRequest("请先在「API 密钥」页配置生图模型")
		}
		return nil, err
	}
	var cfg studioModelConfigDTO
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &cfg)
	}
	if cfg.Image == nil || cfg.Image.Model == "" {
		return nil, newStudioCoverBadRequest("请先在「API 密钥」页配置生图模型")
	}

	ak, err := h.client.APIKey.Get(ctx, cfg.Image.APIKeyID)
	if err != nil || ak.UserID != userID {
		return nil, newStudioCoverBadRequest("生图密钥不可用，请到「API 密钥」页重新配置")
	}

	prompt := buildCoverPrompt(req)
	if prompt == "" {
		return nil, newStudioCoverBadRequest("请填写提示词或小说信息")
	}
	if strings.TrimSpace(req.RefImage) != "" {
		if _, _, _, err := decodeStudioCoverReferenceImage(req.RefImage); err != nil {
			return nil, newStudioCoverBadRequest(err.Error())
		}
	}

	count := normalizeStudioCoverCount(req.Count)
	size := req.Size
	if size == "" {
		size = "1024x1536"
	}

	return &studioCoverGenerationPlan{
		Request: req,
		Model:   cfg.Image.Model,
		APIKey:  ak.Key,
		Prompt:  prompt,
		Size:    size,
		Count:   count,
	}, nil
}

func (h *StudioHandler) generateStudioCoverResult(ctx context.Context, userID int64, plan *studioCoverGenerationPlan, progress func(int, string)) (gin.H, error) {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	gatewayBaseURL := "http://127.0.0.1:" + port
	httpClient := &http.Client{Timeout: studioCoverJobTimeout}

	jobCount := 1
	jobImageCount := plan.Count
	if isStudioGPTImageModel(plan.Model) && plan.Count > 1 {
		jobCount = plan.Count
		jobImageCount = 1
	}

	covers := make([]gin.H, 0, plan.Count)
	for jobIndex := 0; jobIndex < jobCount; jobIndex++ {
		if progress != nil {
			progress(10+(jobIndex*80)/jobCount, "正在等待上游生成图片")
		}
		endpoint, payload, contentType, err := buildStudioCoverGatewayPayload(plan.Model, plan.Prompt, plan.Size, jobImageCount, plan.Request.RefImage)
		if err != nil {
			return nil, newStudioCoverBadRequest(err.Error())
		}
		hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayBaseURL+endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		hreq.Header.Set("Content-Type", contentType)
		hreq.Header.Set("Authorization", "Bearer "+plan.APIKey)

		hresp, err := httpClient.Do(hreq)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(hresp.Body)
		_ = hresp.Body.Close()
		if hresp.StatusCode != http.StatusOK {
			return nil, &studioCoverHTTPError{StatusCode: hresp.StatusCode, Body: body}
		}

		var gres struct {
			Data []struct {
				URL string `json:"url"`
				B64 string `json:"b64_json"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &gres)
		for _, d := range gres.Data {
			u := d.URL
			if u == "" && d.B64 != "" {
				u = "data:image/png;base64," + d.B64
			}
			if u == "" {
				continue
			}
			covers = append(covers, gin.H{"id": fmt.Sprintf("cover-%d", len(covers)), "url": u})
		}
		if progress != nil {
			progress(10+((jobIndex+1)*80)/jobCount, "正在整理生成结果")
		}
	}
	result := gin.H{"covers": covers}
	ctitle := plan.Request.CoverTitle
	if ctitle == "" {
		ctitle = plan.Request.Title
	}
	h.recordCreation(ctx, userID, "cover", titleOr(ctitle, "封面"), plan.Request, result, plan.Model)
	return result, nil
}

func (h *StudioHandler) runStudioCoverJob(jobID string, userID int64, plan *studioCoverGenerationPlan) {
	ctx, cancel := context.WithTimeout(context.Background(), studioCoverJobTimeout)
	defer cancel()

	_, _ = updateStudioCoverJob(jobID, func(job *studioCoverJob) {
		job.Status = studioCoverJobRunning
		job.Progress = 10
		job.Message = "正在提交生图任务"
	})

	result, err := h.generateStudioCoverResult(ctx, userID, plan, func(progress int, message string) {
		_, _ = updateStudioCoverJob(jobID, func(job *studioCoverJob) {
			job.Status = studioCoverJobRunning
			job.Progress = progress
			job.Message = message
		})
	})
	if err != nil {
		_, _ = updateStudioCoverJob(jobID, func(job *studioCoverJob) {
			job.Status = studioCoverJobFailed
			job.Progress = 100
			job.Message = "生成失败"
			job.Error = studioCoverErrorMessage(err)
		})
		return
	}

	_, _ = updateStudioCoverJob(jobID, func(job *studioCoverJob) {
		job.Status = studioCoverJobSucceeded
		job.Progress = 100
		job.Message = "生成完成"
		job.Result = result
	})
}

func (h *StudioHandler) StartCoverJob(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req coverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	plan, err := h.prepareStudioCoverGeneration(c.Request.Context(), subject.UserID, req)
	if err != nil {
		writeStudioCoverGenerationError(c, err)
		return
	}

	now := time.Now()
	job := &studioCoverJob{
		ID:        newStudioCoverJobID(),
		UserID:    subject.UserID,
		Status:    studioCoverJobRunning,
		Progress:  5,
		Message:   "任务已提交，正在排队生成",
		CreatedAt: now,
		UpdatedAt: now,
	}
	snapshot := storeStudioCoverJob(job)
	go h.runStudioCoverJob(job.ID, subject.UserID, plan)

	response.Accepted(c, snapshot)
}

func (h *StudioHandler) GetCoverJob(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	jobID := strings.TrimSpace(c.Param("id"))
	if jobID == "" {
		response.BadRequest(c, "invalid job id")
		return
	}
	snapshot, userID, ok := getStudioCoverJobSnapshot(jobID)
	if !ok {
		response.NotFound(c, "任务不存在")
		return
	}
	if userID != subject.UserID {
		response.Forbidden(c, "无权访问该任务")
		return
	}
	response.Success(c, snapshot)
}

// GenerateCover 用已保存的生图模型 + 对应密钥，经网关生成封面候选图。
func (h *StudioHandler) GenerateCover(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	row, err := h.client.StudioModelConfig.Query().
		Where(studiomodelconfig.UserID(subject.UserID)).
		Only(c.Request.Context())
	if err != nil {
		if dbent.IsNotFound(err) {
			response.BadRequest(c, "请先在「API 密钥」页配置生图模型")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	var cfg studioModelConfigDTO
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &cfg)
	}
	if cfg.Image == nil || cfg.Image.Model == "" {
		response.BadRequest(c, "请先在「API 密钥」页配置生图模型")
		return
	}

	ak, err := h.client.APIKey.Get(c.Request.Context(), cfg.Image.APIKeyID)
	if err != nil || ak.UserID != subject.UserID {
		response.BadRequest(c, "生图密钥不可用，请到「API 密钥」页重新配置")
		return
	}

	var req coverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	prompt := buildCoverPrompt(req)
	if prompt == "" {
		response.BadRequest(c, "请填写提示词或小说信息")
		return
	}
	count := normalizeStudioCoverCount(req.Count)
	size := req.Size
	if size == "" {
		size = "1024x1536"
	}

	// 自调网关需指向容器内部监听端口（SERVER_PORT，默认 8080），
	// 不能用 c.Request.Host（那是外部映射端口，容器内不可达）。
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	gatewayBaseURL := "http://127.0.0.1:" + port
	httpClient := &http.Client{Timeout: 180 * time.Second}

	jobCount := 1
	jobImageCount := count
	if isStudioGPTImageModel(cfg.Image.Model) && count > 1 {
		jobCount = count
		jobImageCount = 1
	}

	covers := make([]gin.H, 0, count)
	for jobIndex := 0; jobIndex < jobCount; jobIndex++ {
		endpoint, payload, contentType, err := buildStudioCoverGatewayPayload(cfg.Image.Model, prompt, size, jobImageCount, req.RefImage)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		hreq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, gatewayBaseURL+endpoint, bytes.NewReader(payload))
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		hreq.Header.Set("Content-Type", contentType)
		hreq.Header.Set("Authorization", "Bearer "+ak.Key)

		hresp, err := httpClient.Do(hreq)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		body, _ := io.ReadAll(hresp.Body)
		_ = hresp.Body.Close()
		if hresp.StatusCode != http.StatusOK {
			// 透传上游错误（含状态码），让前端按错误提示
			c.Data(hresp.StatusCode, "application/json; charset=utf-8", body)
			return
		}

		var gres struct {
			Data []struct {
				URL string `json:"url"`
				B64 string `json:"b64_json"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &gres)
		for _, d := range gres.Data {
			u := d.URL
			if u == "" && d.B64 != "" {
				u = "data:image/png;base64," + d.B64
			}
			if u == "" {
				continue
			}
			covers = append(covers, gin.H{"id": fmt.Sprintf("cover-%d", len(covers)), "url": u})
		}
	}
	result := gin.H{"covers": covers}
	ctitle := req.CoverTitle
	if ctitle == "" {
		ctitle = req.Title
	}
	h.recordCreation(c.Request.Context(), subject.UserID, "cover", titleOr(ctitle, "封面"), req, result, cfg.Image.Model)
	response.Success(c, result)
}

type teardownRequest struct {
	Content string `json:"content"`
	Title   string `json:"title"`
	Genre   string `json:"genre"`
	Tone    string `json:"tone"`
}

type teardownScoreItem struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type teardownReport struct {
	OverallScore int                 `json:"overall_score"`
	Verdict      string              `json:"verdict"`
	Summary      string              `json:"summary"`
	Scores       []teardownScoreItem `json:"scores"`
	Highlights   []string            `json:"highlights"`
	RottenPoints []string            `json:"rotten_points"`
	Suggestions  []string            `json:"suggestions"`
}

func teardownToneHint(tone string) string {
	switch tone {
	case "gentle":
		return "点评以鼓励为主、温和提点，但仍要点出真实问题"
	case "neutral":
		return "点评保持中性客观"
	default:
		return "点评要毒舌犀利、直接戳痛点、不留情面，但每条都要有理有据"
	}
}

func buildTeardownPrompt(req teardownRequest) (string, string) {
	system := "你是「烂番茄」的毒舌创作质检官，擅长把小说拆成结构化诊断并对照爆款标准评分。只输出一个 JSON 对象，不要任何解释，不要 markdown 代码块。"
	var b strings.Builder
	b.WriteString("请拆解并诊断下面的小说内容")
	if req.Title != "" {
		b.WriteString("（书名：" + req.Title + "）")
	}
	if req.Genre != "" {
		b.WriteString("（题材：" + req.Genre + "）")
	}
	b.WriteString("。" + teardownToneHint(req.Tone) + "。\n")
	b.WriteString("严格按以下 JSON 结构输出，所有 value 为 0-100 的整数：\n")
	b.WriteString(`{"overall_score":0,"verdict":"一句话毒舌诊断","summary":"2-4句结构概览","scores":[{"label":"开篇钩子","value":0},{"label":"节奏","value":0},{"label":"爽点密度","value":0},{"label":"人物","value":0},{"label":"套路新鲜度","value":0}],"highlights":["亮点"],"rotten_points":["烂点/烂梗/烂节奏"],"suggestions":["可直接动刀的改进建议"]}`)
	b.WriteString("\n\n小说内容：\n")
	b.WriteString(req.Content)
	return system, b.String()
}

// extractJSONObject 从模型输出里提取 JSON 对象（去掉 markdown 代码块、取首尾大括号）。
func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

// GenerateTeardown 用已保存的文案模型，对小说做毒舌拆书 & 爆款分析。
func (h *StudioHandler) GenerateTeardown(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	row, err := h.client.StudioModelConfig.Query().
		Where(studiomodelconfig.UserID(subject.UserID)).
		Only(c.Request.Context())
	if err != nil {
		if dbent.IsNotFound(err) {
			response.BadRequest(c, "请先在「API 密钥」页配置文案模型")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	var cfg studioModelConfigDTO
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &cfg)
	}
	if cfg.Text == nil || cfg.Text.Model == "" {
		response.BadRequest(c, "请先在「API 密钥」页配置文案模型")
		return
	}
	ak, err := h.client.APIKey.Get(c.Request.Context(), cfg.Text.APIKeyID)
	if err != nil || ak.UserID != subject.UserID {
		response.BadRequest(c, "文案密钥不可用，请到「API 密钥」页重新配置")
		return
	}

	var req teardownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		response.BadRequest(c, "请填写要拆解的小说内容")
		return
	}

	system, user := buildTeardownPrompt(req)
	payload, _ := json.Marshal(map[string]any{
		"model": cfg.Text.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature":     0.7,
		"response_format": map[string]string{"type": "json_object"},
	})

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	gwURL := "http://127.0.0.1:" + port + "/v1/chat/completions"
	hreq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, gwURL, bytes.NewReader(payload))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Authorization", "Bearer "+ak.Key)

	httpClient := &http.Client{Timeout: 180 * time.Second}
	hresp, err := httpClient.Do(hreq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer hresp.Body.Close()
	body, _ := io.ReadAll(hresp.Body)
	if hresp.StatusCode != http.StatusOK {
		c.Data(hresp.StatusCode, "application/json; charset=utf-8", body)
		return
	}

	var chat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chat); err != nil || len(chat.Choices) == 0 {
		response.ErrorFrom(c, fmt.Errorf("拆书失败：上游未返回有效结果"))
		return
	}
	var report teardownReport
	if err := json.Unmarshal([]byte(extractJSONObject(chat.Choices[0].Message.Content)), &report); err != nil {
		response.ErrorFrom(c, fmt.Errorf("解析模型输出失败：%w", err))
		return
	}
	if report.Scores == nil {
		report.Scores = []teardownScoreItem{}
	}
	if report.Highlights == nil {
		report.Highlights = []string{}
	}
	if report.RottenPoints == nil {
		report.RottenPoints = []string{}
	}
	if report.Suggestions == nil {
		report.Suggestions = []string{}
	}
	h.recordCreation(c.Request.Context(), subject.UserID, "teardown", titleOr(req.Title, "拆书报告"), req, report, cfg.Text.Model)
	response.Success(c, report)
}

type scriptRequest struct {
	Content  string `json:"content"`
	Form     string `json:"form"`
	Episodes int    `json:"episodes"`
}

type scriptScene struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

type scriptResult struct {
	Title  string        `json:"title"`
	Form   string        `json:"form"`
	Scenes []scriptScene `json:"scenes"`
}

func scriptFormHint(form string) string {
	switch form {
	case "long":
		return "完整长剧本：详细的场景描述、人物对白、动作提示"
	case "storyboard":
		return "分镜脚本：逐镜头给出景别、画面内容、旁白/对白与时长提示"
	default:
		return "竖屏短剧脚本：节奏快、冲突强，每集结尾留钩子"
	}
}

func buildScriptPrompt(req scriptRequest) (string, string) {
	system := "你是「烂番茄」的剧本改编官，把小说改写成剧本/分镜。只输出一个 JSON 对象，不要任何解释，不要 markdown 代码块。"
	var b strings.Builder
	b.WriteString("把下面的小说内容改写成" + scriptFormHint(req.Form) + "。保留原作的人物与设定一致性。")
	if req.Episodes > 0 {
		b.WriteString("控制在约 " + itoa(req.Episodes) + " 集/段。")
	}
	b.WriteString("\n严格按以下 JSON 结构输出：\n")
	b.WriteString(`{"title":"剧本标题","scenes":[{"heading":"场景一 · 时间/地点","content":"场景描述 + 对白 + 动作 + 镜头提示"}]}`)
	b.WriteString("\n\n小说内容：\n")
	b.WriteString(req.Content)
	return system, b.String()
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// GenerateScript 用已保存的文案模型，把小说改写成剧本 / 分镜。
func (h *StudioHandler) GenerateScript(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	row, err := h.client.StudioModelConfig.Query().
		Where(studiomodelconfig.UserID(subject.UserID)).
		Only(c.Request.Context())
	if err != nil {
		if dbent.IsNotFound(err) {
			response.BadRequest(c, "请先在「API 密钥」页配置文案模型")
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	var cfg studioModelConfigDTO
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &cfg)
	}
	if cfg.Text == nil || cfg.Text.Model == "" {
		response.BadRequest(c, "请先在「API 密钥」页配置文案模型")
		return
	}
	ak, err := h.client.APIKey.Get(c.Request.Context(), cfg.Text.APIKeyID)
	if err != nil || ak.UserID != subject.UserID {
		response.BadRequest(c, "文案密钥不可用，请到「API 密钥」页重新配置")
		return
	}

	var req scriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		response.BadRequest(c, "请填写要改编的小说内容")
		return
	}

	system, user := buildScriptPrompt(req)
	payload, _ := json.Marshal(map[string]any{
		"model": cfg.Text.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature":     0.7,
		"response_format": map[string]string{"type": "json_object"},
	})

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	gwURL := "http://127.0.0.1:" + port + "/v1/chat/completions"
	hreq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, gwURL, bytes.NewReader(payload))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Authorization", "Bearer "+ak.Key)

	httpClient := &http.Client{Timeout: 180 * time.Second}
	hresp, err := httpClient.Do(hreq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer hresp.Body.Close()
	body, _ := io.ReadAll(hresp.Body)
	if hresp.StatusCode != http.StatusOK {
		c.Data(hresp.StatusCode, "application/json; charset=utf-8", body)
		return
	}

	var chat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chat); err != nil || len(chat.Choices) == 0 {
		response.ErrorFrom(c, fmt.Errorf("剧本生成失败：上游未返回有效结果"))
		return
	}
	var result scriptResult
	if err := json.Unmarshal([]byte(extractJSONObject(chat.Choices[0].Message.Content)), &result); err != nil {
		response.ErrorFrom(c, fmt.Errorf("解析模型输出失败：%w", err))
		return
	}
	result.Form = req.Form
	if result.Form == "" {
		result.Form = "short"
	}
	if result.Scenes == nil {
		result.Scenes = []scriptScene{}
	}
	h.recordCreation(c.Request.Context(), subject.UserID, "script", titleOr(result.Title, "剧本"), req, result, cfg.Text.Model)
	response.Success(c, result)
}

func (h *StudioHandler) requireStudioTextModel(c *gin.Context, userID int64) (*dbent.APIKey, string, bool) {
	row, err := h.client.StudioModelConfig.Query().
		Where(studiomodelconfig.UserID(userID)).
		Only(c.Request.Context())
	if err != nil {
		if dbent.IsNotFound(err) {
			response.BadRequest(c, "请先在「API 密钥」页配置文案模型")
			return nil, "", false
		}
		response.ErrorFrom(c, err)
		return nil, "", false
	}
	var cfg studioModelConfigDTO
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &cfg)
	}
	if cfg.Text == nil || cfg.Text.Model == "" {
		response.BadRequest(c, "请先在「API 密钥」页配置文案模型")
		return nil, "", false
	}
	ak, err := h.client.APIKey.Get(c.Request.Context(), cfg.Text.APIKeyID)
	if err != nil || ak.UserID != userID {
		response.BadRequest(c, "文案密钥不可用，请到「API 密钥」页重新配置")
		return nil, "", false
	}
	return ak, cfg.Text.Model, true
}

func studioGatewayChatURL() string {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	return "http://127.0.0.1:" + port + "/v1/chat/completions"
}

func postStudioChatCompletion(ctx context.Context, ak *dbent.APIKey, model, system, user string) ([]byte, int, error) {
	payload, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature":     0.72,
		"response_format": map[string]string{"type": "json_object"},
	})
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, studioGatewayChatURL(), bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	hreq.Header.Set("Authorization", "Bearer "+ak.Key)

	httpClient := &http.Client{Timeout: 180 * time.Second}
	hresp, err := httpClient.Do(hreq)
	if err != nil {
		return nil, 0, err
	}
	defer hresp.Body.Close()
	body, _ := io.ReadAll(hresp.Body)
	return body, hresp.StatusCode, nil
}

func extractStudioChatContent(body []byte) (string, error) {
	var chat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chat); err != nil || len(chat.Choices) == 0 {
		return "", fmt.Errorf("上游未返回有效结果")
	}
	return extractJSONObject(chat.Choices[0].Message.Content), nil
}

type hotspotRequest struct {
	Content   string `json:"content"`
	Benchmark string `json:"benchmark"`
	Title     string `json:"title"`
	Genre     string `json:"genre"`
	Goal      string `json:"goal"`
}

type hotspotSample struct {
	Title  string `json:"title"`
	Lesson string `json:"lesson"`
}

type hotspotReport struct {
	MarketScore int                 `json:"market_score"`
	Verdict     string              `json:"verdict"`
	Radar       []teardownScoreItem `json:"radar"`
	Tropes      []string            `json:"tropes"`
	Gaps        []string            `json:"gaps"`
	Actions     []string            `json:"actions"`
	Samples     []hotspotSample     `json:"samples"`
}

func hotspotGoalHint(goal string) string {
	switch goal {
	case "rewrite":
		return "目标是修订已有作品，请重点找可直接动刀的结构差距"
	case "short-video":
		return "目标是短视频/短剧改编，请重点找高钩子、高反差、强情绪价值桥段"
	default:
		return "目标是新书立项和开篇策略，请重点判断题材卖点、黄金三章和爽点兑现"
	}
}

func buildHotspotPrompt(req hotspotRequest) (string, string) {
	system := "你是「烂番茄」爆款对标分析师，负责把作者作品与对标样本拆成可执行策略。只输出一个 JSON 对象，不要解释，不要 markdown 代码块。"
	var b strings.Builder
	b.WriteString("请做爆款对标分析。")
	if req.Title != "" {
		b.WriteString("作品：" + req.Title + "。")
	}
	if req.Genre != "" {
		b.WriteString("题材：" + req.Genre + "。")
	}
	b.WriteString(hotspotGoalHint(req.Goal) + "。\n")
	b.WriteString("严格按以下 JSON 结构输出：\n")
	b.WriteString(`{"market_score":0,"verdict":"一句话判断","radar":[{"label":"三秒钩子","value":0},{"label":"爽点密度","value":0},{"label":"章节尾钩","value":0},{"label":"题材辨识度","value":0},{"label":"改编潜力","value":0}],"tropes":["可复用套路"],"gaps":["与爆款样本的差距"],"actions":["下一步可执行动作"],"samples":[{"title":"样本/参照","lesson":"可借鉴点"}]}`)
	b.WriteString("\n\n我的作品片段：\n")
	b.WriteString(req.Content)
	if strings.TrimSpace(req.Benchmark) != "" {
		b.WriteString("\n\n对标样本 / 爆款信息：\n")
		b.WriteString(req.Benchmark)
	}
	return system, b.String()
}

// GenerateHotspot 用文案模型生成爆款对标报告。
func (h *StudioHandler) GenerateHotspot(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	ak, model, ok := h.requireStudioTextModel(c, subject.UserID)
	if !ok {
		return
	}
	var req hotspotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		response.BadRequest(c, "请填写要对标的作品片段")
		return
	}
	if req.Goal == "" {
		req.Goal = "new-book"
	}

	system, user := buildHotspotPrompt(req)
	body, status, err := postStudioChatCompletion(c.Request.Context(), ak, model, system, user)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if status != http.StatusOK {
		c.Data(status, "application/json; charset=utf-8", body)
		return
	}
	content, err := extractStudioChatContent(body)
	if err != nil {
		response.ErrorFrom(c, fmt.Errorf("爆款对标失败：%w", err))
		return
	}
	var report hotspotReport
	if err := json.Unmarshal([]byte(content), &report); err != nil {
		response.ErrorFrom(c, fmt.Errorf("解析模型输出失败：%w", err))
		return
	}
	if report.Radar == nil {
		report.Radar = []teardownScoreItem{}
	}
	if report.Tropes == nil {
		report.Tropes = []string{}
	}
	if report.Gaps == nil {
		report.Gaps = []string{}
	}
	if report.Actions == nil {
		report.Actions = []string{}
	}
	if report.Samples == nil {
		report.Samples = []hotspotSample{}
	}
	h.recordCreation(c.Request.Context(), subject.UserID, "hotspot", titleOr(req.Title, "爆款对标"), req, report, model)
	response.Success(c, report)
}

type creativeRequest struct {
	Content     string `json:"content"`
	Mode        string `json:"mode"`
	Brief       string `json:"brief"`
	Genre       string `json:"genre"`
	Style       string `json:"style"`
	TargetWords int    `json:"target_words"`
	Episodes    int    `json:"episodes"`
}

type creativeSection struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

type creativeResult struct {
	Title     string            `json:"title"`
	Mode      string            `json:"mode"`
	Summary   string            `json:"summary"`
	Sections  []creativeSection `json:"sections"`
	Checklist []string          `json:"checklist"`
	NextSteps []string          `json:"next_steps"`
}

func creativeModeHint(mode string) string {
	switch mode {
	case "outline":
		return "大纲生成：输出卷纲、主线推进、阶段爽点和章节钩子"
	case "draft":
		return "正文续写：承接原文风格，输出可直接使用的章节正文"
	case "rewrite":
		return "改写增强：保留设定和剧情，增强冲突、节奏、爽点与人设选择"
	case "long":
		return "完整长剧本：详细场景、人物对白、动作提示"
	case "storyboard":
		return "分镜脚本：逐镜头给出景别、画面内容、旁白/对白与时长提示"
	default:
		return "竖屏短剧脚本：节奏快、冲突强，每集结尾留钩子"
	}
}

func buildCreativePrompt(req creativeRequest) (string, string) {
	system := "你是「烂番茄」创作生成台，负责把作者素材生成可继续写作、可修订、可改编的结构化内容。只输出一个 JSON 对象，不要解释，不要 markdown 代码块。"
	if req.Mode == "" {
		req.Mode = "draft"
	}
	var b strings.Builder
	b.WriteString("生成类型：" + creativeModeHint(req.Mode) + "。")
	if req.Genre != "" {
		b.WriteString("题材：" + req.Genre + "。")
	}
	if req.Style != "" {
		b.WriteString("风格要求：" + req.Style + "。")
	}
	if req.Brief != "" {
		b.WriteString("创作目标：" + req.Brief + "。")
	}
	if req.TargetWords > 0 {
		b.WriteString("目标字数约 " + itoa(req.TargetWords) + " 字。")
	}
	if req.Episodes > 0 {
		b.WriteString("控制在约 " + itoa(req.Episodes) + " 集/段。")
	}
	b.WriteString("\n严格按以下 JSON 结构输出：\n")
	b.WriteString(`{"title":"标题","summary":"本次生成意图概述","sections":[{"heading":"段落/章节/场景标题","content":"正文内容"}],"checklist":["一致性检查项"],"next_steps":["下一步建议"]}`)
	b.WriteString("\n\n素材内容：\n")
	b.WriteString(req.Content)
	return system, b.String()
}

// GenerateCreative 用文案模型生成大纲、正文续写、改写和剧本类内容。
func (h *StudioHandler) GenerateCreative(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	ak, model, ok := h.requireStudioTextModel(c, subject.UserID)
	if !ok {
		return
	}
	var req creativeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		response.BadRequest(c, "请填写创作素材")
		return
	}
	if req.Mode == "" {
		req.Mode = "draft"
	}

	system, user := buildCreativePrompt(req)
	body, status, err := postStudioChatCompletion(c.Request.Context(), ak, model, system, user)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if status != http.StatusOK {
		c.Data(status, "application/json; charset=utf-8", body)
		return
	}
	content, err := extractStudioChatContent(body)
	if err != nil {
		response.ErrorFrom(c, fmt.Errorf("创作生成失败：%w", err))
		return
	}
	var result creativeResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		response.ErrorFrom(c, fmt.Errorf("解析模型输出失败：%w", err))
		return
	}
	result.Mode = req.Mode
	if result.Sections == nil {
		result.Sections = []creativeSection{}
	}
	if result.Checklist == nil {
		result.Checklist = []string{}
	}
	if result.NextSteps == nil {
		result.NextSteps = []string{}
	}
	h.recordCreation(c.Request.Context(), subject.UserID, "generate", titleOr(result.Title, "创作生成"), req, result, model)
	response.Success(c, result)
}

type studioImportRequest struct {
	Source  string   `json:"source"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	URLs    []string `json:"urls"`
	Consent bool     `json:"consent"`
}

type studioImportChapter struct {
	Title     string `json:"title"`
	WordCount int    `json:"word_count"`
	Source    string `json:"source,omitempty"`
}

type studioImportResult struct {
	Title       string                `json:"title"`
	Source      string                `json:"source"`
	Status      string                `json:"status"`
	Chapters    []studioImportChapter `json:"chapters"`
	Notes       []string              `json:"notes"`
	NextActions []string              `json:"next_actions"`
}

func looksLikeChapterTitle(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	lower := strings.ToLower(line)
	return (strings.HasPrefix(line, "第") && strings.Contains(line, "章")) ||
		strings.HasPrefix(lower, "chapter ") ||
		strings.HasPrefix(lower, "chapter.")
}

func countCJKWords(s string) int {
	return len([]rune(strings.TrimSpace(s)))
}

func splitManualChapters(content string) []studioImportChapter {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	chapters := make([]studioImportChapter, 0)
	currentTitle := ""
	var body strings.Builder

	flush := func() {
		text := strings.TrimSpace(body.String())
		if currentTitle == "" && text == "" {
			return
		}
		title := currentTitle
		if title == "" {
			title = "全文导入"
		}
		chapters = append(chapters, studioImportChapter{Title: title, WordCount: countCJKWords(text)})
		body.Reset()
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if looksLikeChapterTitle(line) {
			flush()
			currentTitle = line
			continue
		}
		if line != "" {
			body.WriteString(line)
			body.WriteString("\n")
		}
	}
	flush()
	return chapters
}

func buildStudioImportResult(req studioImportRequest) studioImportResult {
	source := req.Source
	if source == "" {
		source = "manual"
	}
	title := titleOr(req.Title, "未命名作品")
	result := studioImportResult{
		Title:  title,
		Source: source,
		Status: "completed",
		Notes: []string{
			"仅保存你确认有权处理的内容，适合作为个人作品备份和后续创作上下文。",
		},
		NextActions: []string{"送去拆书诊断", "生成续写大纲", "沉淀人物与设定"},
	}
	if strings.TrimSpace(req.Content) != "" {
		result.Chapters = splitManualChapters(req.Content)
		result.Notes = append(result.Notes, "已按章节标题拆分；未识别标题时会作为全文导入。")
		return result
	}
	for i, u := range req.URLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		result.Chapters = append(result.Chapters, studioImportChapter{
			Title:  fmt.Sprintf("待导入链接 %d", i+1),
			Source: u,
		})
	}
	if len(result.Chapters) > 0 {
		result.Status = "queued"
		result.Notes = append(result.Notes, "已记录链接清单；自动抓取器接入后可按任务队列处理。")
	}
	return result
}

// ImportStudioContent 保存合规的个人作品导入结果，当前支持手动章节内容与链接清单记录。
func (h *StudioHandler) ImportStudioContent(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req studioImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !req.Consent {
		response.BadRequest(c, "请先确认仅导入你有权备份和处理的内容")
		return
	}
	if strings.TrimSpace(req.Content) == "" && len(req.URLs) == 0 {
		response.BadRequest(c, "请粘贴章节内容或填写链接清单")
		return
	}
	result := buildStudioImportResult(req)
	if len(result.Chapters) == 0 {
		response.BadRequest(c, "未识别到可导入内容")
		return
	}
	h.recordCreation(c.Request.Context(), subject.UserID, "import", titleOr(result.Title+"导入", "作品导入"), req, result, "local-importer")
	response.Success(c, result)
}
