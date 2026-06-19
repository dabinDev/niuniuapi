import { describe, expect, it, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAdminComplianceStore } from '@/stores/adminCompliance'

vi.mock('@/api/admin/compliance', () => ({
  default: {
    getStatus: vi.fn(),
    accept: vi.fn()
  }
}))

vi.mock('@/i18n', () => ({
  getLocale: () => 'zh'
}))

describe('useAdminComplianceStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('uses the maintained niuniuapi compliance channel when backend metadata is missing', () => {
    const store = useAdminComplianceStore()

    store.requireAcknowledgement()

    expect(store.status?.document_url_zh).toBe(
      'https://github.com/dabinDev/niuniuapi/blob/writer-workbench-v0/docs/legal/admin-compliance.zh.md'
    )
    expect(store.status?.document_url_en).toBe(
      'https://github.com/dabinDev/niuniuapi/blob/writer-workbench-v0/docs/legal/admin-compliance.en.md'
    )
    expect(store.status?.document_url_zh).not.toContain('Wei-Shaw/sub2api')
    expect(store.expectedPhrase).toContain('烂番茄')
    expect(store.expectedPhrase).not.toContain('Sub2API')
    expect(store.expectedPhrase).not.toContain('�')
  })
})
