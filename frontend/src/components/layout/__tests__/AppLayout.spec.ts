import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'

import AppLayout from '../AppLayout.vue'
import { useAuthStore } from '@/stores/auth'

const routeState = vi.hoisted(() => ({ path: '/studio/teardown' }))
const onboardingCalls = vi.hoisted(() => [] as Array<Record<string, unknown>>)
const replayTour = vi.hoisted(() => vi.fn())

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => routeState
  }
})

vi.mock('@/composables/useOnboardingTour', () => ({
  useOnboardingTour: vi.fn((options: Record<string, unknown>) => {
    onboardingCalls.push(options)
    return { replayTour }
  })
}))

function mountLayout(path: string) {
  const pinia = createPinia()
  setActivePinia(pinia)
  routeState.path = path
  onboardingCalls.length = 0
  replayTour.mockClear()

  const authStore = useAuthStore()
  authStore.user = {
    id: 1,
    email: 'admin@example.com',
    username: 'Admin',
    role: 'admin'
  } as typeof authStore.user

  return mount(AppLayout, {
    global: {
      plugins: [pinia],
      stubs: {
        AppSidebar: { template: '<aside />' },
        AppHeader: { template: '<header />' }
      }
    },
    slots: {
      default: '<section data-testid="page-content">content</section>'
    }
  })
}

describe('AppLayout onboarding auto-start', () => {
  it('does not auto-start the onboarding tour on studio workspace pages', () => {
    const wrapper = mountLayout('/studio/teardown')
    const options = onboardingCalls[0]

    expect(options.storageKey).toBe('admin_guide')
    expect(options.autoStart).toBe(true)
    expect(options.shouldAutoStart).toEqual(expect.any(Function))
    expect((options.shouldAutoStart as () => boolean)()).toBe(false)

    wrapper.unmount()
  })

  it('keeps onboarding auto-start available outside the studio workspace', () => {
    const wrapper = mountLayout('/admin/dashboard')
    const options = onboardingCalls[0]

    expect(options.shouldAutoStart).toEqual(expect.any(Function))
    expect((options.shouldAutoStart as () => boolean)()).toBe(true)

    wrapper.unmount()
  })
})
