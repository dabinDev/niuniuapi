import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import HomeView from '../HomeView.vue'

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(() => ({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })),
})

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    isAuthenticated: true,
    isAdmin: false,
    checkAuth: vi.fn(),
  }),
  useAppStore: () => ({
    cachedPublicSettings: {},
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  }),
}))

describe('HomeView', () => {
  it('renders the rotten tomato critique homepage without the old Lingxi copy', () => {
    const wrapper = shallowMount(HomeView, {
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :data-to="to"><slot /></a>',
          },
          LocaleSwitcher: true,
          Icon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('烂番茄')
    expect(wrapper.text()).toContain('LANFANQIE · 毒舌创作质检台')
    expect(wrapper.text()).toContain('毒舌创作质检台')
    expect(wrapper.text()).toContain('专治三烂')
    expect(wrapper.text()).toMatch(/你的稿子\s*烂\s*在哪，\s*烂番茄一眼挑出来/)
    expect(wrapper.text()).toContain('爆款潜力分')
    expect(wrapper.text()).toContain('小说封面生成')
    expect(wrapper.text()).toContain('拆书诊断')
    expect(wrapper.text()).toContain('爆款对标')
    expect(wrapper.text()).toContain('番茄小说下载器')
    expect(wrapper.text()).toContain('创作生成')
    expect(wrapper.text()).toContain('Token 套餐')
    expect(wrapper.find('.tomato .t-leaf').exists()).toBe(true)
    expect(wrapper.find('.rot-stamp').text()).toContain('烂')
    expect(wrapper.find('[data-to="/studio/teardown"]').exists()).toBe(true)
    expect(wrapper.find('[data-to="/studio/cover"]').exists()).toBe(true)
    expect(wrapper.find('[data-to="/studio/hotspot"]').exists()).toBe(true)
    expect(wrapper.find('[data-to="/studio/generate"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('灵犀文创')
    expect(wrapper.text()).not.toContain('LINGXI CREATIVE')
  })
})
