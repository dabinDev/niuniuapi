import { describe, expect, it } from 'vitest'

import {
  DEFAULT_STUDIO_FEATURE_VISIBILITY,
  firstVisibleStudioPath,
  isStudioFeatureVisible,
  studioFeatureForPath,
  visibleStudioFeatures,
} from '../studioFeatures'

describe('studio feature visibility', () => {
  it('defaults every built-in studio feature to visible', () => {
    expect(DEFAULT_STUDIO_FEATURE_VISIBILITY).toEqual({
      fanqie: true,
      cover: true,
      works: true,
      teardown: true,
      hotspot: true,
      generate: true,
    })
  })

  it('hides disabled features for regular users while keeping admins unrestricted', () => {
    const visibility = {
      fanqie: false,
      cover: true,
      works: true,
      teardown: false,
      hotspot: true,
      generate: true,
    }

    expect(isStudioFeatureVisible('fanqie', visibility, false)).toBe(false)
    expect(isStudioFeatureVisible('teardown', visibility, false)).toBe(false)
    expect(isStudioFeatureVisible('fanqie', visibility, true)).toBe(true)
    expect(visibleStudioFeatures(visibility, false).map((feature) => feature.id)).toEqual([
      'cover',
      'works',
      'hotspot',
      'generate',
    ])
  })

  it('maps legacy paths and resolves a fallback target from visible features', () => {
    expect(studioFeatureForPath('/studio/downloader')).toBe('fanqie')
    expect(studioFeatureForPath('/studio/script')).toBe('generate')
    expect(studioFeatureForPath('/studio/generate?from=old')).toBe('generate')
    expect(firstVisibleStudioPath({ fanqie: false, cover: false, works: true }, false)).toBe('/studio/works')
    expect(firstVisibleStudioPath({
      fanqie: false,
      cover: false,
      works: false,
      teardown: false,
      hotspot: false,
      generate: false,
    }, false)).toBe('/dashboard')
    expect(firstVisibleStudioPath({}, true)).toBe('/studio/fanqie')
  })
})
