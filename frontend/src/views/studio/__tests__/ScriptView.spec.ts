import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const generateScript = vi.hoisted(() => vi.fn())
const getModelConfig = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ generateScript, getModelConfig }))

import ScriptView from '../ScriptView.vue'

const mountView = () =>
  mount(ScriptView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })

const longText = '夜'.repeat(60)

describe('ScriptView', () => {
  beforeEach(() => {
    generateScript.mockReset()
    getModelConfig.mockReset()
    getModelConfig.mockResolvedValue({ text: { api_key_id: 1, model: 'gpt-5.4' } })
  })

  it('disables submit until content reaches the minimum length', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('#sc-content').setValue('太短')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('#sc-content').setValue(longText)
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeUndefined()
  })

  it('blocks script generation when no text model is configured', async () => {
    getModelConfig.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#sc-content').setValue(longText)
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('还没配置文案模型')
  })

  it('submits the script request and renders the scenes', async () => {
    generateScript.mockResolvedValue({
      title: '北境书塔 · 短剧',
      form: 'short',
      scenes: [
        { heading: '场景一 · 雨夜', content: '主角推开书塔大门。' },
        { heading: '场景二 · 顶层', content: '城市在脚下铺开。' },
      ],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#sc-content').setValue(longText)
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(generateScript).toHaveBeenCalledWith(expect.objectContaining({ content: longText, form: 'short' }))
    expect(wrapper.findAll('.scene')).toHaveLength(2)
    expect(wrapper.text()).toContain('场景一 · 雨夜')
  })
})
