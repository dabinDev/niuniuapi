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

/** 拆书 & 爆款分析报告 */
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
 * 提交拆书 & 爆款分析任务。
 * @returns 结构化拆书报告
 */
export async function analyzeTeardown(payload: TeardownRequest): Promise<TeardownReport> {
  const { data } = await apiClient.post<TeardownReport>('/studio/teardown', payload)
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

/** 生成小说封面（多版候选）。 */
export async function generateCover(payload: CoverRequest): Promise<CoverResult> {
  const { data } = await apiClient.post<CoverResult>('/studio/cover', payload)
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

// ==================== 剧本生成 ====================

/** 剧本形态：长剧本 / 短剧 / 分镜脚本 */
export type ScriptForm = 'long' | 'short' | 'storyboard'

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

const MODEL_CONFIG_LS_KEY = 'studio_model_config'

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

export type WorkType = 'cover' | 'teardown' | 'script'

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
  listWorks,
  getWork,
  getImageModels,
  testImageModel,
  getImageConfig,
  saveImageConfig,
  getModelConfig,
  saveModelConfig,
  generateCover,
  generateScript,
}

export default studioAPI
