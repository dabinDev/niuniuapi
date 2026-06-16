import { describe, expect, it } from 'vitest'
import router from '@/router'

describe('studio information architecture routes', () => {
  it('registers hotspot and generate as first-class studio routes', () => {
    const routes = router.getRoutes()

    const hotspot = routes.find((route) => route.path === '/studio/hotspot')
    expect(hotspot?.name).toBe('StudioHotspot')
    expect(hotspot?.meta.titleKey).toBe('nav.studioHotspot')

    const generate = routes.find((route) => route.path === '/studio/generate')
    expect(generate?.name).toBe('StudioGenerate')
    expect(generate?.meta.titleKey).toBe('nav.studioGenerate')
  })

  it('keeps the old script URL as a compatibility redirect', () => {
    const legacyScript = router.getRoutes().find((route) => route.path === '/studio/script')

    expect(legacyScript?.redirect).toBe('/studio/generate')
  })
})
