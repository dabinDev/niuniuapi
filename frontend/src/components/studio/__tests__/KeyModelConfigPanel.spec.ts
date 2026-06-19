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
    api.getKeyModels.mockResolvedValue([])
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
    expect(api.saveModelConfig).toHaveBeenLastCalledWith({
      image: { api_key_id: 1, model: 'gpt-image-1' },
      text: { api_key_id: 1, model: 'gpt-5.4' },
    })
  })

  it('preloads a saved config', async () => {
    api.getModelConfig.mockResolvedValue({ image: { api_key_id: 1, model: 'gpt-image-1' } })
    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    expect(w.find('[data-test=slot-image]').text()).toContain('gpt-image-1')
    expect(w.find('[data-test=save]').attributes('disabled')).toBeUndefined()
  })

  it('automatically configures image and text models from the first key when no config exists', async () => {
    api.getKeyModels.mockResolvedValue(['gpt-4.1', 'gpt-image-2', 'gpt-5.1'])
    api.saveModelConfig.mockResolvedValue(undefined)

    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    expect(api.getKeyModels).toHaveBeenCalledWith(1)
    expect(api.saveModelConfig).toHaveBeenCalledWith({
      image: { api_key_id: 1, model: 'gpt-image-2' },
      text: { api_key_id: 1, model: 'gpt-5.1' },
    })
    expect(w.find('[data-test=slot-image]').text()).toContain('gpt-image-2')
    expect(w.find('[data-test=slot-text]').text()).toContain('gpt-5.1')
    expect(w.text()).toContain('最新生图模型')
    expect(w.text()).toContain('最新文案模型')
    expect(w.text()).toContain('已根据第一把密钥自动配置模型')
    expect(w.find('[data-test=auto-config-status]').text()).toContain('已准备好创作模型')
  })

  it('keeps saved manual config instead of auto-overwriting it', async () => {
    api.getModelConfig.mockResolvedValue({
      image: { api_key_id: 1, model: 'saved-image-model' },
      text: { api_key_id: 1, model: 'saved-text-model' },
    })
    api.getKeyModels.mockResolvedValue(['gpt-image-2', 'gpt-5.1'])

    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    expect(api.saveModelConfig).not.toHaveBeenCalled()
    expect(w.find('[data-test=slot-image]').text()).toContain('saved-image-model')
    expect(w.find('[data-test=slot-text]').text()).toContain('saved-text-model')
  })

  it('does not require testing before saving automatically selected defaults', async () => {
    api.getKeyModels.mockResolvedValue(['gpt-image-2', 'gpt-5.1'])
    api.saveModelConfig.mockResolvedValue(undefined)

    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    expect(w.find('[data-test=save]').attributes('disabled')).toBeUndefined()
  })

  it('waits for the first key before auto configuring models', async () => {
    api.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100, pages: 0 })
    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    expect(api.getKeyModels).not.toHaveBeenCalled()
    expect(api.saveModelConfig).not.toHaveBeenCalled()
    expect(w.find('[data-test=auto-config-status]').text()).toContain('创建第一把密钥后自动配置')
  })

  it('shows an actionable empty-key guide before the first key exists', async () => {
    api.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100, pages: 0 })
    const w = mount(KeyModelConfigPanel)
    await flushPromises()

    expect(w.find('[data-test=empty-key-guide]').exists()).toBe(true)
    expect(w.find('[data-test=empty-key-guide]').text()).toContain('创建第一把密钥')
    expect(w.find('[data-test=key]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-test=fetch]').attributes('disabled')).toBeDefined()
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
