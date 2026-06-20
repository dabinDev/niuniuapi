package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"regexp"
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

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
			dto, autoErr := h.autoConfigureStudioModelConfig(c.Request.Context(), subject.UserID, studioModelConfigDTO{})
			if autoErr != nil {
				slog.Warn("studio_model_auto_config_failed", "user_id", subject.UserID, "error", autoErr)
				response.Success(c, nil)
				return
			}
			if dto.Image == nil && dto.Text == nil {
				response.Success(c, nil)
				return
			}
			response.Success(c, dto)
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	var dto studioModelConfigDTO
	if row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &dto)
	}
	if dto.Image == nil || dto.Text == nil {
		if next, autoErr := h.autoConfigureStudioModelConfig(c.Request.Context(), subject.UserID, dto); autoErr == nil {
			dto = next
		} else {
			slog.Warn("studio_model_auto_config_failed", "user_id", subject.UserID, "error", autoErr)
		}
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

	models, err := h.collectStudioModelsFromAccounts(c.Request.Context(), g, accounts)
	if err != nil {
		writeStudioModelDiscoveryError(c, err)
		return
	}
	response.Success(c, models)
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

type coverPromptPolishResponse struct {
	Prompt      string `json:"prompt,omitempty"`
	Protagonist string `json:"protagonist,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Mood        string `json:"mood,omitempty"`
	KeyScene    string `json:"key_scene,omitempty"`
	CoverTitle  string `json:"cover_title,omitempty"`
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

func normalizeStudioCoverErrorMessage(message string) string {
	normalized := strings.TrimSpace(message)
	if normalized == "" {
		return ""
	}
	lower := strings.ToLower(normalized)
	switch {
	case strings.Contains(lower, "insufficient account balance") || strings.Contains(lower, "insufficient balance"):
		return "账户余额不足，请先充值或联系管理员增加余额"
	case strings.Contains(lower, "image generation is not enabled for this group"):
		return "当前 API 密钥所属分组未启用生图，请在管理员后台为该分组开启图片生成"
	case strings.Contains(lower, "upstream authentication failed") || strings.Contains(lower, "invalid api key"):
		return "上游生图账号鉴权失败，请在管理员后台检查 OpenAI 账号 API Key"
	case strings.Contains(lower, "no available accounts"):
		return "当前分组没有可用的上游生图账号，请在管理员后台检查账号状态"
	default:
		return normalized
	}
}

func studioCoverErrorMessage(err error) string {
	var httpErr *studioCoverHTTPError
	if errors.As(err, &httpErr) {
		if httpErr.Message != "" {
			return normalizeStudioCoverErrorMessage(httpErr.Message)
		}
		var parsed struct {
			Message string `json:"message"`
			Detail  string `json:"detail"`
			Error   any    `json:"error"`
		}
		if len(httpErr.Body) > 0 && json.Unmarshal(httpErr.Body, &parsed) == nil {
			if parsed.Message != "" {
				return normalizeStudioCoverErrorMessage(parsed.Message)
			}
			if parsed.Detail != "" {
				return normalizeStudioCoverErrorMessage(parsed.Detail)
			}
			switch v := parsed.Error.(type) {
			case string:
				return normalizeStudioCoverErrorMessage(v)
			case map[string]any:
				if msg, _ := v["message"].(string); msg != "" {
					return normalizeStudioCoverErrorMessage(msg)
				}
			}
		}
	}
	if err != nil {
		return normalizeStudioCoverErrorMessage(err.Error())
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
		raw := strings.TrimSpace(req.Prompt)
		if raw == "" {
			return ""
		}
		var b strings.Builder
		b.WriteString("为一本网络小说设计竖版小说封面。")
		b.WriteString("创意 brief：" + raw + "。")
		b.WriteString("这是小说封面，不是普通插画；构图需要适合移动端书城缩略图。")
		b.WriteString("封面文字/书名区域要明确醒目，有主标题层级和版式设计；文字应清晰、有设计感，避免乱码、错别字、无意义字符。")
		b.WriteString("如果无法稳定渲染准确中文，请预留干净醒目的书名标题区，不要生成随机文字。")
		return b.String()
	}
	var b strings.Builder
	b.WriteString("为一本网络小说设计一张精美的竖版小说封面。")
	if req.Title != "" {
		b.WriteString("书名：" + req.Title + "。")
	}
	if req.CoverTitle != "" {
		b.WriteString("封面主标题文字：" + req.CoverTitle + "。")
	}
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
	b.WriteString("这是小说封面，不是普通插画；竖版书封构图，画面精致、细节丰富、有氛围感，适合移动端书城缩略图。")
	b.WriteString("封面文字/书名区域要明确醒目，有主标题层级和版式设计；如提供封面标题或书名，优先作为主标题文字，文字应清晰、有设计感，避免乱码、错别字、无意义字符。")
	b.WriteString("如果模型无法稳定渲染准确中文，请预留干净醒目的书名标题区，不要生成随机文字。")
	return b.String()
}

func buildCoverPromptPolishPrompt(req coverRequest) (string, string) {
	system := "你是「烂番茄」小说封面提示词导演，负责把作者的粗略想法整理成可直接用于生图的封面 brief。只输出一个 JSON 对象，不要解释，不要 markdown 代码块。"
	var b strings.Builder
	if req.Mode == "custom" {
		b.WriteString("请完善下面的自定义封面提示词，输出字段：")
		b.WriteString(`{"prompt":"可直接用于生图的中文提示词"}`)
		b.WriteString("。要求：必须明确这是竖版网络小说封面，不是普通插画；保留原始创意，不改变题材；强化画面焦点、构图、氛围、光影、人物/场景细节；重点强调封面文字、书名区域、标题层级、移动端缩略图可读性；避免乱码、错别字、无意义字符；如果无法准确渲染中文，要求预留干净醒目的标题区。")
		b.WriteString("\n原始提示词：\n")
		b.WriteString(req.Prompt)
		return system, b.String()
	}

	b.WriteString("请根据小说信息补全封面生成表单，输出字段：")
	b.WriteString(`{"protagonist":"主角外貌/气质","genre":"题材/画风","mood":"情绪基调","key_scene":"关键场景/意象","cover_title":"封面标题文字"}`)
	b.WriteString("。要求：字段要短、具体、可直接进入生图 brief；所有内容都服务于竖版网络小说封面；cover_title 优先使用用户提供的封面标题或书名；如果没有书名，根据简介提炼一个适合封面的短标题；重点考虑封面文字/书名区域、标题层级和移动端点击承诺，避免引导模型生成乱码或随机文字。")
	if req.Title != "" {
		b.WriteString("\n书名：" + req.Title)
	}
	if req.CoverTitle != "" {
		b.WriteString("\n已有封面标题：" + req.CoverTitle)
	}
	if req.Genre != "" {
		b.WriteString("\n已有题材/画风：" + req.Genre)
	}
	if req.Protagonist != "" {
		b.WriteString("\n已有主角：" + req.Protagonist)
	}
	if req.Mood != "" {
		b.WriteString("\n已有情绪基调：" + req.Mood)
	}
	if req.KeyScene != "" {
		b.WriteString("\n已有关键场景/意象：" + req.KeyScene)
	}
	b.WriteString("\n简介：\n")
	b.WriteString(req.Synopsis)
	return system, b.String()
}

func trimCoverPromptPolishResponse(mode string, req coverRequest, result *coverPromptPolishResponse) {
	result.Prompt = strings.TrimSpace(result.Prompt)
	result.Protagonist = firstNonEmptyString(result.Protagonist, req.Protagonist)
	result.Genre = firstNonEmptyString(result.Genre, req.Genre)
	result.Mood = firstNonEmptyString(result.Mood, req.Mood)
	result.KeyScene = firstNonEmptyString(result.KeyScene, req.KeyScene)
	result.CoverTitle = firstNonEmptyString(result.CoverTitle, req.CoverTitle, req.Title)
	if mode == "custom" {
		result.Protagonist = ""
		result.Genre = ""
		result.Mood = ""
		result.KeyScene = ""
		result.CoverTitle = ""
		return
	}
	result.Prompt = ""
}

// PolishCoverPrompt 用已配置的文案模型完善封面提示词或补全小说驱动字段。
func (h *StudioHandler) PolishCoverPrompt(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	ak, model, ok := h.requireStudioTextModel(c, subject.UserID)
	if !ok {
		return
	}
	var req coverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Mode = strings.TrimSpace(req.Mode)
	if req.Mode == "" {
		req.Mode = "novel"
	}
	if req.Mode != "custom" && req.Mode != "novel" {
		response.BadRequest(c, "不支持的封面生成模式")
		return
	}
	if req.Mode == "custom" && strings.TrimSpace(req.Prompt) == "" {
		response.BadRequest(c, "请先填写提示词")
		return
	}
	if req.Mode == "novel" && strings.TrimSpace(req.Synopsis) == "" {
		response.BadRequest(c, "请先填写简介")
		return
	}

	system, user := buildCoverPromptPolishPrompt(req)
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
		response.ErrorFrom(c, fmt.Errorf("提示词完善失败：%w", err))
		return
	}
	var result coverPromptPolishResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		response.ErrorFrom(c, fmt.Errorf("解析模型输出失败：%w", err))
		return
	}
	trimCoverPromptPolishResponse(req.Mode, req, &result)
	if req.Mode == "custom" && result.Prompt == "" {
		response.Error(c, http.StatusBadGateway, "模型未返回可用提示词")
		return
	}
	response.Success(c, result)
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
		"model":        model,
		"instructions": system,
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

type fanqieRankChannel string

const (
	fanqieRankHot    fanqieRankChannel = "hot"
	fanqieRankPeak   fanqieRankChannel = "peak"
	fanqieRankMale   fanqieRankChannel = "male"
	fanqieRankFemale fanqieRankChannel = "female"
)

type fanqieBook struct {
	ID          string   `json:"id"`
	Rank        int      `json:"rank"`
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	Category    string   `json:"category"`
	Status      string   `json:"status"`
	WordCount   string   `json:"word_count"`
	Score       string   `json:"score"`
	Description string   `json:"description"`
	CoverURL    string   `json:"cover_url,omitempty"`
	SourceURL   string   `json:"source_url"`
	Tags        []string `json:"tags"`
}

type fanqieRankResponse struct {
	Channel   fanqieRankChannel `json:"channel"`
	UpdatedAt time.Time         `json:"updated_at"`
	Source    string            `json:"source"`
	Books     []fanqieBook      `json:"books"`
}

type fanqieSeedBook struct {
	Title       string
	Author      string
	Category    string
	Status      string
	WordCount   string
	Description string
	Tags        []string
}

func normalizeFanqieRankChannel(raw string) fanqieRankChannel {
	switch fanqieRankChannel(strings.ToLower(strings.TrimSpace(raw))) {
	case fanqieRankPeak:
		return fanqieRankPeak
	case fanqieRankMale:
		return fanqieRankMale
	case fanqieRankFemale:
		return fanqieRankFemale
	default:
		return fanqieRankHot
	}
}

func fanqieRankLabel(ch fanqieRankChannel) string {
	switch ch {
	case fanqieRankPeak:
		return "巅峰榜"
	case fanqieRankMale:
		return "男生榜"
	case fanqieRankFemale:
		return "女生榜"
	default:
		return "热榜"
	}
}

type fanqieRankCacheEntry struct {
	Books     []fanqieBook
	UpdatedAt time.Time
}

type fanqieRankSnapshot struct {
	Books     []fanqieBook
	Source    string
	FetchedAt time.Time
}

type fanqieCoverSourceEntry struct {
	RawURL    string
	CreatedAt time.Time
}

type fanqieCoverCacheEntry struct {
	ContentType string
	Body        []byte
	UpdatedAt   time.Time
}

type fanqieRankAPISource struct {
	Gender     int
	RankMold   int
	CategoryID int
	Category   string
	Limit      int
}

var fanqieRankOfficialFetchTimeout = 8 * time.Second

const fanqieRankSnapshotFreshTTL = 6 * time.Hour
const fanqieCoverCacheTTL = 24 * time.Hour
const fanqieCoverSourceTTL = 7 * 24 * time.Hour
const fanqieCoverFetchTimeout = 12 * time.Second
const fanqieCoverMaxBytes = 4 << 20

var fanqieRankCache = struct {
	sync.Mutex
	entries map[fanqieRankChannel]fanqieRankCacheEntry
}{entries: map[fanqieRankChannel]fanqieRankCacheEntry{}}

var fanqieCoverSources = struct {
	sync.Mutex
	entries map[string]fanqieCoverSourceEntry
}{entries: map[string]fanqieCoverSourceEntry{}}

var fanqieCoverBytesCache = struct {
	sync.Mutex
	entries map[string]fanqieCoverCacheEntry
}{entries: map[string]fanqieCoverCacheEntry{}}

var fetchFanqieRankBooksFromOfficialFunc = fetchFanqieRankBooksFromOfficial

func fanqieRankAPISources(ch fanqieRankChannel) []fanqieRankAPISource {
	maleRead := []fanqieRankAPISource{
		{Gender: 1, RankMold: 2, CategoryID: 262, Category: "都市脑洞", Limit: 12},
		{Gender: 1, RankMold: 2, CategoryID: 539, Category: "悬疑脑洞", Limit: 12},
		{Gender: 1, RankMold: 2, CategoryID: 261, Category: "都市日常", Limit: 12},
		{Gender: 1, RankMold: 2, CategoryID: 258, Category: "传统玄幻", Limit: 12},
	}
	femaleRead := []fanqieRankAPISource{
		{Gender: 0, RankMold: 2, CategoryID: 1139, Category: "古风世情", Limit: 12},
		{Gender: 0, RankMold: 2, CategoryID: 267, Category: "现言脑洞", Limit: 12},
		{Gender: 0, RankMold: 2, CategoryID: 23, Category: "种田", Limit: 12},
		{Gender: 0, RankMold: 2, CategoryID: 24, Category: "快穿", Limit: 12},
	}
	maleNew := []fanqieRankAPISource{
		{Gender: 1, RankMold: 1, CategoryID: 262, Category: "都市脑洞", Limit: 10},
		{Gender: 1, RankMold: 1, CategoryID: 539, Category: "悬疑脑洞", Limit: 10},
	}
	femaleNew := []fanqieRankAPISource{
		{Gender: 0, RankMold: 1, CategoryID: 1139, Category: "古风世情", Limit: 10},
		{Gender: 0, RankMold: 1, CategoryID: 267, Category: "现言脑洞", Limit: 10},
	}
	switch ch {
	case fanqieRankMale:
		return maleRead
	case fanqieRankFemale:
		return femaleRead
	case fanqieRankPeak:
		return append(append([]fanqieRankAPISource{}, maleRead...), femaleRead...)
	default:
		sources := append([]fanqieRankAPISource{}, maleNew...)
		sources = append(sources, femaleNew...)
		sources = append(sources, maleRead[:2]...)
		return append(sources, femaleRead[:2]...)
	}
}

func fanqieRankAPIURL(src fanqieRankAPISource) string {
	limit := src.Limit
	if limit <= 0 {
		limit = 10
	}
	q := url.Values{}
	q.Set("app_id", "2503")
	q.Set("rank_list_type", "3")
	q.Set("offset", "0")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("category_id", strconv.Itoa(src.CategoryID))
	q.Set("rank_version", "")
	q.Set("gender", strconv.Itoa(src.Gender))
	q.Set("rankMold", strconv.Itoa(src.RankMold))
	return "https://fanqienovel.com/api/rank/category/list?" + q.Encode()
}

func fanqieRankURL(ch fanqieRankChannel) string {
	switch ch {
	case fanqieRankMale:
		return "https://fanqienovel.com/rank/1_0_0"
	case fanqieRankFemale:
		return "https://fanqienovel.com/rank/0_0_0"
	case fanqieRankPeak:
		return "https://fanqienovel.com/rank/0_1_0"
	default:
		return "https://fanqienovel.com/rank"
	}
}

func isFanqieCoverHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	if host == "localhost" || strings.HasPrefix(host, "127.") || host == "::1" {
		return true
	}
	return host == "fanqienovel.com" ||
		host == "fqnovelpic.com" ||
		strings.HasSuffix(host, ".fqnovelpic.com") ||
		strings.HasSuffix(host, ".fanqienovel.com") ||
		strings.HasSuffix(host, ".pstatp.com") ||
		strings.HasSuffix(host, ".byteimg.com") ||
		strings.HasSuffix(host, ".bytedance.com") ||
		strings.HasSuffix(host, ".toutiaoimg.com")
}

func normalizeFanqieCoverSourceURL(raw string) (string, bool) {
	normalized := normalizeFanqieImageURL(raw)
	if normalized == "" {
		return "", false
	}
	u, err := url.Parse(normalized)
	if err != nil || u.Hostname() == "" {
		return "", false
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", false
	}
	if u.Scheme == "http" && u.Hostname() != "localhost" && !strings.HasPrefix(u.Hostname(), "127.") && u.Hostname() != "::1" {
		return "", false
	}
	if !isFanqieCoverHost(u.Hostname()) {
		return "", false
	}
	return u.String(), true
}

func fanqieCoverCacheKey(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(sum[:])
}

func registerFanqieCoverSource(rawURL string) (string, bool) {
	normalized, ok := normalizeFanqieCoverSourceURL(rawURL)
	if !ok {
		return "", false
	}
	key := fanqieCoverCacheKey(normalized)
	now := time.Now().UTC()
	fanqieCoverSources.Lock()
	fanqieCoverSources.entries[key] = fanqieCoverSourceEntry{RawURL: normalized, CreatedAt: now}
	fanqieCoverSources.Unlock()
	return key, true
}

func fanqieCoverLocalURL(rawURL string) string {
	key, ok := registerFanqieCoverSource(rawURL)
	if !ok {
		return rawURL
	}
	return "/api/v1/studio/fanqie/covers/" + key
}

func withFanqieLocalCoverURLs(books []fanqieBook) []fanqieBook {
	out := make([]fanqieBook, len(books))
	copy(out, books)
	for i := range out {
		if out[i].CoverURL != "" {
			out[i].CoverURL = fanqieCoverLocalURL(out[i].CoverURL)
		}
	}
	return out
}

func lookupFanqieCoverSource(key string) (string, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false
	}
	now := time.Now().UTC()
	fanqieCoverSources.Lock()
	defer fanqieCoverSources.Unlock()
	entry, ok := fanqieCoverSources.entries[key]
	if !ok || now.Sub(entry.CreatedAt) > fanqieCoverSourceTTL {
		delete(fanqieCoverSources.entries, key)
		return "", false
	}
	return entry.RawURL, true
}

func getCachedFanqieCoverBytes(key string) (fanqieCoverCacheEntry, bool) {
	now := time.Now().UTC()
	fanqieCoverBytesCache.Lock()
	defer fanqieCoverBytesCache.Unlock()
	entry, ok := fanqieCoverBytesCache.entries[key]
	if !ok || now.Sub(entry.UpdatedAt) > fanqieCoverCacheTTL || len(entry.Body) == 0 {
		delete(fanqieCoverBytesCache.entries, key)
		return fanqieCoverCacheEntry{}, false
	}
	body := make([]byte, len(entry.Body))
	copy(body, entry.Body)
	entry.Body = body
	return entry, true
}

func setCachedFanqieCoverBytes(key string, entry fanqieCoverCacheEntry) {
	body := make([]byte, len(entry.Body))
	copy(body, entry.Body)
	entry.Body = body
	entry.UpdatedAt = time.Now().UTC()
	fanqieCoverBytesCache.Lock()
	fanqieCoverBytesCache.entries[key] = entry
	fanqieCoverBytesCache.Unlock()
}

func fetchFanqieCoverBytes(ctx context.Context, rawURL string) (fanqieCoverCacheEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, fanqieCoverFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fanqieCoverCacheEntry{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LanfanqieStudioCoverCache/1.0; +https://qbook.top)")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://fanqienovel.com/")
	client := &http.Client{
		Timeout: fanqieCoverFetchTimeout,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if _, ok := normalizeFanqieCoverSourceURL(req.URL.String()); !ok {
				return fmt.Errorf("fanqie cover redirect target is not allowed")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fanqieCoverCacheEntry{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fanqieCoverCacheEntry{}, fmt.Errorf("official cover returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, fanqieCoverMaxBytes+1))
	if err != nil {
		return fanqieCoverCacheEntry{}, err
	}
	if len(body) == 0 {
		return fanqieCoverCacheEntry{}, fmt.Errorf("official cover is empty")
	}
	if len(body) > fanqieCoverMaxBytes {
		return fanqieCoverCacheEntry{}, fmt.Errorf("official cover is too large")
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return fanqieCoverCacheEntry{}, fmt.Errorf("official cover returned non-image content")
	}
	return fanqieCoverCacheEntry{ContentType: contentType, Body: body, UpdatedAt: time.Now().UTC()}, nil
}

func writeFanqieCoverBytes(c *gin.Context, entry fanqieCoverCacheEntry, cacheStatus string) {
	if entry.ContentType == "" {
		entry.ContentType = http.DetectContentType(entry.Body)
	}
	c.Header("Cache-Control", "public, max-age=86400, immutable")
	c.Header("X-Fanqie-Cover-Cache", cacheStatus)
	c.Data(http.StatusOK, entry.ContentType, entry.Body)
}

// ServeFanqieCover serves Fanqie cover images through a small local cache so
// hotlist pages do not make every browser view hit the upstream image host.
func (h *StudioHandler) ServeFanqieCover(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(key) {
		response.BadRequest(c, "invalid cover cache key")
		return
	}
	rawURL, ok := lookupFanqieCoverSource(key)
	if !ok {
		response.Error(c, http.StatusNotFound, "cover cache source expired")
		return
	}
	if entry, ok := getCachedFanqieCoverBytes(key); ok {
		writeFanqieCoverBytes(c, entry, "hit")
		return
	}
	entry, err := fetchFanqieCoverBytes(c.Request.Context(), rawURL)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "cover cache fetch failed")
		return
	}
	setCachedFanqieCoverBytes(key, entry)
	writeFanqieCoverBytes(c, entry, "miss")
}

func containsPrivateUseRune(s string) bool {
	for _, r := range s {
		if r >= 0xE000 && r <= 0xF8FF {
			return true
		}
	}
	return false
}

func parseFanqieRankHTML(ch fanqieRankChannel, pageHTML string) ([]fanqieBook, error) {
	stateJSON, ok := extractInitialStateJSON(pageHTML)
	if !ok {
		return nil, fmt.Errorf("official rank state not found")
	}
	var state map[string]any
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, err
	}
	rank, _ := state["rank"].(map[string]any)
	if rank == nil {
		return nil, fmt.Errorf("official rank data not found")
	}
	rawList, _ := rank["book_list"].([]any)
	if len(rawList) == 0 {
		return nil, fmt.Errorf("official rank list is empty")
	}
	books := make([]fanqieBook, 0, len(rawList))
	for i, raw := range rawList {
		item, _ := raw.(map[string]any)
		if item == nil {
			continue
		}
		title := asString(item["bookName"])
		if title == "" || containsPrivateUseRune(title) {
			continue
		}
		id := asString(item["bookId"])
		if id == "" {
			continue
		}
		rankNo := asInt(item["currentPos"])
		if rankNo <= 0 {
			rankNo = i + 1
		}
		status := "连载中"
		if asString(item["creationStatus"]) == "0" {
			status = "已完结"
		}
		wordCount := "-"
		if n := asInt(item["wordNumber"]); n > 0 {
			wordCount = fmt.Sprintf("%d万字", n/10000)
		}
		score := "官方榜单"
		if read := asString(item["read_count"]); read != "" {
			score = "阅读 " + read
		}
		books = append(books, fanqieBook{
			ID:          id,
			Rank:        rankNo,
			Title:       title,
			Author:      asString(item["author"]),
			Category:    titleOr(asString(item["categoryV2"]), asString(item["category"])),
			Status:      status,
			WordCount:   wordCount,
			Score:       score,
			Description: html.UnescapeString(asString(item["abstract"])),
			CoverURL:    strings.ReplaceAll(asString(item["thumbUri"]), `\u002F`, "/"),
			SourceURL:   "https://fanqienovel.com/page/" + id,
			Tags:        []string{fanqieRankLabel(ch), "官方榜单"},
		})
	}
	if len(books) == 0 {
		return nil, fmt.Errorf("official rank list is obfuscated")
	}
	return books, nil
}

func parseFanqieRankAPIResponse(ch fanqieRankChannel, fallbackCategory string, body []byte) ([]fanqieBook, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if code := asInt(payload["code"]); code != 0 {
		return nil, fmt.Errorf("official rank api returned code %d", code)
	}
	data, _ := payload["data"].(map[string]any)
	if data == nil {
		return nil, fmt.Errorf("official rank api data not found")
	}
	rawList, _ := data["book_list"].([]any)
	if len(rawList) == 0 {
		return nil, fmt.Errorf("official rank api list is empty")
	}
	books := make([]fanqieBook, 0, len(rawList))
	for i, raw := range rawList {
		item, _ := raw.(map[string]any)
		if item == nil {
			continue
		}
		title := asString(item["bookName"])
		id := asString(item["bookId"])
		if title == "" || id == "" {
			continue
		}
		rankNo := asInt(item["currentPos"])
		if rankNo <= 0 {
			rankNo = i + 1
		}
		status := "连载中"
		if asString(item["creationStatus"]) == "0" {
			status = "已完结"
		}
		wordCount := "-"
		if n := asInt(item["wordNumber"]); n > 0 {
			if n >= 10000 {
				wordCount = fmt.Sprintf("%d万字", n/10000)
			} else {
				wordCount = fmt.Sprintf("%d字", n)
			}
		}
		score := "官方实时榜"
		if read := asString(item["read_count"]); read != "" {
			score = "在读 " + read
		}
		category := titleOr(asString(item["categoryV2"]), asString(item["category"]))
		category = titleOr(category, fallbackCategory)
		books = append(books, fanqieBook{
			ID:          id,
			Rank:        rankNo,
			Title:       title,
			Author:      asString(item["author"]),
			Category:    category,
			Status:      status,
			WordCount:   wordCount,
			Score:       score,
			Description: html.UnescapeString(asString(item["abstract"])),
			CoverURL:    strings.ReplaceAll(asString(item["thumbUri"]), `\u002F`, "/"),
			SourceURL:   "https://fanqienovel.com/page/" + id,
			Tags:        []string{fanqieRankLabel(ch), "官方实时榜", category},
		})
	}
	if len(books) == 0 {
		return nil, fmt.Errorf("official rank api list is empty after normalization")
	}
	return books, nil
}

func fanqieBookReadScore(book fanqieBook) int {
	digits := regexp.MustCompile(`\d+`).FindAllString(book.Score, -1)
	if len(digits) == 0 {
		return 0
	}
	joined := strings.Join(digits, "")
	n, _ := strconv.Atoi(joined)
	return n
}

func mergeFanqieRankBooks(books []fanqieBook) []fanqieBook {
	seen := make(map[string]bool, len(books))
	merged := make([]fanqieBook, 0, len(books))
	for _, book := range books {
		key := book.ID
		if key == "" {
			key = book.Title + "|" + book.Author
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, book)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		left := fanqieBookReadScore(merged[i])
		right := fanqieBookReadScore(merged[j])
		if left == right {
			return merged[i].Rank < merged[j].Rank
		}
		return left > right
	})
	if len(merged) > 30 {
		merged = merged[:30]
	}
	for i := range merged {
		merged[i].Rank = i + 1
	}
	return merged
}

func fetchFanqieRankBooksFromOfficialAPI(ctx context.Context, ch fanqieRankChannel) ([]fanqieBook, error) {
	var all []fanqieBook
	var errs []string
	for _, src := range fanqieRankAPISources(ch) {
		body, err := fetchURLText(ctx, fanqieRankAPIURL(src))
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		books, err := parseFanqieRankAPIResponse(ch, src.Category, []byte(body))
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		all = append(all, books...)
		if len(mergeFanqieRankBooks(all)) >= 30 {
			break
		}
	}
	merged := mergeFanqieRankBooks(all)
	merged = enrichFanqieObfuscatedRankBooks(ctx, merged)
	if len(merged) == 0 {
		return nil, fmt.Errorf("official rank api unavailable: %s", strings.Join(errs, "; "))
	}
	return merged, nil
}

func fetchFanqieRankBooksFromOfficial(ctx context.Context, ch fanqieRankChannel) ([]fanqieBook, error) {
	if books, err := fetchFanqieRankBooksFromOfficialAPI(ctx, ch); err == nil && len(books) > 0 {
		return books, nil
	}
	pageHTML, err := fetchURLText(ctx, fanqieRankURL(ch))
	if err != nil {
		return nil, err
	}
	return parseFanqieRankHTML(ch, pageHTML)
}

func fanqieRankSQLPlaceholder(dialectName string, index int) string {
	if dialectName == "postgres" {
		return "$" + strconv.Itoa(index)
	}
	return "?"
}

func fanqieRankSnapshotTableMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "fanqie_rank_snapshots") &&
		(strings.Contains(msg, "no such table") || strings.Contains(msg, "does not exist"))
}

func loadFanqieRankSnapshot(ctx context.Context, client *dbent.Client, ch fanqieRankChannel) (fanqieRankSnapshot, bool) {
	if client == nil {
		return fanqieRankSnapshot{}, false
	}
	ph := fanqieRankSQLPlaceholder(client.Driver().Dialect(), 1)
	rows, err := client.QueryContext(ctx, "SELECT books, source, fetched_at FROM fanqie_rank_snapshots WHERE channel = "+ph, string(ch))
	if err != nil {
		if !fanqieRankSnapshotTableMissing(err) {
			slog.Warn("fanqie rank snapshot read failed", "channel", ch, "error", err)
		}
		return fanqieRankSnapshot{}, false
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return fanqieRankSnapshot{}, false
	}
	var booksRaw string
	var source string
	var fetchedAt time.Time
	if err := rows.Scan(&booksRaw, &source, &fetchedAt); err != nil {
		slog.Warn("fanqie rank snapshot scan failed", "channel", ch, "error", err)
		return fanqieRankSnapshot{}, false
	}
	var books []fanqieBook
	if err := json.Unmarshal([]byte(booksRaw), &books); err != nil || len(books) == 0 {
		slog.Warn("fanqie rank snapshot decode failed", "channel", ch, "error", err)
		return fanqieRankSnapshot{}, false
	}
	return fanqieRankSnapshot{Books: books, Source: source, FetchedAt: fetchedAt}, true
}

func saveFanqieRankSnapshot(ctx context.Context, client *dbent.Client, ch fanqieRankChannel, books []fanqieBook, source string, fetchedAt time.Time) {
	if client == nil || len(books) == 0 {
		return
	}
	rawBooks, err := json.Marshal(books)
	if err != nil {
		slog.Warn("fanqie rank snapshot encode failed", "channel", ch, "error", err)
		return
	}
	dialectName := client.Driver().Dialect()
	var query string
	if dialectName == "postgres" {
		query = `
INSERT INTO fanqie_rank_snapshots (channel, books, source, fetched_at, created_at, updated_at)
VALUES ($1, $2::jsonb, $3, $4, NOW(), NOW())
ON CONFLICT (channel) DO UPDATE SET
	books = EXCLUDED.books,
	source = EXCLUDED.source,
	fetched_at = EXCLUDED.fetched_at,
	updated_at = NOW()`
	} else {
		query = `
INSERT INTO fanqie_rank_snapshots (channel, books, source, fetched_at, created_at, updated_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(channel) DO UPDATE SET
	books = excluded.books,
	source = excluded.source,
	fetched_at = excluded.fetched_at,
	updated_at = CURRENT_TIMESTAMP`
	}
	if _, err := client.ExecContext(ctx, query, string(ch), string(rawBooks), source, fetchedAt); err != nil {
		if !fanqieRankSnapshotTableMissing(err) {
			slog.Warn("fanqie rank snapshot write failed", "channel", ch, "error", err)
		}
	}
}

func liveFanqieRankBooks(ctx context.Context, client *dbent.Client, ch fanqieRankChannel, forceRefresh bool) ([]fanqieBook, string, time.Time) {
	if !forceRefresh {
		if snapshot, ok := loadFanqieRankSnapshot(ctx, client, ch); ok {
			books := make([]fanqieBook, len(snapshot.Books))
			copy(books, snapshot.Books)
			if time.Since(snapshot.FetchedAt) <= fanqieRankSnapshotFreshTTL {
				return books, "fanqie-db-cache", snapshot.FetchedAt
			}
			return books, "fanqie-db-stale", snapshot.FetchedAt
		}
	}

	now := time.Now().UTC()
	if !forceRefresh {
		fanqieRankCache.Lock()
		if cached, ok := fanqieRankCache.entries[ch]; ok && now.Sub(cached.UpdatedAt) < 30*time.Minute && len(cached.Books) > 0 {
			books := make([]fanqieBook, len(cached.Books))
			copy(books, cached.Books)
			fanqieRankCache.Unlock()
			return books, "fanqie-official-cache", cached.UpdatedAt
		}
		fanqieRankCache.Unlock()
	}

	fanqieRankCache.Lock()
	delete(fanqieRankCache.entries, ch)
	fanqieRankCache.Unlock()

	fetchCtx, cancel := context.WithTimeout(ctx, fanqieRankOfficialFetchTimeout)
	defer cancel()

	books, err := fetchFanqieRankBooksFromOfficialFunc(fetchCtx, ch)
	if err == nil && len(books) >= 30 {
		fanqieRankCache.Lock()
		fanqieRankCache.entries[ch] = fanqieRankCacheEntry{Books: books, UpdatedAt: now}
		fanqieRankCache.Unlock()
		saveFanqieRankSnapshot(ctx, client, ch, books, "fanqie-official", now)
		return books, "fanqie-official", now
	}

	if snapshot, ok := loadFanqieRankSnapshot(ctx, client, ch); ok {
		books := make([]fanqieBook, len(snapshot.Books))
		copy(books, snapshot.Books)
		return books, "fanqie-db-stale", snapshot.FetchedAt
	}

	books = buildFanqieRankBooks(ch)
	saveFanqieRankSnapshot(ctx, client, ch, books, "fanqie-rank-cache", now)
	return books, "fanqie-rank-cache", now
}

func fanqieSeeds(ch fanqieRankChannel) []fanqieSeedBook {
	common := []fanqieSeedBook{
		{Title: "十日终焉", Author: "杀虫队队员", Category: "悬疑脑洞", Status: "已完结", WordCount: "240万字", Description: "以强规则、群像博弈和连续反转建立高讨论度，适合拆解悬疑爽点和章节尾钩。", Tags: []string{"规则怪谈", "强反转"}},
		{Title: "我在精神病院学斩神", Author: "三九音域", Category: "都市脑洞", Status: "已完结", WordCount: "420万字", Description: "用现代都市和神话体系做反差，开篇钩子明确，角色团体记忆点强。", Tags: []string{"都市脑洞", "群像"}},
		{Title: "异兽迷城", Author: "彭湃", Category: "悬疑脑洞", Status: "已完结", WordCount: "190万字", Description: "以身份悬疑和异能设定驱动剧情，适合研究设定揭示节奏。", Tags: []string{"身份悬疑", "异能"}},
	}

	switch ch {
	case fanqieRankPeak:
		return append(common, []fanqieSeedBook{
			{Title: "天渊", Author: "沐潇三生", Category: "传统玄幻", Status: "连载中", WordCount: "330万字", Description: "长线升级、宿命感和势力冲突并行，适合观察大长篇主线推进。", Tags: []string{"玄幻", "长线升级"}},
			{Title: "诡舍", Author: "夜来风雨声丶", Category: "悬疑灵异", Status: "连载中", WordCount: "180万字", Description: "副本式推进和悬念钩子密度高，适合拆章节危机递进。", Tags: []string{"副本", "悬疑"}},
			{Title: "开局停职？我转投纪委调查组", Author: "江门二爷", Category: "都市日常", Status: "连载中", WordCount: "260万字", Description: "现实向权谋升级，人物选择与信息差共同制造追读。", Tags: []string{"现实向", "权谋"}},
		}...)
	case fanqieRankMale:
		return append(common[:1], []fanqieSeedBook{
			{Title: "公考捡漏：从女友抛弃到权力巅峰", Author: "元明小阳", Category: "都市日常", Status: "连载中", WordCount: "170万字", Description: "低谷开局叠加职场上升线，主打现实逆袭和连续目标。", Tags: []string{"职场", "逆袭"}},
			{Title: "天眼风水师", Author: "道之光", Category: "都市脑洞", Status: "已完结", WordCount: "230万字", Description: "知识点包装成爽点，适合拆解专业题材的可信感。", Tags: []string{"玄学", "专业感"}},
			{Title: "宦海官途", Author: "风流小二", Category: "都市日常", Status: "连载中", WordCount: "390万字", Description: "基层困局到高位博弈，适合研究现实题材的升级节奏。", Tags: []string{"官场", "升级"}},
			{Title: "青梅暗恋我十年，还好我重生了", Author: "夜雨i", Category: "都市日常", Status: "连载中", WordCount: "110万字", Description: "重生补偿和情绪兑现明显，适合拆甜爽线的读者预期。", Tags: []string{"重生", "情绪兑现"}},
		}...)
	case fanqieRankFemale:
		return []fanqieSeedBook{
			{Title: "一介咸鱼，竟迎娶尚书嫡女", Author: "祈之安宁", Category: "古风世情", Status: "连载中", WordCount: "65万字", Description: "轻喜感男主视角和婚恋反差开局，适合拆人设反差。", Tags: []string{"古言", "轻喜"}},
			{Title: "穿越老朱后宫，我开局冒充长平", Author: "渺渺清音", Category: "历史古代", Status: "连载中", WordCount: "82万字", Description: "身份错位和历史人物互动制造强钩子。", Tags: []string{"穿越", "身份错位"}},
			{Title: "和亲五年，新帝逼我写下和离书", Author: "桃花山里桃花仙", Category: "古风世情", Status: "连载中", WordCount: "101万字", Description: "强情绪拉扯和关系张力突出，适合做追妻线对标。", Tags: []string{"追妻", "强情绪"}},
			{Title: "穿成开国皇帝病弱早逝的好大儿", Author: "青见", Category: "古言脑洞", Status: "连载中", WordCount: "95万字", Description: "胎穿、亲情和命运改写并行，开篇设定清晰。", Tags: []string{"胎穿", "亲情"}},
			{Title: "父皇他两辈子都在装", Author: "沐辉", Category: "古言脑洞", Status: "连载中", WordCount: "88万字", Description: "重生救赎和父子关系反差，情绪记忆点强。", Tags: []string{"重生", "救赎"}},
		}
	default:
		return append(common, []fanqieSeedBook{
			{Title: "公考捡漏：从女友抛弃到权力巅峰", Author: "元明小阳", Category: "都市日常", Status: "连载中", WordCount: "170万字", Description: "低谷开局叠加职场上升线，主打现实逆袭和连续目标。", Tags: []string{"职场", "逆袭"}},
			{Title: "天眼风水师", Author: "道之光", Category: "都市脑洞", Status: "已完结", WordCount: "230万字", Description: "知识点包装成爽点，适合拆解专业题材的可信感。", Tags: []string{"玄学", "专业感"}},
			{Title: "一介咸鱼，竟迎娶尚书嫡女", Author: "祈之安宁", Category: "古风世情", Status: "连载中", WordCount: "65万字", Description: "轻喜感男主视角和婚恋反差开局，适合拆人设反差。", Tags: []string{"古言", "轻喜"}},
		}...)
	}
}

func buildFanqieRankBooks(ch fanqieRankChannel) []fanqieBook {
	seeds := fanqieSeeds(ch)
	books := make([]fanqieBook, 0, 30)
	for i := 0; i < 30; i++ {
		seed := seeds[i%len(seeds)]
		title := seed.Title
		if i >= len(seeds) {
			title = fmt.Sprintf("%s样本 %02d · %s", fanqieRankLabel(ch), i+1, seed.Category)
		}
		book := fanqieBook{
			ID:          fmt.Sprintf("%s-%02d", ch, i+1),
			Rank:        i + 1,
			Title:       title,
			Author:      seed.Author,
			Category:    seed.Category,
			Status:      seed.Status,
			WordCount:   seed.WordCount,
			Score:       fmt.Sprintf("热度 %d", 990-i*13),
			Description: seed.Description,
			SourceURL:   "https://fanqienovel.com/search/" + title,
			Tags:        append([]string{}, seed.Tags...),
		}
		books = append(books, book)
	}
	return books
}

func filterFanqieBooks(query string, books []fanqieBook, limit int) []fanqieBook {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []fanqieBook{}
	}
	if limit <= 0 {
		limit = 20
	}
	seen := map[string]struct{}{}
	out := make([]fanqieBook, 0, limit)
	for _, book := range books {
		key := book.ID
		if key == "" {
			key = strings.ToLower(book.Title + "|" + book.Author)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		haystack := strings.ToLower(book.Title + book.Author + book.Category + strings.Join(book.Tags, ""))
		if !strings.Contains(haystack, query) {
			continue
		}
		book.Rank = len(out) + 1
		book.Score = "搜索命中"
		out = append(out, book)
		seen[key] = struct{}{}
		if len(out) >= limit {
			return out
		}
	}
	return out
}

func fanqieBookFromQuery(query string) (fanqieBook, bool) {
	id := ""
	if match := fanqiePageIDPattern.FindStringSubmatch(query); len(match) == 2 {
		id = match[1]
	} else if regexp.MustCompile(`^\d{10,}$`).MatchString(strings.TrimSpace(query)) {
		id = strings.TrimSpace(query)
	}
	if id == "" {
		return fanqieBook{}, false
	}
	return fanqieBook{
		ID:          id,
		Rank:        1,
		Title:       "番茄作品 " + id,
		Author:      "番茄小说",
		Category:    "指定作品",
		Status:      "待确认",
		WordCount:   "-",
		Score:       "作品 ID 命中",
		Description: "已识别为番茄作品链接/ID，可在右侧继续下载导入或分析首页与前 10 章。",
		SourceURL:   "https://fanqienovel.com/page/" + id,
		Tags:        []string{"指定作品"},
	}, true
}

func searchLiveFanqieBooks(ctx context.Context, query string) ([]fanqieBook, error) {
	if book, ok := fanqieBookFromQuery(query); ok {
		summary, err := fetchFanqieBookSummary(ctx, book)
		if err != nil {
			return []fanqieBook{book}, nil
		}
		return []fanqieBook{summary}, nil
	}
	books, err := fetchFanqieSearchBooks(ctx, strings.TrimSpace(query))
	if err != nil {
		return searchFanqieSeedBooks(query), nil
	}
	if len(books) == 0 {
		return searchFanqieSeedBooks(query), nil
	}
	return books, nil
}

func searchFanqieSeedBooks(query string) []fanqieBook {
	query = strings.TrimSpace(query)
	if query == "" {
		return []fanqieBook{}
	}
	seen := map[string]struct{}{}
	var out []fanqieBook
	for _, ch := range []fanqieRankChannel{fanqieRankHot, fanqieRankPeak, fanqieRankMale, fanqieRankFemale} {
		for _, book := range buildFanqieRankBooks(ch) {
			key := strings.ToLower(book.Title + "|" + book.Author)
			if _, ok := seen[key]; ok {
				continue
			}
			haystack := strings.ToLower(book.Title + book.Author + book.Category + strings.Join(book.Tags, ""))
			if strings.Contains(haystack, strings.ToLower(query)) || strings.Contains(book.Title, query) {
				book.Rank = len(out) + 1
				book.Score = "搜索命中"
				out = append(out, book)
				seen[key] = struct{}{}
			}
			if len(out) >= 20 {
				return out
			}
		}
	}
	if len(out) == 0 {
		out = append(out, fanqieBook{
			ID:          "search-" + strconv.Itoa(len([]rune(query))),
			Rank:        1,
			Title:       query,
			Author:      "番茄搜索",
			Category:    "指定搜索",
			Status:      "待确认",
			WordCount:   "-",
			Score:       "搜索入口",
			Description: "未在本地榜单缓存中命中，已生成番茄搜索入口，可继续按书名到官方页面核对。",
			SourceURL:   "https://fanqienovel.com/search/" + query,
			Tags:        []string{"指定小说"},
		})
	}
	return out
}

// GetFanqieRank 返回番茄榜单聚合数据。当前接口保证四个频道均可用，
// 页面只依赖本后端，后续可在这里替换为官方页面抓取或定时缓存。
func (h *StudioHandler) GetFanqieRank(c *gin.Context) {
	if _, ok := middleware2.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	ch := normalizeFanqieRankChannel(c.Query("channel"))
	forceRefresh := strings.EqualFold(c.Query("force_refresh"), "true") || c.Query("force_refresh") == "1"
	books, source, updatedAt := liveFanqieRankBooks(c.Request.Context(), h.client, ch, forceRefresh)
	response.Success(c, fanqieRankResponse{
		Channel:   ch,
		UpdatedAt: updatedAt,
		Source:    source,
		Books:     withFanqieLocalCoverURLs(books),
	})
}

// SearchFanqieBooks 按书名/作者/题材搜索番茄小说榜单缓存。
func (h *StudioHandler) SearchFanqieBooks(c *gin.Context) {
	if _, ok := middleware2.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		response.BadRequest(c, "请输入要搜索的小说名或作者")
		return
	}
	books, err := searchLiveFanqieBooks(c.Request.Context(), query)
	if err != nil {
		response.Error(c, http.StatusBadGateway, err.Error())
		return
	}
	response.Success(c, gin.H{"books": withFanqieLocalCoverURLs(books)})
}

type fanqieChapter struct {
	Index        int    `json:"index"`
	ItemID       string `json:"item_id"`
	Title        string `json:"title"`
	WordCount    int    `json:"word_count,omitempty"`
	Content      string `json:"content,omitempty"`
	Readable     bool   `json:"readable"`
	SourceURL    string `json:"source_url,omitempty"`
	DecodeStatus string `json:"decode_status,omitempty"`
}

type fanqieBookDetail struct {
	Book            fanqieBook      `json:"book"`
	Chapters        []fanqieChapter `json:"chapters"`
	ReadableContent bool            `json:"readable_content"`
	FontEncoded     bool            `json:"font_encoded"`
	PartialContent  bool            `json:"partial_content"`
	Blocked         bool            `json:"blocked"`
	Source          string          `json:"source"`
	Notes           []string        `json:"notes"`
}

type fanqieBookActionRequest struct {
	Book    fanqieBook `json:"book"`
	Consent bool       `json:"consent"`
}

type fanqieDownloadResult struct {
	Title        string                `json:"title"`
	FileName     string                `json:"file_name"`
	Status       string                `json:"status"`
	ChapterCount int                   `json:"chapter_count"`
	Text         string                `json:"text"`
	Chapters     []studioImportChapter `json:"chapters"`
	Notes        []string              `json:"notes"`
	Source       string                `json:"source"`
	DecodeStatus string                `json:"decode_status"`
}

type fanqieAnalysisReport struct {
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Hooks        []string `json:"hooks"`
	ChapterNotes []struct {
		Title string `json:"title"`
		Note  string `json:"note"`
	} `json:"chapter_notes"`
	Actions []string `json:"actions"`
	Source  string   `json:"source,omitempty"`
	Notes   []string `json:"notes,omitempty"`
}

const fanqieAnalysisChapterLimit = 10

var (
	errFanqieOfficialSearchBlocked = errors.New("番茄官方搜索接口触发验证码校验，暂时无法自动按书名搜索；请粘贴番茄作品页链接或作品 ID 后重试")
	errFanqieBrowserRequired       = errors.New("番茄官方需要浏览器安全校验")
	fetchFanqieSearchBooks         = fetchFanqieSearchBooksFromOfficial
	fetchFanqieBookSummary         = fetchFanqieBookSummaryFromOfficial
	fetchFanqieBookDetail          = fetchFanqieBookDetailFromOfficial
	fetchFanqieBookAnalysisDetail  = fetchFanqieBookAnalysisDetailFromOfficial
)

func normalizeFanqieActionBook(book fanqieBook) fanqieBook {
	book.ID = strings.TrimSpace(book.ID)
	book.Title = strings.TrimSpace(book.Title)
	book.Author = strings.TrimSpace(book.Author)
	book.Category = strings.TrimSpace(book.Category)
	book.Status = strings.TrimSpace(book.Status)
	book.WordCount = strings.TrimSpace(book.WordCount)
	book.Description = strings.TrimSpace(book.Description)
	book.SourceURL = strings.TrimSpace(book.SourceURL)
	if book.SourceURL == "" && book.ID != "" {
		book.SourceURL = "https://fanqienovel.com/page/" + book.ID
	}
	return book
}

var fanqiePageIDPattern = regexp.MustCompile(`/page/([0-9]+)`)

func fanqieBookID(book fanqieBook) string {
	if strings.TrimSpace(book.ID) != "" && regexp.MustCompile(`^[0-9]+$`).MatchString(strings.TrimSpace(book.ID)) {
		return strings.TrimSpace(book.ID)
	}
	if match := fanqiePageIDPattern.FindStringSubmatch(book.SourceURL); len(match) == 2 {
		return match[1]
	}
	return strings.TrimSpace(book.ID)
}

func fetchURLText(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LanfanqieStudio/1.0; +https://qbook.top)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", "https://fanqienovel.com/")
	client := &http.Client{Timeout: 18 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("official page returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func fanqieBookInfoAPIURL(bookID string) string {
	q := url.Values{}
	q.Set("bookId", bookID)
	return "https://fanqienovel.com/api/book/info?" + q.Encode()
}

func fanqieSearchAPIURL(query string) string {
	q := url.Values{}
	q.Set("filter", "127,127,127,127")
	q.Set("page_count", "10")
	q.Set("page_index", "0")
	q.Set("query_type", "0")
	q.Set("query_word", query)
	return "https://fanqienovel.com/api/author/search/search_book/v1?" + q.Encode()
}

func parseFanqieBookInfoAPIResponse(body []byte, fallback fanqieBook) (fanqieBook, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return fanqieBook{}, fmt.Errorf("official book info is empty")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fanqieBook{}, err
	}
	if code := asInt(payload["code"]); code != 0 {
		message := asString(payload["message"])
		if message == "" {
			message = fmt.Sprintf("official book info returned code %d", code)
		}
		return fanqieBook{}, errors.New(message)
	}
	data, _ := payload["data"].(map[string]any)
	if len(data) == 0 {
		return fanqieBook{}, fmt.Errorf("official book info data not found")
	}
	book := normalizeFanqieActionBook(fallback)
	if s := asString(data["bookId"]); s != "" {
		book.ID = s
	}
	if s := asString(data["bookName"]); s != "" {
		book.Title = s
	}
	if s := asString(data["author"]); s != "" {
		book.Author = s
	}
	if book.Author == "" {
		book.Author = asString(data["authorName"])
	}
	book.Category = titleOr(fanqieCategoryFromValue(data["category"]), fanqieCategoryFromValue(data["categoryV2"]))
	if book.Category == "" {
		book.Category = fallback.Category
	}
	if s := asString(data["abstract"]); s != "" {
		book.Description = html.UnescapeString(s)
	}
	if n := asInt(data["wordNumber"]); n > 0 {
		book.WordCount = fanqieWordCountText(n)
	}
	if status := fanqieCreationStatusText(data["creationStatus"]); status != "" {
		book.Status = status
	}
	book.CoverURL = normalizeFanqieImageURL(firstNonEmptyString(asString(data["thumbUrl"]), asString(data["thumbUri"])))
	if read := asString(data["readCount"]); read != "" {
		book.Score = "在读 " + read
	}
	if book.Score == "" {
		book.Score = "官方详情"
	}
	if book.ID != "" {
		book.SourceURL = "https://fanqienovel.com/page/" + book.ID
	}
	if len(book.Tags) == 0 {
		book.Tags = []string{"官方详情"}
	}
	if book.Category != "" && !strings.Contains(strings.Join(book.Tags, "|"), book.Category) {
		book.Tags = append(book.Tags, book.Category)
	}
	return book, nil
}

func parseFanqieSearchAPIResponse(body []byte) ([]fanqieBook, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, errFanqieOfficialSearchBlocked
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if code := asInt(payload["code"]); code != 0 {
		message := asString(payload["message"])
		if message == "" {
			message = fmt.Sprintf("official search returned code %d", code)
		}
		return nil, errors.New(message)
	}
	data, _ := payload["data"].(map[string]any)
	if data == nil {
		return nil, fmt.Errorf("official search data not found")
	}
	rawList, _ := data["search_book_data_list"].([]any)
	if len(rawList) == 0 {
		rawList, _ = data["book_list"].([]any)
	}
	books := make([]fanqieBook, 0, len(rawList))
	for i, raw := range rawList {
		item, _ := raw.(map[string]any)
		if item == nil {
			continue
		}
		id := asString(item["book_id"])
		if id == "" {
			id = asString(item["bookId"])
		}
		title := asString(item["book_name"])
		if title == "" {
			title = asString(item["bookName"])
		}
		if id == "" || title == "" {
			continue
		}
		category := fanqieCategoryFromValue(item["category"])
		status := fanqieCreationStatusText(item["creation_status"])
		wordCount := fanqieWordCountText(asInt(item["word_count"]))
		score := "官方搜索"
		if read := asString(item["read_count"]); read != "" {
			score = "在读 " + read
		}
		books = append(books, fanqieBook{
			ID:          id,
			Rank:        i + 1,
			Title:       title,
			Author:      asString(item["author"]),
			Category:    category,
			Status:      status,
			WordCount:   wordCount,
			Score:       score,
			Description: html.UnescapeString(asString(item["book_abstract"])),
			CoverURL:    normalizeFanqieImageURL(firstNonEmptyString(asString(item["thumb_url"]), asString(item["thumbUrl"]))),
			SourceURL:   "https://fanqienovel.com/page/" + id,
			Tags:        []string{"官方搜索", category},
		})
	}
	return books, nil
}

func fetchFanqieBookSummaryFromOfficial(ctx context.Context, book fanqieBook) (fanqieBook, error) {
	book = normalizeFanqieActionBook(book)
	bookID := fanqieBookID(book)
	if bookID == "" {
		return fanqieBook{}, fmt.Errorf("缺少番茄作品 ID 或页面链接")
	}
	body, err := fetchURLText(ctx, fanqieBookInfoAPIURL(bookID))
	if err == nil {
		if summary, parseErr := parseFanqieBookInfoAPIResponse([]byte(body), book); parseErr == nil {
			if summary.Rank == 0 {
				summary.Rank = 1
			}
			return summary, nil
		} else {
			err = parseErr
		}
	}
	sourceURL := "https://fanqienovel.com/page/" + bookID
	pageHTML, pageErr := fetchURLText(ctx, sourceURL)
	if pageErr != nil {
		if err != nil {
			return fanqieBook{}, fmt.Errorf("%w; %v", err, pageErr)
		}
		return fanqieBook{}, pageErr
	}
	detail, pageErr := parseFanqiePageHTML(pageHTML, sourceURL, book)
	if pageErr != nil {
		return fanqieBook{}, pageErr
	}
	summary := detail.Book
	if summary.Rank == 0 {
		summary.Rank = 1
	}
	if summary.Score == "" {
		summary.Score = "官方详情"
	}
	if len(summary.Tags) == 0 {
		summary.Tags = []string{"官方详情"}
	}
	return summary, nil
}

func fetchFanqieSearchBooksFromOfficial(ctx context.Context, query string) ([]fanqieBook, error) {
	body, err := fetchURLText(ctx, fanqieSearchAPIURL(query))
	if err != nil {
		return nil, err
	}
	books, err := parseFanqieSearchAPIResponse([]byte(body))
	if err != nil {
		return nil, err
	}
	for i := range books {
		summary, err := fetchFanqieBookSummary(ctx, books[i])
		if err != nil {
			continue
		}
		summary.Rank = i + 1
		if summary.Score == "" || summary.Score == "官方详情" {
			summary.Score = books[i].Score
		}
		if len(summary.Tags) == 0 {
			summary.Tags = books[i].Tags
		}
		books[i] = summary
	}
	return books, nil
}

func extractInitialStateJSON(pageHTML string) (string, bool) {
	marker := "window.__INITIAL_STATE__="
	idx := strings.Index(pageHTML, marker)
	if idx < 0 {
		return "", false
	}
	start := strings.Index(pageHTML[idx:], "{")
	if start < 0 {
		return "", false
	}
	pos := idx + start
	depth := 0
	inString := false
	escaped := false
	for i := pos; i < len(pageHTML); i++ {
		ch := pageHTML[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return pageHTML[pos : i+1], true
			}
		}
	}
	return "", false
}

func asString(v any) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return ""
	}
}

func asInt(v any) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(val))
		return i
	default:
		return 0
	}
}

func asBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return strings.EqualFold(strings.TrimSpace(val), "true") || strings.TrimSpace(val) == "1"
	case float64:
		return val != 0
	default:
		return false
	}
}

func fanqieCreationStatusText(v any) string {
	switch strings.TrimSpace(asString(v)) {
	case "0":
		return "已完结"
	case "1":
		return "连载中"
	default:
		return ""
	}
}

func fanqieWordCountText(n int) string {
	if n <= 0 {
		return ""
	}
	if n >= 10000 {
		return fmt.Sprintf("%d万字", n/10000)
	}
	return fmt.Sprintf("%d字", n)
}

func normalizeFanqieImageURL(raw string) string {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, `\u002F`, "/"))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return raw
}

func fanqieCategoryFromValue(v any) string {
	if s := asString(v); s != "" {
		if strings.HasPrefix(s, "[") {
			var rawList []any
			if err := json.Unmarshal([]byte(s), &rawList); err == nil {
				return fanqieCategoryFromValue(rawList)
			}
		}
		return s
	}
	rawList, ok := v.([]any)
	if !ok {
		return ""
	}
	names := make([]string, 0, 3)
	for _, raw := range rawList {
		item, _ := raw.(map[string]any)
		if item == nil {
			continue
		}
		name := asString(item["Name"])
		if name == "" {
			name = asString(item["name"])
		}
		if name == "" {
			continue
		}
		names = append(names, name)
		if len(names) >= 3 {
			break
		}
	}
	return strings.Join(names, " / ")
}

func parseFanqiePageHTML(pageHTML, sourceURL string, fallback fanqieBook) (fanqieBookDetail, error) {
	stateJSON, ok := extractInitialStateJSON(pageHTML)
	if !ok {
		return fanqieBookDetail{}, fmt.Errorf("official state not found")
	}
	var state map[string]any
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return fanqieBookDetail{}, err
	}
	page, _ := state["page"].(map[string]any)
	if page == nil {
		return fanqieBookDetail{}, fmt.Errorf("official page state not found")
	}

	book := normalizeFanqieActionBook(fallback)
	if s := asString(page["bookId"]); s != "" {
		book.ID = s
	}
	if s := asString(page["bookName"]); s != "" {
		book.Title = s
	}
	if s := asString(page["author"]); s != "" {
		book.Author = s
	}
	if s := asString(page["category"]); s != "" {
		book.Category = s
	}
	if book.Category == "" {
		book.Category = fanqieCategoryFromValue(page["categoryV2"])
	}
	if s := asString(page["abstract"]); s != "" {
		book.Description = html.UnescapeString(s)
	}
	if s := asString(page["thumbUri"]); s != "" {
		book.CoverURL = normalizeFanqieImageURL(s)
	}
	if n := asInt(page["wordNumber"]); n > 0 {
		book.WordCount = fanqieWordCountText(n)
	}
	if status := fanqieCreationStatusText(page["creationStatus"]); status != "" {
		book.Status = status
	}
	if book.SourceURL == "" {
		book.SourceURL = sourceURL
	}

	var chapters []fanqieChapter
	if groups, ok := page["chapterListWithVolume"].([]any); ok {
		for _, group := range groups {
			items, _ := group.([]any)
			for _, raw := range items {
				item, _ := raw.(map[string]any)
				if item == nil {
					continue
				}
				index := len(chapters) + 1
				if order := asInt(item["realChapterOrder"]); order > 0 {
					index = order
				}
				itemID := asString(item["itemId"])
				sourceURL := ""
				decodeStatus := "正文待采集。"
				if itemID == "" {
					decodeStatus = "章节缺少 reader itemId，无法采集正文。"
				} else {
					sourceURL = "https://fanqienovel.com/reader/" + itemID
					if asInt(item["needPay"]) > 0 ||
						asBool(item["isChapterLock"]) ||
						asBool(item["isPaidPublication"]) ||
						asBool(item["isPaidStory"]) {
						decodeStatus = "章节标记需要付费、登录或额外授权；将尝试采集公开 reader 可见正文。"
					}
				}
				chapters = append(chapters, fanqieChapter{
					Index:        index,
					ItemID:       itemID,
					Title:        asString(item["title"]),
					SourceURL:    sourceURL,
					Readable:     false,
					DecodeStatus: decodeStatus,
				})
			}
		}
	}
	return fanqieBookDetail{
		Book:            book,
		Chapters:        chapters,
		ReadableContent: false,
		Source:          "fanqie-official",
		Notes:           []string{"已从官方页面获取简介与目录；正文如遇字体混淆，会保留为待解码状态。"},
	}, nil
}

type fanqieURLTextFetcher func(context.Context, string) (string, error)

var fanqieReaderIDPattern = regexp.MustCompile(`/reader/([0-9]+)`)

func htmlFragmentToPlainText(fragment string) string {
	text := html.UnescapeString(fragment)
	text = regexp.MustCompile(`(?i)<\s*br\s*/?\s*>`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`(?i)</\s*p\s*>`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`(?is)<\s*script[^>]*>.*?</\s*script\s*>`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(?is)<\s*style[^>]*>.*?</\s*style\s*>`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(text, "")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func parseFanqieReaderHTML(readerHTML, sourceURL string) (fanqieChapter, error) {
	stateJSON, ok := extractInitialStateJSON(readerHTML)
	if !ok {
		if strings.Contains(readerHTML, "验证码中间页") || strings.Contains(readerHTML, "captcha/index.js") {
			return fanqieChapter{}, fmt.Errorf("%w：reader 返回验证码中间页", errFanqieBrowserRequired)
		}
		return fanqieChapter{}, fmt.Errorf("official reader state not found")
	}
	var state map[string]any
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return fanqieChapter{}, err
	}
	reader, _ := state["reader"].(map[string]any)
	if reader == nil {
		return fanqieChapter{}, fmt.Errorf("official reader data not found")
	}
	data, _ := reader["chapterData"].(map[string]any)
	if data == nil {
		return fanqieChapter{}, fmt.Errorf("official chapter data not found")
	}
	rawContent := asString(data["content"])
	content := htmlFragmentToPlainText(rawContent)
	if content == "" {
		return fanqieChapter{}, fmt.Errorf("official chapter content is empty")
	}
	itemID := asString(data["itemId"])
	if itemID == "" {
		if match := fanqieReaderIDPattern.FindStringSubmatch(sourceURL); len(match) == 2 {
			itemID = match[1]
		}
	}
	status := "readable"
	if containsPrivateUseRune(content) {
		status = "font_encoded"
	}
	if isFanqieReaderPreview(data, rawContent, content) {
		status = "web_preview"
	}
	return fanqieChapter{
		ItemID:       itemID,
		Title:        asString(data["title"]),
		Content:      content,
		Readable:     true,
		SourceURL:    sourceURL,
		DecodeStatus: status,
	}, nil
}

func isFanqieReaderPreview(data map[string]any, rawContent, content string) bool {
	if !asBool(data["isChapterLock"]) && asInt(data["needPay"]) <= 0 {
		return false
	}
	expectedWords := asInt(data["chapterWordNumber"])
	if expectedWords <= 0 {
		return strings.HasSuffix(strings.TrimSpace(rawContent), "<p") || strings.HasSuffix(strings.TrimSpace(rawContent), "\u003Cp")
	}
	actualWords := countCJKWords(content)
	return actualWords > 0 && actualWords*3 < expectedWords
}

func isFanqieBrowserRequiredError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, errFanqieBrowserRequired) || strings.Contains(err.Error(), "验证码")
}

