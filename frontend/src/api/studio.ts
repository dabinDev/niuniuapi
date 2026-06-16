/**
 * Studio (创作台) API endpoints.
 *
 * 创作功能统一走后端 /studio/* 端点，由后端转发到底层 AI 网关并按 token 计费。
 * 当前后端端点尚在开发中（需 Go 环境），前端先按约定接口对接，后端就绪即可联调。
 */

import { apiClient } from './client'

/** 毒舌点评尺度 */
export type TeardownTone = 'savage' | 'neutral' | 'gentle'

export interface TeardownRequest {
  /** 待拆解的小说正文 / 章节 */
  content: string
  /** 可选：书名 */
  title?: string
  /** 可选：题材标签 */
  genre?: string
  /** 点评尺度：毒舌 / 中性 / 鼓励 */
  tone: TeardownTone
}

/** 单项维度评分（0-100） */
export interface TeardownScore {
  label: string
  value: number
}

/** 拆书诊断报告 */
export interface TeardownReport {
  /** 综合评分 0-100 */
  overall_score: number
  /** 一句话诊断 */
  verdict: string
  /** 结构概览 */
  summary: string
  /** 各维度评分（节奏 / 爽点 / 人物 等） */
  scores: TeardownScore[]
  /** 亮点 */
  highlights: string[]
  /** 烂点（毒舌质检挑出的问题） */
  rotten_points: string[]
  /** 改进建议 */
  suggestions: string[]
}

/**
 * 提交拆书诊断任务。
 * @returns 结构化拆书报告
 */
export async function analyzeTeardown(payload: TeardownRequest): Promise<TeardownReport> {
  const { data } = await apiClient.post<TeardownReport>('/studio/teardown', payload)
  return data
}

// ==================== 爆款对标 ====================

export interface HotspotRequest {
  content: string
  benchmark?: string
  title?: string
  genre?: string
  goal?: 'new-book' | 'rewrite' | 'short-video'
}

export interface HotspotSample {
  title: string
  lesson: string
}

export interface HotspotReport {
  market_score: number
  verdict: string
  radar: TeardownScore[]
  tropes: string[]
  gaps: string[]
  actions: string[]
  samples: HotspotSample[]
}

export async function analyzeHotspot(payload: HotspotRequest): Promise<HotspotReport> {
  const { data } = await apiClient.post<HotspotReport>('/studio/hotspot', payload)
  return data
}

// ==================== 图像模型配置 ====================

export interface StudioImageConfig {
  api_key_id: number
  model: string
}

export interface ImageModelItem {
  model: string
}

/** 拉取某密钥可用的图像模型列表。 */
export async function getImageModels(apiKeyId: number): Promise<ImageModelItem[]> {
  const { data } = await apiClient.get<ImageModelItem[]>('/studio/image-models', {
    params: { api_key_id: apiKeyId },
  })
  return data
}

/** 测试某密钥+模型是否可生图（会生成一张最小测试图）。 */
export async function testImageModel(apiKeyId: number, model: string): Promise<{ ok: boolean }> {
  const { data } = await apiClient.post<{ ok: boolean }>('/studio/image/test', {
    api_key_id: apiKeyId,
    model,
  })
  return data
}

/** 读取已保存的生图配置（无则 null）。 */
export async function getImageConfig(): Promise<StudioImageConfig | null> {
  const { data } = await apiClient.get<StudioImageConfig | null>('/studio/image-config')
  return data
}

/** 保存生图配置。 */
export async function saveImageConfig(cfg: StudioImageConfig): Promise<void> {
  await apiClient.put('/studio/image-config', cfg)
}

// ==================== 封面生成 ====================

export interface CoverImage {
  id: string
  url: string
  prompt?: string
}

export interface CoverResult {
  covers: CoverImage[]
}

export type CoverJobStatus = 'running' | 'succeeded' | 'failed'

export interface CoverJob {
  job_id: string
  status: CoverJobStatus
  progress: number
  message?: string
  error?: string
  result?: CoverResult
  created_at?: string
  updated_at?: string
}

