import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { adminAPI } from '@/api'

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getSettings: vi.fn(),
    },
    payment: {
      getConfig: vi.fn(),
    },
  },
}))

describe('useAdminSettingsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('treats admin compliance acknowledgement as a gated state, not a console error', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(adminAPI.settings.getSettings).mockRejectedValue({
      status: 423,
      code: 'ADMIN_COMPLIANCE_ACK_REQUIRED',
      message: 'administrator compliance acknowledgement is required',
    })
    vi.mocked(adminAPI.payment.getConfig).mockResolvedValue({ data: { enabled: true } })

    const store = useAdminSettingsStore()
    await store.fetch(true)

    expect(store.loaded).toBe(true)
    expect(consoleError).not.toHaveBeenCalled()
    consoleError.mockRestore()
  })
})