func mergeFanqieParsedChapter(fallback, parsed fanqieChapter) fanqieChapter {
	if parsed.Index == 0 {
		parsed.Index = fallback.Index
	}
	if parsed.ItemID == "" {
		parsed.ItemID = fallback.ItemID
	}
	if parsed.Title == "" {
		parsed.Title = fallback.Title
	}
	if parsed.SourceURL == "" {
		parsed.SourceURL = fallback.SourceURL
	}
	return parsed
}

func markFanqieBrowserRequired(detail fanqieBookDetail, chapters []fanqieChapter) fanqieBookDetail {
	const status = "正文未采集：番茄官方 reader 需要浏览器安全校验，后端直连无法获取正文"
	for idx := range chapters {
		if strings.TrimSpace(chapters[idx].Content) == "" {
			chapters[idx].DecodeStatus = status
		}
	}
	detail.Chapters = chapters
	detail.ReadableContent = false
	detail.FontEncoded = false
	detail.PartialContent = false
	detail.Blocked = true
	detail.Notes = append(detail.Notes,
		fmt.Sprintf("仍有 %d 章未采集到正文，已保留目录和状态。", len(chapters)),
		"官方 reader 需要浏览器安全校验，后端直连会返回验证码中间页；已停止批量请求，避免继续触发拦截。",
		"如需完整正文，请使用已登录且有权限的番茄会话或已授权 TXT 文本导入。",
	)
	return detail
}

