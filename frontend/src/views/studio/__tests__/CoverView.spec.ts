import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const generateCover = vi.hoisted(() => vi.fn())
const getModelConfig = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/studio')>()
  return { ...actual, generateCover, getModelConfig }
})

import CoverView from '../CoverView.vue'

const mountView = () =>
  mount(CoverView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })

const clickByText = async (wrapper: ReturnType<typeof mountView>, text: string) => {
  const btn = wrapper.findAll('button').find((b) => b.text() === text)
  if (!btn) throw new Error(`button not found: ${text}`)
  await btn.trigger('click')
}

describe('CoverView', () => {
  beforeEach(() => {
    generateCover.mockReset()
    getModelConfig.mockReset()
    getModelConfig.mockResolvedValue({ image: { api_key_id: 1, model: 'gpt-image-1' } })
  })

  it('disables generate until novel mode required fields are filled', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('#cv-synopsis').setValue('每本书都是一座城')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('#cv-protagonist').setValue('林见微')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeUndefined()
  })

  it('blocks generation when no image model is configured', async () => {
    getModelConfig.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('每本书都是一座城')
    await wrapper.find('#cv-protagonist').setValue('林见微')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('还没配置生图模型')
  })

  it('submits a novel-mode cover request and renders covers', async () => {
    generateCover.mockResolvedValue({ covers: [{ id: 'a', url: 'a.png' }, { id: 'b', url: 'b.png' }] })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('每本书都是一座城')
    await wrapper.find('#cv-protagonist').setValue('林见微')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(generateCover).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'novel', synopsis: '每本书都是一座城', protagonist: '林见微' }),
    )
    expect(wrapper.findAll('figure')).toHaveLength(2)
  })

  it('submits a custom-mode cover request', async () => {
    generateCover.mockResolvedValue({ covers: [] })
    const wrapper = mountView()
    await flushPromises()

    await clickByText(wrapper, '自定义')
    await wrapper.find('#cv-prompt').setValue('赛博朋克雨夜街道')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(generateCover).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'custom', prompt: '赛博朋克雨夜街道' }),
    )
  })

  it('shows a backend-developing notice on 404', async () => {
    generateCover.mockRejectedValue({ response: { status: 404 } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('每本书都是一座城')
    await wrapper.find('#cv-protagonist').setValue('林见微')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('后端封面生成服务正在开发中')
  })
})
