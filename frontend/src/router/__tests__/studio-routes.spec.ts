import { describe, expect, it } from 'vitest'
import router from '@/router'

describe('studio information architecture routes', () => {
  it('opens the studio workbench on the fanqie hotlist by default', () => {
    const studioRoot = router.getRoutes().find((route) => route.path === '/studio')

    expect(studioRoot?.redirect).toBe('/studio/fanqie')
  })

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

  it('renames the legacy downloader route into the fanqie hotlist page', () => {
    const routes = router.getRoutes()

    const fanqie = routes.find((route) => route.path === '/studio/fanqie')
    expect(fanqie?.name).toBe('StudioFanqieHotlist')
    expect(fanqie?.meta.titleKey).toBe('nav.studioFanqieHotlist')

    const legacyDownloader = routes.find((route) => route.path === '/studio/downloader')
    expect(legacyDownloader?.redirect).toBe('/studio/fanqie')
  })
})