func populateFanqieChapterContents(ctx context.Context, detail fanqieBookDetail, fetcher fanqieURLTextFetcher) fanqieBookDetail {
	if fetcher == nil || len(detail.Chapters) == 0 {
		return detail
	}
	chapters := make([]fanqieChapter, len(detail.Chapters))
	copy(chapters, detail.Chapters)
	preflightIndex := -1
	for idx, chapter := range chapters {
		if chapter.SourceURL != "" {
			preflightIndex = idx
			break
		}
	}
	if preflightIndex >= 0 {
		chapter := chapters[preflightIndex]
		readerHTML, err := fetcher(ctx, chapter.SourceURL)
		if err != nil {
			chapter.DecodeStatus = "正文采集失败：" + err.Error()
			chapters[preflightIndex] = chapter
		} else {
			parsed, parseErr := parseFanqieReaderHTML(readerHTML, chapter.SourceURL)
			if isFanqieBrowserRequiredError(parseErr) {
				return markFanqieBrowserRequired(detail, chapters)
			}
			if parseErr != nil {
				chapter.DecodeStatus = "正文解析失败：" + parseErr.Error()
				chapters[preflightIndex] = chapter
			} else {
				chapters[preflightIndex] = mergeFanqieParsedChapter(chapter, parsed)
			}
		}
	}
	type result struct {
		index   int
		chapter fanqieChapter
	}
	jobs := make(chan int)
	results := make(chan result, len(chapters))
	workerCount := 4
	if len(chapters) < workerCount {
		workerCount = len(chapters)
	}
	var wg sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				chapter := chapters[idx]
				if chapter.SourceURL == "" {
					results <- result{index: idx, chapter: chapter}
					continue
				}
				readerHTML, err := fetcher(ctx, chapter.SourceURL)
				if err != nil {
					chapter.DecodeStatus = "正文采集失败：" + err.Error()
					results <- result{index: idx, chapter: chapter}
					continue
				}
				parsed, err := parseFanqieReaderHTML(readerHTML, chapter.SourceURL)
				if err != nil {
					chapter.DecodeStatus = "正文解析失败：" + err.Error()
					results <- result{index: idx, chapter: chapter}
					continue
				}
				if parsed.Index == 0 {
					parsed.Index = chapter.Index
				}
				if parsed.ItemID == "" {
					parsed.ItemID = chapter.ItemID
				}
				if parsed.Title == "" {
					parsed.Title = chapter.Title
				}
				results <- result{index: idx, chapter: mergeFanqieParsedChapter(chapter, parsed)}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for idx := range chapters {
			if idx == preflightIndex {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case jobs <- idx:
			}
		}
	}()
	wg.Wait()
	close(results)
	for item := range results {
		chapters[item.index] = item.chapter
	}
	collected := 0
	fontEncoded := false
	partial := false
	blocked := false
	for _, chapter := range chapters {
		if strings.TrimSpace(chapter.Content) != "" {
			collected++
		}
		if chapter.DecodeStatus == "font_encoded" {
			fontEncoded = true
		}
		if chapter.DecodeStatus == "web_preview" {
			partial = true
		}
		if strings.Contains(chapter.DecodeStatus, "验证码") {
			blocked = true
		}
	}
	detail.Chapters = chapters
	detail.ReadableContent = collected > 0
	detail.FontEncoded = fontEncoded
	detail.PartialContent = partial
	detail.Blocked = blocked
	if collected > 0 {
		detail.Notes = append(detail.Notes, fmt.Sprintf("已采集 %d/%d 章正文。", collected, len(chapters)))
	}
	if collected < len(chapters) {
		detail.Notes = append(detail.Notes, fmt.Sprintf("仍有 %d 章未采集到正文，已保留目录和状态。", len(chapters)-collected))
	}
	if fontEncoded {
		detail.Notes = append(detail.Notes, "部分正文包含番茄网页字体编码；内容已采集，纯 TXT 可能显示为私有字符。")
	}
	if partial {
		detail.Notes = append(detail.Notes, "部分章节只返回网页预览段落，不是完整正文；需要使用已登录且有权限的番茄会话或授权文本源获取全文。")
	}
	if blocked {
		detail.Notes = append(detail.Notes, "官方 reader 需要浏览器安全校验，后端直连无法自动采集正文；请使用已登录且有权限的番茄会话或已授权文本导入。")
	}
	return detail
}

