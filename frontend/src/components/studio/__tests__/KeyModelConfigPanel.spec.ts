import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const api = vi.hoisted(() => ({
  list: vi.fn(),
  gwListModels: vi.fn(),
  gwTestImage: vi.fn(),
  gwTestChat: vi.fn(),
  getKeyModels: vi.fn(),
  getModelConfig: vi.fn(),
  saveModelConfig: vi.fn(),
  testModelSlot: vi.fn(),
}))

vi.mock('@/api/keys', () => ({ list: api.list }))
vi.mock('@/api/gateway', () => ({
  gwListModels: api.gwListModels,
  gwTestImage: api.gwTestImage,
  gwTestChat: api.gwTestChat,
}))
vi.mock('@/api/studio', () => ({
  getKeyModels: api.getKeyModels,
  getModelConfig: api.getModelConfig,
  saveModelConfig: api.saveModelConfig,
  testModelSlot: api.testModelSlot,
}))

import KeyModelConfigPanel from '../KeyModelConfigPanel.vue'

describe('KeyModelConfigPanel', () => {
  beforeEach(() => {
    Object.values(api).forEach((f) => f.mockReset())
    api.list.mockResolvedValue({ items: [{ id: 1, name: 'GPT', key: 'sk-x' }], total: 1, page: 1, page_size: 100, pages: 1 })
    api.getModelConfig.mockResolvedValue({})
  })

  it('fetches the model list for the selected key', async () => {
    api.getKeyModels.mockResolvedValue(['gpt-image-2', 'gpt-5.4'])
    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    await w.find('[data-test=key]').setValue('1')
    await w.find('[data-test=fetch]').trigger('click')
    await flushPromises()

    expect(api.getKeyModels).toHaveBeenCalledWith(1)
    expect(api.gwListModels).not.toHaveBeenCalled()
    expect(w.findAll('[data-test=model] option')).toHaveLength(3) // placeholder + 2
  })

  it('assigns an image model, requires test before save, then persists', async () => {
    api.getKeyModels.mockResolvedValue(['gpt-image-1', 'gpt-5.4'])
    api.testModelSlot.mockResolvedValue({ ok: true })
    api.saveModelConfig.mockResolvedValue(undefined)
    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    await w.find('[data-test=key]').setValue('1')
    await w.find('[data-test=fetch]').trigger('click')
    await flushPromises()
    await w.find('[data-test=model]').setValue('gpt-image-1')
    await w.find('[data-test=set-image]').trigger('click')

    expect(w.find('[data-test=slot-image]').text()).toContain('gpt-image-1')
    expect(w.find('[data-test=save]').attributes('disabled')).toBeDefined()

    await w.find('[data-test=test-image]').trigger('click')
    await flushPromises()
    expect(api.testModelSlot).toHaveBeenCalledWith('image', 1, 'gpt-image-1')
    expect(api.gwTestImage).not.toHaveBeenCalled()
    expect(w.find('[data-test=save]').attributes('disabled')).toBeUndefined()

    await w.find('[data-test=save]').trigger('click')
    await flushPromises()
    expect(api.saveModelConfig).toHaveBeenCalledWith({ image: { api_key_id: 1, model: 'gpt-image-1' } })
  })

  it('preloads a saved config', async () => {
    api.getModelConfig.mockResolvedValue({ image: { api_key_id: 1, model: 'gpt-image-1' } })
    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    expect(w.find('[data-test=slot-image]').text()).toContain('gpt-image-1')
    expect(w.find('[data-test=save]').attributes('disabled')).toBeUndefined()
  })

  it('shows backend model test failure details', async () => {
    api.getKeyModels.mockResolvedValue(['gpt-image-2'])
    api.testModelSlot.mockRejectedValue({ message: 'Image generation is not enabled for this group' })
    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    await w.find('[data-test=key]').setValue('1')
    await w.find('[data-test=fetch]').trigger('click')
    await flushPromises()
    await w.find('[data-test=model]').setValue('gpt-image-2')
    await w.find('[data-test=set-image]').trigger('click')
    await w.find('[data-test=test-image]').trigger('click')
    await flushPromises()

    expect(w.text()).toContain('Image generation is not enabled for this group')
  })
})
