import { createMemoryHistory, createRouter, type RouteLocationNormalized, type NavigationGuardNext, type RouteRecordRaw } from 'vue-router'
import { describe, expect, it } from 'vitest'
import router from '@/router'
import { firstVisibleStudioPath, isStudioFeatureVisible, studioFeatureForPath } from '@/utils/studioFeatures'

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

describe('studio visibility route guard behavior', () => {
  function createGuardedRouter(options: {
    isAdmin?: boolean
    visibility?: Record<string, boolean>
  }) {
    const testRouter = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/dashboard', component: { template: '<div />' } },
        { path: '/studio', redirect: '/studio/fanqie' },
        { path: '/studio/fanqie', component: { template: '<div />' } },
        { path: '/studio/cover', component: { template: '<div />' } },
        { path: '/studio/works', component: { template: '<div />' } },
        { path: '/studio/teardown', component: { template: '<div />' } },
        { path: '/studio/hotspot', component: { template: '<div />' } },
        { path: '/studio/generate', component: { template: '<div />' } },
        { path: '/studio/downloader', redirect: '/studio/fanqie' },
        { path: '/studio/script', redirect: '/studio/generate' },
      ] as RouteRecordRaw[],
    })

    testRouter.beforeEach((to: RouteLocationNormalized, _from: RouteLocationNormalized, next: NavigationGuardNext) => {
      if (to.path === '/studio' || to.path.startsWith('/studio/')) {
        const fallbackPath = firstVisibleStudioPath(options.visibility, options.isAdmin === true)
        const featureId = studioFeatureForPath(to.path)

        if (to.path === '/studio') {
          next(fallbackPath)
          return
        }

        if (!options.isAdmin && featureId && !isStudioFeatureVisible(featureId, options.visibility, false)) {
          next(fallbackPath === to.path ? '/dashboard' : fallbackPath)
          return
        }
      }
      next()
    })

    return testRouter
  }

  it('redirects regular users away from disabled studio feature URLs', async () => {
    const testRouter = createGuardedRouter({
      visibility: { fanqie: false, cover: true, works: true, teardown: true, hotspot: true, generate: true },
    })

    await testRouter.push('/studio/fanqie')
    await testRouter.isReady()

    expect(testRouter.currentRoute.value.path).toBe('/studio/cover')
  })

  it('redirects regular users to dashboard when every studio feature is disabled', async () => {
    const testRouter = createGuardedRouter({
      visibility: { fanqie: false, cover: false, works: false, teardown: false, hotspot: false, generate: false },
    })

    await testRouter.push('/studio/cover')
    await testRouter.isReady()

    expect(testRouter.currentRoute.value.path).toBe('/dashboard')
  })

  it('allows admins to open studio feature URLs even when user visibility is disabled', async () => {
    const testRouter = createGuardedRouter({
      isAdmin: true,
      visibility: { fanqie: false, cover: false, works: false, teardown: false, hotspot: false, generate: false },
    })

    await testRouter.push('/studio/fanqie')
    await testRouter.isReady()

    expect(testRouter.currentRoute.value.path).toBe('/studio/fanqie')
  })
})