func fetchFanqieBookPageDetailFromOfficial(ctx context.Context, book fanqieBook) (fanqieBookDetail, error) {
	book = normalizeFanqieActionBook(book)
	bookID := fanqieBookID(book)
	if bookID == "" {
		return fanqieBookDetail{}, fmt.Errorf("缺少番茄作品 ID 或页面链接")
	}
	sourceURL := "https://fanqienovel.com/page/" + bookID
	pageHTML, err := fetchURLText(ctx, sourceURL)
	if err != nil {
		return fanqieBookDetail{}, err
	}
	detail, err := parseFanqiePageHTML(pageHTML, sourceURL, book)
	if err != nil {
		return fanqieBookDetail{}, err
	}
	return detail, nil
}

func limitFanqieDetailChapters(detail fanqieBookDetail, limit int) fanqieBookDetail {
	if limit <= 0 || len(detail.Chapters) <= limit {
		return detail
	}
	chapters := make([]fanqieChapter, limit)
	copy(chapters, detail.Chapters[:limit])
	detail.Chapters = chapters
	return detail
}

func fetchFanqieBookDetailFromOfficial(ctx context.Context, book fanqieBook) (fanqieBookDetail, error) {
	detail, err := fetchFanqieBookPageDetailFromOfficial(ctx, book)
	if err != nil {
		return fanqieBookDetail{}, err
	}
	return populateFanqieChapterContents(ctx, detail, fetchURLText), nil
}

