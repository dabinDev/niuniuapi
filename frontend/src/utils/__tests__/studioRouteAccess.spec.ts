import { describe, expect, it } from 'vitest'

import { resolveStudioVisibilityRedirect } from '../studioRouteAccess'

describe('resolveStudioVisibilityRedirect', () => {
  it('redirects regular users away from a studio feature that was hidden after settings refresh', () => {
    const redirect = resolveStudioVisibilityRedirect(
      '/studio/fanqie',
      { fanqie: false, cover: true, works: false, teardown: false, hotspot: false, generate: false },
      false,
    )

    expect(redirect).toBe('/studio/cover')
  })

  it('sends regular users to dashboard when every studio feature is hidden', () => {
    const redirect = resolveStudioVisibilityRedirect(
      '/studio/cover',
      { fanqie: false, cover: false, works: false, teardown: false, hotspot: false, generate: false },
      false,
    )

    expect(redirect).toBe('/dashboard')
  })

  it('keeps admin users and non-studio routes in place', () => {
    const visibility = {
      fanqie: false,
      cover: false,
      works: false,
      teardown: false,
      hotspot: false,
      generate: false,
    }

    expect(resolveStudioVisibilityRedirect('/studio/fanqie', visibility, true)).toBeNull()
    expect(resolveStudioVisibilityRedirect('/keys', visibility, false)).toBeNull()
  })
})