/** 自定义模式：用户提示词 + 可选参考图 */
export interface CoverCustomRequest {
  mode: 'custom'
  prompt: string
  ref_image?: string
  size: string
  count: number
}

/** 小说驱动模式：结构化字段 */
export interface CoverNovelRequest {
  mode: 'novel'
  title?: string
  synopsis: string
  protagonist: string
  genre?: string
  mood?: string
  key_scene?: string
  cover_title?: string
  size: string
  count: number
}

export type CoverRequest = CoverCustomRequest | CoverNovelRequest
const STUDIO_COVER_TIMEOUT_MS = 240_000

/** 生成小说封面（多版候选）。 */
export async function generateCover(payload: CoverRequest): Promise<CoverResult> {
  const { data } = await apiClient.post<CoverResult>('/studio/cover', payload, {
    timeout: STUDIO_COVER_TIMEOUT_MS,
  })
  return data
}

export async function startCoverJob(payload: CoverRequest): Promise<CoverJob> {
  const { data } = await apiClient.post<CoverJob>('/studio/cover/jobs', payload)
  return data
}

export async function getCoverJob(jobId: string): Promise<CoverJob> {
  const { data } = await apiClient.get<CoverJob>(`/studio/cover/jobs/${encodeURIComponent(jobId)}`)
  return data
}

/** 小说驱动模式的表单数据（前端），用 buildNovelCoverPayload 转成请求。 */
export interface NovelCoverForm {
  title?: string
  synopsis: string
  protagonist: string
  genre?: string
  mood?: string
  keyScene?: string
  coverTitle?: string
  size: string
  count: number
}

/** 把小说驱动表单拼成 novel-mode 请求。 */
export function buildNovelCoverPayload(f: NovelCoverForm): CoverNovelRequest {
  return {
    mode: 'novel',
    title: f.title || undefined,
    synopsis: f.synopsis.trim(),
    protagonist: f.protagonist.trim(),
    genre: f.genre || undefined,
    mood: f.mood || undefined,
    key_scene: f.keyScene || undefined,
    cover_title: f.coverTitle || undefined,
    size: f.size,
    count: f.count,
  }
}

// ==================== 创作生成 ====================

export type CreativeMode = 'outline' | 'draft' | 'rewrite' | 'long' | 'short' | 'storyboard'
export type ScriptForm = 'long' | 'short' | 'storyboard'

export interface CreativeRequest {
  content: string
  mode: CreativeMode
  brief?: string
  genre?: string
  style?: string
  target_words?: number
  episodes?: number
}

export interface CreativeSection {
  heading: string
  content: string
}

export interface CreativeResult {
  title: string
  mode: CreativeMode | string
  summary?: string
  sections: CreativeSection[]
  checklist?: string[]
  next_steps?: string[]
}

export async function generateCreative(payload: CreativeRequest): Promise<CreativeResult> {
  const { data } = await apiClient.post<CreativeResult>('/studio/generate', payload)
  return data
}

export interface ScriptRequest {
  /** 小说正文 / 大纲 */
  content: string
  /** 目标形态 */
  form: ScriptForm
  /** 可选：集数 / 时长约束 */
  episodes?: number
}

export interface ScriptScene {
  heading: string
  content: string
}

export interface ScriptResult {
  title: string
  form: ScriptForm
  scenes: ScriptScene[]
}

/** 把小说转成剧本 / 分镜脚本。 */
export async function generateScript(payload: ScriptRequest): Promise<ScriptResult> {
  const { data } = await apiClient.post<ScriptResult>('/studio/script', payload)
  return data
}

// ==================== 番茄导入器 ====================

export type StudioImportSource = 'manual' | 'fanqie'

export interface StudioImportRequest {
  source: StudioImportSource
  title?: string
  content?: string
  urls?: string[]
  consent: boolean
}

export interface StudioImportChapter {
  title: string
  word_count: number
  source?: string
}