func fetchFanqieBookAnalysisDetailFromOfficial(ctx context.Context, book fanqieBook) (fanqieBookDetail, error) {
	detail, err := fetchFanqieBookPageDetailFromOfficial(ctx, book)
	if err != nil {
		return fanqieBookDetail{}, err
	}
	detail = limitFanqieDetailChapters(detail, fanqieAnalysisChapterLimit)
	detail.Notes = append(detail.Notes, "分析仅采集首页简介和前 10 章内容，下载仍按整本处理。")
	return populateFanqieChapterContents(ctx, detail, fetchURLText), nil
}

func fallbackFanqieBookDetail(book fanqieBook, err error) fanqieBookDetail {
	book = normalizeFanqieActionBook(book)
	note := "官方页面暂时不可用，已基于当前榜单/搜索结果生成导入包。"
	if err != nil {
		note = note + " 原因：" + err.Error()
	}
	return fanqieBookDetail{
		Book:            book,
		Chapters:        []fanqieChapter{},
		ReadableContent: false,
		Source:          "fanqie-current-selection",
		Notes:           []string{note},
	}
}

func buildFanqieDownloadText(detail fanqieBookDetail) string {
	var b strings.Builder
	b.WriteString("【仅限个人备份 / 授权素材导入】\n")
	b.WriteString("书名：" + titleOr(detail.Book.Title, "未命名作品") + "\n")
	if detail.Book.Author != "" {
		b.WriteString("作者：" + detail.Book.Author + "\n")
	}
	if detail.Book.Category != "" || detail.Book.Status != "" || detail.Book.WordCount != "" {
		b.WriteString("信息：" + strings.Trim(strings.Join([]string{detail.Book.Category, detail.Book.Status, detail.Book.WordCount}, " / "), " /") + "\n")
	}
	if detail.Book.SourceURL != "" {
		b.WriteString("来源：" + detail.Book.SourceURL + "\n")
	}
	if detail.Book.Description != "" {
		b.WriteString("\n【首页简介】\n" + detail.Book.Description + "\n")
	}
	if len(detail.Chapters) > 0 {
		b.WriteString("\n【章节目录 / 已采集正文】\n")
	}
	for _, chapter := range detail.Chapters {
		b.WriteString(fmt.Sprintf("\n%s\n", titleOr(chapter.Title, fmt.Sprintf("第%d章", chapter.Index))))
		if strings.TrimSpace(chapter.Content) != "" {
			b.WriteString(strings.TrimSpace(chapter.Content) + "\n")
		} else if chapter.DecodeStatus != "" {
			b.WriteString("（" + chapter.DecodeStatus + "）\n")
		}
	}
	return b.String()
}

