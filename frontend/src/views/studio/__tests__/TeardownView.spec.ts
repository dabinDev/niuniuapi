import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const analyzeTeardown = vi.hoisted(() => vi.fn())
const getModelConfig = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ analyzeTeardown, getModelConfig }))

import TeardownView from '../TeardownView.vue'

const mountView = () =>
  mount(TeardownView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })

const longText = '雨'.repeat(120)

const sampleReport = {
  overall_score: 18,
  verdict: '套路堆叠，毫无新鲜感',
  summary: '签到流爽文，缺乏冲突。',
  scores: [{ label: '节奏', value: 22 }],
  highlights: ['节奏快'],
  rotten_points: ['签到系统毫无代价'],
  suggestions: ['给签到加代价'],
}

describe('TeardownView', () => {
  beforeEach(() => {
    analyzeTeardown.mockReset()
    getModelConfig.mockReset()
    getModelConfig.mockResolvedValue({ text: { api_key_id: 1, model: 'gpt-5.4' } })
  })

  it('disables submit until content reaches the minimum length', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('#td-content').setValue('太短了')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('#td-content').setValue(longText)
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeUndefined()
  })

  it('frames teardown as a structured diagnosis workspace', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('拆书诊断')
    expect(wrapper.text()).toContain('黄金三章')
    expect(wrapper.text()).toContain('节奏热区')
    expect(wrapper.text()).toContain('伏笔追踪')
  })

  it('blocks teardown when no text model is configured', async () => {
    getModelConfig.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#td-content').setValue(longText)
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('还没配置文案模型')
  })

  it('submits the teardown request and renders the report', async () => {
    analyzeTeardown.mockResolvedValue(sampleReport)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#td-content').setValue(longText)
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(analyzeTeardown).toHaveBeenCalledWith(expect.objectContaining({ content: longText, tone: 'savage' }))
    expect(wrapper.text()).toContain('套路堆叠，毫无新鲜感')
    expect(wrapper.text()).toContain('签到系统毫无代价')
  })
})
