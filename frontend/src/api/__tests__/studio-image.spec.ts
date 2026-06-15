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

import { buildNovelCoverPayload, generateCover, getCoverJob, startCoverJob } from '@/api/studio'

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

  it('starts and polls cover generation jobs', async () => {
    const payload = { mode: 'custom' as const, prompt: 'p', size: '1024x1024', count: 1 }
    apiClientPost.mockResolvedValue({ data: { job_id: 'job-1', status: 'running', progress: 20 } })
    apiClientGet.mockResolvedValue({ data: { job_id: 'job-1', status: 'succeeded', progress: 100, result: { covers: [] } } })

    await expect(startCoverJob(payload)).resolves.toMatchObject({ job_id: 'job-1', status: 'running' })
    await expect(getCoverJob('job-1')).resolves.toMatchObject({ status: 'succeeded' })

    expect(apiClientPost).toHaveBeenCalledWith('/studio/cover/jobs', payload)
    expect(apiClientGet).toHaveBeenCalledWith('/studio/cover/jobs/job-1')
  })
})
