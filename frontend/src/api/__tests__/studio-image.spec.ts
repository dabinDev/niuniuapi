import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiClientPost = vi.hoisted(() => vi.fn())
const apiClientGet = vi.hoisted(() => vi.fn())

vi.mock('@/api/client', () => ({
  apiClient: {
    get: apiClientGet,
    post: apiClientPost,
    put: vi.fn(),
  },
}))

import {
  buildNovelCoverPayload,
  generateCover,
  getCoverJob,
  analyzeFanqieBook,
  downloadFanqieBook,
  getFanqieRank,
  searchFanqieBooks,
  startCoverJob,
  testModelSlot,
} from '@/api/studio'

describe('buildNovelCoverPayload', () => {
  beforeEach(() => {
    apiClientGet.mockReset()
    apiClientPost.mockReset()
  })

  it('maps structured novel fields into a novel-mode cover request', () => {
    const p = buildNovelCoverPayload({
      title: '北境书塔',
      synopsis: '每本书都是一座城',
      protagonist: '林见微，外冷内热',
      genre: '玄幻',
      mood: '清冷',
      keyScene: '雨夜书塔',
      coverTitle: '北境书塔',
      size: '1024x1536',
      count: 2,
    })
    expect(p).toMatchObject({
      mode: 'novel',
      synopsis: '每本书都是一座城',
      protagonist: '林见微，外冷内热',
      genre: '玄幻',
      mood: '清冷',
      key_scene: '雨夜书塔',
      cover_title: '北境书塔',
      size: '1024x1536',
      count: 2,
    })
  })

  it('omits empty optional fields', () => {
    const p = buildNovelCoverPayload({ synopsis: 's', protagonist: 'p', size: '1024x1024', count: 1 })
    expect(p.genre).toBeUndefined()
    expect(p.key_scene).toBeUndefined()
    expect(p.cover_title).toBeUndefined()
  })

  it('uses a long timeout for cover image generation jobs', async () => {
    const payload = { mode: 'custom' as const, prompt: 'p', size: '1024x1024', count: 1 }
    apiClientPost.mockResolvedValue({ data: { covers: [] } })

    await generateCover(payload)

    expect(apiClientPost).toHaveBeenCalledWith('/studio/cover', payload, expect.objectContaining({ timeout: 240000 }))
  })

  it('uses a long timeout for image model slot tests', async () => {
    apiClientPost.mockResolvedValue({ data: { ok: true } })

    await testModelSlot('image', 1, 'gpt-image-2')

    expect(apiClientPost).toHaveBeenCalledWith(
      '/studio/model-test',
      { type: 'image', api_key_id: 1, model: 'gpt-image-2' },
      expect.objectContaining({ timeout: 240000 }),
    )
  })

  it('starts and polls cover generation jobs', async () => {
    const payload = { mode: 'custom' as const, prompt: 'p', size: '1024x1024', count: 1 }
    apiClientPost.mockResolvedValue({ data: { job_id: 'job-1', status: 'running', progress: 20 } })
    apiClientGet.mockResolvedValue({ data: { job_id: 'job-1', status: 'succeeded', progress: 100, result: { covers: [] } } })

    await expect(startCoverJob(payload)).resolves.toMatchObject({ job_id: 'job-1', status: 'running' })
    await expect(getCoverJob('job-1')).resolves.toMatchObject({ status: 'succeeded' })

    expect(apiClientPost).toHaveBeenCalledWith('/studio/cover/jobs', payload)
    expect(apiClientGet).toHaveBeenCalledWith('/studio/cover/jobs/job-1')
  })

  it('fetches fanqie rank channels through backend studio endpoints', async () => {
    apiClientGet.mockResolvedValue({ data: { channel: 'hot', updated_at: '2026-06-17T00:00:00Z', books: [] } })

    await expect(getFanqieRank('hot')).resolves.toMatchObject({ channel: 'hot', books: [] })

    expect(apiClientGet).toHaveBeenCalledWith('/studio/fanqie/rank', {
      params: { channel: 'hot' },
    })
  })

  it('searches fanqie books through the backend proxy', async () => {
    apiClientGet.mockResolvedValue({ data: { books: [{ id: '1', title: '十日终焉' }] } })

    await expect(searchFanqieBooks('十日终焉')).resolves.toEqual([{ id: '1', title: '十日终焉' }])

    expect(apiClientGet).toHaveBeenCalledWith('/studio/fanqie/search', {
      params: { q: '十日终焉' },
    })
  })

  it('downloads a fanqie book through the backend with consent', async () => {
    const book = { id: '1', title: '十日终焉', source_url: 'https://fanqienovel.com/page/1' } as never
    apiClientPost.mockResolvedValue({ data: { title: '十日终焉', file_name: '十日终焉.txt' } })

    await expect(downloadFanqieBook(book, true)).resolves.toMatchObject({ file_name: '十日终焉.txt' })

    expect(apiClientPost).toHaveBeenCalledWith(
      '/studio/fanqie/download',
      {
        book,
        consent: true,
      },
      expect.objectContaining({ timeout: 600000 }),
    )
  })

  it('requests fanqie first-ten-chapter analysis through the backend', async () => {
    const book = { id: '1', title: '十日终焉', source_url: 'https://fanqienovel.com/page/1' } as never
    apiClientPost.mockResolvedValue({ data: { title: '十日终焉开篇分析', hooks: ['空屋'] } })

    await expect(analyzeFanqieBook(book)).resolves.toMatchObject({ title: '十日终焉开篇分析' })

    expect(apiClientPost).toHaveBeenCalledWith('/studio/fanqie/analyze', { book })
  })
})