export interface StudioImportResult {
  title: string
  source: string
  status: 'completed' | 'queued' | string
  chapters: StudioImportChapter[]
  notes: string[]
  next_actions: string[]
}

export async function importStudioContent(payload: StudioImportRequest): Promise<StudioImportResult> {
  const { data } = await apiClient.post<StudioImportResult>('/studio/import', payload)
  return data
}

// ==================== 创作模型配置（生图 / 文案）====================

export interface ModelSlot {
  api_key_id: number
  model: string
}

/** 用户在密钥管理页配置的创作模型：哪把密钥的哪个模型用于生图 / 文案。 */
export interface StudioModelConfig {
  image?: ModelSlot
  text?: ModelSlot
}

export type ModelSlotType = 'image' | 'text'

const MODEL_CONFIG_LS_KEY = 'studio_model_config'
const STUDIO_MODEL_TEST_TIMEOUT_MS = 240_000

/** 仅读本地缓存的模型配置。 */
export function loadLocalModelConfig(): StudioModelConfig {
  try {
    return JSON.parse(localStorage.getItem(MODEL_CONFIG_LS_KEY) || '{}') as StudioModelConfig
  } catch {
    return {}
  }
}

/** 读取模型配置：优先后端，回退本地缓存。 */
export async function getModelConfig(): Promise<StudioModelConfig> {
  try {
    const { data } = await apiClient.get<StudioModelConfig | null>('/studio/model-config')
    if (data && (data.image || data.text)) return data
  } catch {
    /* 后端未就绪：回退本地 */
  }
  return loadLocalModelConfig()
}

/** 保存模型配置：同时写本地缓存 + 后端（后端未就绪则仅本地）。 */
export async function saveModelConfig(cfg: StudioModelConfig): Promise<void> {
  localStorage.setItem(MODEL_CONFIG_LS_KEY, JSON.stringify(cfg))
  try {
    await apiClient.put('/studio/model-config', cfg)
  } catch {
    /* 后端未就绪：仅本地缓存 */
  }
}

// ==================== 我的作品 ====================

/** Fetch live upstream model IDs available to a selected user API key. */
export async function getKeyModels(apiKeyId: number): Promise<string[]> {
  const { data } = await apiClient.get<string[]>(`/studio/keys/${apiKeyId}/models`)
  return Array.isArray(data) ? data : []
}

export async function testModelSlot(type: ModelSlotType, apiKeyId: number, model: string): Promise<{ ok: boolean }> {
  const { data } = await apiClient.post<{ ok: boolean }>('/studio/model-test', {
    type,
    api_key_id: apiKeyId,
    model,
  }, {
    timeout: STUDIO_MODEL_TEST_TIMEOUT_MS,
  })
  return data
}

export type WorkType = 'cover' | 'teardown' | 'hotspot' | 'generate' | 'script' | 'import'

export interface WorkItem {
  id: number
  type: WorkType | string
  title: string
  model: string
  created_at: string
}

export interface WorkDetail extends WorkItem {
  output?: unknown
  input?: unknown
}

/** 列出当前用户的创作产出（可按类型过滤）。 */
export async function listWorks(type?: string): Promise<WorkItem[]> {
  const { data } = await apiClient.get<{ works: WorkItem[] }>('/studio/works', {
    params: type ? { type } : {},
  })
  return data.works || []
}

/** 获取单条创作产出（含完整结果）。 */
export async function getWork(id: number): Promise<WorkDetail> {
  const { data } = await apiClient.get<WorkDetail>(`/studio/works/${id}`)
  return data
}

export const studioAPI = {
  analyzeTeardown,
  analyzeHotspot,
  listWorks,
  getWork,
  getImageModels,
  testImageModel,
  getImageConfig,
  saveImageConfig,
  getModelConfig,
  saveModelConfig,
  testModelSlot,
  generateCover,
  startCoverJob,
  getCoverJob,
  generateCreative,
  generateScript,
  importStudioContent,
}

export default studioAPI
