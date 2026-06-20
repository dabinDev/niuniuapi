import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import VersionBadge from '@/components/common/VersionBadge.vue'
import { useAppStore, useAuthStore } from '@/stores'

const checkUpdatesMock = vi.hoisted(() => vi.fn())

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/admin/system', () => ({
  performUpdate: vi.fn(),
  restartService: vi.fn(),
  checkUpdates: checkUpdatesMock,
  default: {
    performUpdate: vi.fn(),
    restartService: vi.fn(),
    checkUpdates: checkUpdatesMock,
  },
}))

describe('VersionBadge', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    checkUpdatesMock.mockReset()
    checkUpdatesMock.mockResolvedValue({
      current_version: '1.0.0',
      latest_version: '1.0.0',
      has_update: false,
      cached: false,
      build_type: 'source',
    })
    const authStore = useAuthStore()
    authStore.user = { id: 1, email: 'admin@example.com', role: 'admin', status: 'active' } as never
  })

  it('does not render upstream release links in the update dropdown', async () => {
    const appStore = useAppStore()
    appStore.currentVersion = '1.0.0'
    appStore.latestVersion = '1.1.0'
    appStore.hasUpdate = true
    appStore.buildType = 'source'
    appStore.versionLoaded = true
    appStore.releaseInfo = {
      name: 'v1.1.0',
      body: '',
      published_at: '2026-06-19T00:00:00Z',
      html_url: 'https://github.com/Wei-Shaw/sub2api/releases/tag/v1.1.0',
    }

    const wrapper = mount(VersionBadge, { props: { version: '1.0.0' } })
    await wrapper.get('button').trigger('click')

    expect(wrapper.text()).toContain('番茄维护渠道')
    expect(wrapper.text()).toContain('dabinDev/niuniuapi')
    expect(wrapper.text()).toContain('--no-build')
    expect(wrapper.find('a[href*="Wei-Shaw/sub2api"]').exists()).toBe(false)
    expect(wrapper.find('a[href*="dabinDev/niuniuapi"]').exists()).toBe(false)
  })

  it('keeps our own release link available', async () => {
    const appStore = useAppStore()
    appStore.currentVersion = '1.0.0'
    appStore.latestVersion = '1.1.0'
    appStore.hasUpdate = true
    appStore.buildType = 'source'
    appStore.versionLoaded = true
    appStore.releaseInfo = {
      name: 'v1.1.0',
      body: '',
      published_at: '2026-06-19T00:00:00Z',
      html_url: 'https://github.com/dabinDev/niuniuapi/releases/tag/v1.1.0',
    }

    const wrapper = mount(VersionBadge, { props: { version: '1.0.0' } })
    await wrapper.get('button').trigger('click')

    expect(wrapper.find('a[href*="dabinDev/niuniuapi"]').exists()).toBe(true)
  })

  it('does not offer in-place server updates for docker release builds', async () => {
    const appStore = useAppStore()
    appStore.currentVersion = '1.0.0'
    appStore.latestVersion = '1.1.0'
    appStore.hasUpdate = true
    appStore.buildType = 'release'
    appStore.versionLoaded = true
    appStore.releaseInfo = {
      name: 'v1.1.0',
      body: '',
      published_at: '2026-06-19T00:00:00Z',
      html_url: 'https://github.com/dabinDev/niuniuapi/releases/tag/v1.1.0',
    }

    const wrapper = mount(VersionBadge, { props: { version: '1.0.0' } })
    await wrapper.get('button').trigger('click')

    expect(wrapper.text()).toContain('本地 Docker')
    expect(wrapper.text()).toContain('--no-build')
    expect(wrapper.text()).not.toContain('version.updateNow')
    expect(wrapper.find('button[data-test="perform-version-update"]').exists()).toBe(false)
    expect(wrapper.find('a[href*="dabinDev/niuniuapi"]').exists()).toBe(true)
  })
})
