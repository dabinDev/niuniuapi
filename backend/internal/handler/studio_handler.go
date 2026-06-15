package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/creationtask"
	"github.com/Wei-Shaw/sub2api/ent/studiomodelconfig"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
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
	client *dbent.Client
}

// NewStudioHandler 构造 StudioHandler。
func NewStudioHandler(client *dbent.Client) *StudioHandler {
	return &StudioHandler{client: client}
}

type studioModelSlot struct {
	APIKeyID int64  `json:"api_key_id"`
	Model    string `json:"model"`
}

type studioModelConfigDTO struct {
	Image *studioModelSlot `json:"image,omitempty"`
	Text  *studioModelSlot `json:"text,omitempty"`
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

func buildCoverPrompt(req coverRequest) string {
	if req.Mode == "custom" {
		p := strings.TrimSpace(req.Prompt)
		if req.RefImage != "" {
			p += "\n参考图：" + req.RefImage
		}
		return p
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
	count := req.Count
	if count < 1 {
		count = 1
	}
	if count > 4 {
		count = 4
	}
	size := req.Size
	if size == "" {
		size = "1024x1536"
	}

	payload, _ := json.Marshal(map[string]any{
		"model":  cfg.Image.Model,
		"prompt": prompt,
		"n":      count,
		"size":   size,
	})

	// 自调网关需指向容器内部监听端口（SERVER_PORT，默认 8080），
	// 不能用 c.Request.Host（那是外部映射端口，容器内不可达）。
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	gwURL := "http://127.0.0.1:" + port + "/v1/images/generations"
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
	covers := make([]gin.H, 0, len(gres.Data))
	for i, d := range gres.Data {
		u := d.URL
		if u == "" && d.B64 != "" {
			u = "data:image/png;base64," + d.B64
		}
		if u == "" {
			continue
		}
		covers = append(covers, gin.H{"id": fmt.Sprintf("cover-%d", i), "url": u})
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
