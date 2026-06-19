import type { FanqieBook } from '@/api/studio'

export type StudioBridgeTarget = 'teardown' | 'hotspot' | 'generate'

export interface StudioBridgePayload {
  source: 'fanqie' | 'works'
  target: StudioBridgeTarget
  title: string
  genre: string
  content: string
  benchmark: string
  brief: string
}

export const STUDIO_BRIDGE_STORAGE_KEY = 'studio_bridge_payload'

export function buildFanqieBridgePayload(book: FanqieBook, benchmark: string, target: StudioBridgeTarget): StudioBridgePayload {
  const tags = book.tags?.length ? book.tags.join('、') : '未标注'
  const content = [
    `热榜样本：${book.title}`,
    `作者：${book.author}`,
    `题材：${book.category}`,
    `状态/字数：${book.status} / ${book.word_count}`,
    `榜单位置：#${book.rank}，${book.score}`,
    `简介：${book.description}`,
    `标签：${tags}`,
  ].join('\n')

  return {
    source: 'fanqie',
    target,
    title: book.title,
    genre: book.category,
    content,
    benchmark,
    brief: `参考《${book.title}》的题材卖点和开篇钩子，生成可执行的创作调整方案。`,
  }
}

export function saveStudioBridgePayload(payload: StudioBridgePayload) {
  localStorage.setItem(STUDIO_BRIDGE_STORAGE_KEY, JSON.stringify(payload))
}

export function consumeStudioBridgePayload(target: StudioBridgeTarget): StudioBridgePayload | null {
  try {
    const raw = localStorage.getItem(STUDIO_BRIDGE_STORAGE_KEY)
    if (!raw) return null
    const payload = JSON.parse(raw) as StudioBridgePayload
    if (!['fanqie', 'works'].includes(payload?.source) || payload.target !== target) return null
    localStorage.removeItem(STUDIO_BRIDGE_STORAGE_KEY)
    return payload
  } catch {
    localStorage.removeItem(STUDIO_BRIDGE_STORAGE_KEY)
    return null
  }
}
