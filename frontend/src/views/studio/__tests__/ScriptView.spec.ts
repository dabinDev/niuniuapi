import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const generateCreative = vi.hoisted(() => vi.fn())
const getModelConfig = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ generateCreative, getModelConfig }))

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
    generateCreative.mockReset()
    getModelConfig.mockReset()
    getModelConfig.mockResolvedValue({ text: { api_key_id: 1, model: 'gpt-5.4' } })
  })

  it('presents the page as the creative generation workspace', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('创作生成')
    expect(wrapper.text()).toContain('大纲')
    expect(wrapper.text()).toContain('正文续写')
    expect(wrapper.text()).toContain('改写')
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
    generateCreative.mockResolvedValue({
      title: '北境书塔 · 短剧',
      mode: 'short',
      summary: '竖屏短剧化，冲突前置。',
      sections: [
        { heading: '第 1 集 · 雨夜', content: '主角推开书塔大门。' },
        { heading: '第 2 集 · 顶层', content: '城市在脚下铺开。' },
      ],
      next_steps: ['回填主角代价规则'],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#sc-content').setValue(longText)
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(generateCreative).toHaveBeenCalledWith(expect.objectContaining({ content: longText, mode: 'short' }))
    expect(wrapper.findAll('.scene')).toHaveLength(2)
    expect(wrapper.text()).toContain('第 1 集 · 雨夜')
    expect(wrapper.text()).toContain('回填主角代价规则')
  })

  it('lets authors switch to outline mode before generating', async () => {
    generateCreative.mockResolvedValue({
      title: '北境书塔 · 大纲',
      mode: 'outline',
      sections: [{ heading: '卷一', content: '旧神书塔觉醒。' }],
      next_steps: [],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="creative-mode-outline"]').trigger('click')
    await wrapper.find('#sc-content').setValue(longText)
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(generateCreative).toHaveBeenCalledWith(expect.objectContaining({ mode: 'outline' }))
    expect(wrapper.text()).toContain('旧神书塔觉醒')
  })
})
