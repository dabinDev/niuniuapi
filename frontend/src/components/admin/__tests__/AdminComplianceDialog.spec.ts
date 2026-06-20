import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { reactive } from 'vue'
import AdminComplianceDialog from '../AdminComplianceDialog.vue'
import { useAdminComplianceStore, useAuthStore } from '@/stores'

const routeState = reactive<{ path: string; meta: Record<string, unknown> }>({
  path: '/admin/accounts',
  meta: { requiresAdmin: true },
})

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/i18n', () => ({
  getLocale: () => 'zh',
}))

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    name: 'BaseDialog',
    props: ['show', 'title'],
    template: '<section v-if="show" data-testid="compliance-dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>',
  },
}))

vi.mock('@/components/common/Input.vue', () => ({
  default: {
    name: 'Input',
    props: ['modelValue'],
    emits: ['update:modelValue', 'enter'],
    template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    template: '<span />',
  },
}))

vi.mock('@/api/admin/compliance', () => ({
  default: {
    getStatus: vi.fn(),
    accept: vi.fn(),
  },
}))

function mountDialog(path: string) {
  routeState.path = path
  routeState.meta = path.startsWith('/admin') ? { requiresAdmin: true } : { requiresAdmin: false }
  setActivePinia(createPinia())

  const authStore = useAuthStore()
  authStore.token = 'token'
  authStore.user = {
    id: 1,
    email: 'admin@example.com',
    username: 'admin',
    role: 'admin',
    status: 'active',
    balance: 0,
    concurrency: 0,
    created_at: '',
    updated_at: '',
  } as never

  const complianceStore = useAdminComplianceStore()
  complianceStore.requireAcknowledgement({
    ack_phrase_zh: '我已阅读、理解并同意番茄部署与运营合规承诺',
  })

  return mount(AdminComplianceDialog)
}

describe('AdminComplianceDialog route scope', () => {
  beforeEach(() => {
    routeState.path = '/admin/accounts'
  })

  it('blocks administrator management pages until acknowledgement is completed', () => {
    const wrapper = mountDialog('/admin/accounts')

    expect(wrapper.find('[data-testid="compliance-dialog"]').exists()).toBe(true)
  })

  it('does not cover creator studio pages for an administrator session', () => {
    const wrapper = mountDialog('/studio/fanqie')

    expect(wrapper.find('[data-testid="compliance-dialog"]').exists()).toBe(false)
  })
})