func buildFanqieDownloadResult(detail fanqieBookDetail) fanqieDownloadResult {
	chapters := make([]studioImportChapter, 0, len(detail.Chapters))
	for _, chapter := range detail.Chapters {
		chapters = append(chapters, studioImportChapter{
			Title:     titleOr(chapter.Title, fmt.Sprintf("第%d章", chapter.Index)),
			WordCount: countCJKWords(chapter.Content),
			Source:    chapter.SourceURL,
			Content:   strings.TrimSpace(chapter.Content),
		})
	}
	notes := append([]string{
		"仅限个人备份、本人作品或已授权素材导入，请遵守来源平台条款。",
	}, detail.Notes...)
	decodeStatus := "catalog_only"
	if detail.ReadableContent {
		decodeStatus = "readable"
		if detail.PartialContent {
			decodeStatus = "web_preview"
		} else if detail.FontEncoded {
			decodeStatus = "font_encoded"
		}
	} else if detail.Blocked {
		decodeStatus = "browser_required"
	}
	title := titleOr(detail.Book.Title, "番茄作品")
	return fanqieDownloadResult{
		Title:        title,
		FileName:     title + ".txt",
		Status:       "completed",
		ChapterCount: len(detail.Chapters),
		Text:         buildFanqieDownloadText(detail),
		Chapters:     chapters,
		Notes:        notes,
		Source:       detail.Source,
		DecodeStatus: decodeStatus,
	}
}

