import type { StudioFeatureVisibility } from '@/types'

export type StudioFeatureId = 'fanqie' | 'cover' | 'works' | 'teardown' | 'hotspot' | 'generate'

export interface StudioFeatureDefinition {
  id: StudioFeatureId
  path: string
  navKey: string
  title: string
  description: string
}

export const STUDIO_FEATURES: StudioFeatureDefinition[] = [
  {
    id: 'fanqie',
    path: '/studio/fanqie',
    navKey: 'nav.studioFanqieHotlist',
    title: '番茄热榜',
    description: '热榜选样本、导入完整本小说、沉淀可复用素材。',
  },
  {
    id: 'cover',
    path: '/studio/cover',
    navKey: 'nav.studioCover',
    title: '封面生成',
    description: '根据题材、卖点和简介生成小说封面方向。',
  },
  {
    id: 'works',
    path: '/studio/works',
    navKey: 'nav.studioWorks',
    title: '我的作品',
    description: '归档封面、拆书、对标和生成记录。',
  },
  {
    id: 'teardown',
    path: '/studio/teardown',
    navKey: 'nav.studioTeardown',
    title: '拆书诊断',
    description: '拆人物、爽点、伏笔和章节结构。',
  },
  {
    id: 'hotspot',
    path: '/studio/hotspot',
    navKey: 'nav.studioHotspot',
    title: '爆款对标',
    description: '用同类样本反推题材、爽点和结构动作。',
  },
  {
    id: 'generate',
    path: '/studio/generate',
    navKey: 'nav.studioGenerate',
    title: '创作生成',
    description: '把素材转成正文方案、短剧脚本和口播素材。',
  },
]

export const DEFAULT_STUDIO_FEATURE_VISIBILITY: Record<StudioFeatureId, boolean> =
  STUDIO_FEATURES.reduce((acc, feature) => {
    acc[feature.id] = true
    return acc
  }, {} as Record<StudioFeatureId, boolean>)

const STUDIO_PATH_TO_FEATURE: Record<string, StudioFeatureId> = {
  '/studio/fanqie': 'fanqie',
  '/studio/downloader': 'fanqie',
  '/studio/cover': 'cover',
  '/studio/works': 'works',
  '/studio/teardown': 'teardown',
  '/studio/hotspot': 'hotspot',
  '/studio/generate': 'generate',
  '/studio/script': 'generate',
}

export function normalizeStudioFeatureVisibility(
  visibility?: Partial<StudioFeatureVisibility> | null,
): Record<StudioFeatureId, boolean> {
  const normalized = { ...DEFAULT_STUDIO_FEATURE_VISIBILITY }
  if (!visibility || typeof visibility !== 'object') {
    return normalized
  }
  for (const feature of STUDIO_FEATURES) {
    const value = visibility[feature.id]
    if (typeof value === 'boolean') {
      normalized[feature.id] = value
    }
  }
  return normalized
}

export function isStudioFeatureVisible(
  featureId: StudioFeatureId,
  visibility?: Partial<StudioFeatureVisibility> | null,
  isAdmin = false,
): boolean {
  if (isAdmin) {
    return true
  }
  return normalizeStudioFeatureVisibility(visibility)[featureId] !== false
}

export function visibleStudioFeatures(
  visibility?: Partial<StudioFeatureVisibility> | null,
  isAdmin = false,
): StudioFeatureDefinition[] {
  return STUDIO_FEATURES.filter((feature) => isStudioFeatureVisible(feature.id, visibility, isAdmin))
}

export function studioFeatureForPath(path: string): StudioFeatureId | null {
  const cleanPath = path.split(/[?#]/, 1)[0].replace(/\/+$/, '') || '/'
  for (const [basePath, featureId] of Object.entries(STUDIO_PATH_TO_FEATURE)) {
    if (cleanPath === basePath || cleanPath.startsWith(`${basePath}/`)) {
      return featureId
    }
  }
  return null
}

export function firstVisibleStudioPath(
  visibility?: Partial<StudioFeatureVisibility> | null,
  isAdmin = false,
): string {
  return visibleStudioFeatures(visibility, isAdmin)[0]?.path ?? '/dashboard'
}
