import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => ({
        'announcements.title': '公告',
        'announcements.empty': '暂无公告',
        'announcements.emptyDescription': '暂时没有任何系统公告',
        'common.close': '关闭',
      }[key] ?? key),
    }),
  }
})

vi.mock('@/stores/announcements', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useAnnouncementStore: () => ({
      announcements: ref([]),
      loading: ref(false),
      unreadCount: 0,
      currentPopup: ref(null),
      markAsRead: vi.fn(),
      markAllAsRead: vi.fn(),
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

describe('AnnouncementBell tutorial entry', () => {
  it('shows the CCSWITCH and Codex tutorial entry before announcement items', async () => {
    setActivePinia(createPinia())
    const { default: AnnouncementBell } = await import('../AnnouncementBell.vue')
    const wrapper = mount(AnnouncementBell, {
      global: {
        stubs: {
          Icon: {
            template: '<span />',
          },
          RouterLink: {
            props: ['to'],
            template: '<a :href="to"><slot /></a>',
          },
          Teleport: true,
          Transition: false,
        },
      },
    })

    await wrapper.find('button[aria-label="公告"]').trigger('click')

    const text = wrapper.text()
    expect(text).toContain('CCSWITCH / Codex 接入教程')
    expect(text).toContain('先看教程，再处理公告')
    expect(text.indexOf('CCSWITCH / Codex 接入教程')).toBeLessThan(text.indexOf('暂无公告'))
    expect(wrapper.find('a[href="/tutorials/ccswitch-codex"]').exists()).toBe(true)
  })
})