// DownloadFanqieBook 生成番茄作品个人备份/授权导入包，并写入「我的作品」归档。
func (h *StudioHandler) DownloadFanqieBook(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req fanqieBookActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !req.Consent {
		response.BadRequest(c, "请先确认该作品为本人作品、你有权备份或已获得授权，仅用于个人备份和学习分析")
		return
	}
	req.Book = normalizeFanqieActionBook(req.Book)
	if req.Book.Title == "" && req.Book.SourceURL == "" && req.Book.ID == "" {
		response.BadRequest(c, "请选择要下载/导入的番茄作品")
		return
	}

	detail, err := fetchFanqieBookDetail(c.Request.Context(), req.Book)
	if err != nil {
		detail = fallbackFanqieBookDetail(req.Book, err)
	}
	result := buildFanqieDownloadResult(detail)
	h.recordCreation(c.Request.Context(), subject.UserID, "import", titleOr(result.Title+"导入", "番茄导入"), req, result, "fanqie-importer")
	response.Success(c, result)
}

func buildFanqieAnalysisPrompt(detail fanqieBookDetail) (string, string) {
	system := "你是「烂番茄」网文开篇分析师。根据作品首页简介、元数据和前 10 章信息判断卖点与开篇问题。只输出一个 JSON 对象，不要解释，不要 markdown 代码块。"
	var b strings.Builder
	b.WriteString("请分析这本番茄小说的首页介绍和前 10 章。\n")
	b.WriteString("书名：" + titleOr(detail.Book.Title, "未知") + "\n")
	if detail.Book.Author != "" {
		b.WriteString("作者：" + detail.Book.Author + "\n")
	}
	if detail.Book.Category != "" || detail.Book.Status != "" || detail.Book.WordCount != "" {
		b.WriteString("元数据：" + strings.Trim(strings.Join([]string{detail.Book.Category, detail.Book.Status, detail.Book.WordCount}, " / "), " /") + "\n")
	}
	if detail.Book.Description != "" {
		b.WriteString("首页简介：" + detail.Book.Description + "\n")
	}
	b.WriteString("\n前 10 章：\n")
	limit := len(detail.Chapters)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		ch := detail.Chapters[i]
		b.WriteString(fmt.Sprintf("%02d. %s", i+1, titleOr(ch.Title, fmt.Sprintf("第%d章", ch.Index))))
		if strings.TrimSpace(ch.Content) != "" {
			content := strings.TrimSpace(ch.Content)
			runes := []rune(content)
			if len(runes) > 900 {
				content = string(runes[:900]) + "..."
			}
			b.WriteString("\n正文片段：" + content)
		} else if ch.DecodeStatus != "" {
			b.WriteString("\n正文状态：" + ch.DecodeStatus)
		}
		b.WriteString("\n")
	}
	if !detail.ReadableContent {
		b.WriteString("\n注意：如果正文因字体混淆不可读，请主要依据首页简介、章节标题、题材和元数据分析，并在结论里说明置信度。\n")
	}
	b.WriteString("\n严格按以下 JSON 输出：")
	b.WriteString(`{"title":"分析标题","summary":"综合判断","hooks":["开篇钩子"],"chapter_notes":[{"title":"章节名","note":"这一章的结构作用"}],"actions":["下一步建议"]}`)
	return system, b.String()
}

