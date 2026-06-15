import { describe, expect, it } from 'vitest'
import { buildNovelCoverPayload } from '@/api/studio'

describe('buildNovelCoverPayload', () => {
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
})