// AnalyzeFanqieBook 用配置好的文案模型分析番茄作品首页简介和前 10 章。
func (h *StudioHandler) AnalyzeFanqieBook(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	ak, model, ok := h.requireStudioTextModel(c, subject.UserID)
	if !ok {
		return
	}
	var req fanqieBookActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Book = normalizeFanqieActionBook(req.Book)
	if req.Book.Title == "" && req.Book.SourceURL == "" && req.Book.ID == "" {
		response.BadRequest(c, "请选择要分析的番茄作品")
		return
	}

	detail, err := fetchFanqieBookAnalysisDetail(c.Request.Context(), req.Book)
	if err != nil {
		detail = fallbackFanqieBookDetail(req.Book, err)
	}
	system, user := buildFanqieAnalysisPrompt(detail)
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
		response.ErrorFrom(c, fmt.Errorf("番茄开篇分析失败：%w", err))
		return
	}
	var report fanqieAnalysisReport
	if err := json.Unmarshal([]byte(content), &report); err != nil {
		response.ErrorFrom(c, fmt.Errorf("解析模型输出失败：%w", err))
		return
	}
	if report.Hooks == nil {
		report.Hooks = []string{}
	}
	if report.Actions == nil {
		report.Actions = []string{}
	}
	report.Source = detail.Source
	report.Notes = detail.Notes
	h.recordCreation(c.Request.Context(), subject.UserID, "hotspot", titleOr(report.Title, detail.Book.Title+"开篇分析"), req, report, model)
	response.Success(c, report)
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
	Content   string `json:"content,omitempty"`
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
		chapters = append(chapters, studioImportChapter{Title: title, WordCount: countCJKWords(text), Content: text})
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
