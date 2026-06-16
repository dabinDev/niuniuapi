import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const analyzeHotspot = vi.hoisted(() => vi.fn())
const getModelConfig = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ analyzeHotspot, getModelConfig }))

import HotspotView from '../HotspotView.vue'

const mountView = () =>
  mount(HotspotView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })

describe('HotspotView', () => {
  beforeEach(() => {
    analyzeHotspot.mockReset()
    getModelConfig.mockReset()
    getModelConfig.mockResolvedValue({ text: { api_key_id: 1, model: 'gpt-5.4' } })
  })

  it('renders hotspot as a usable benchmark workbench', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('爆款对标')
    expect(wrapper.text()).toContain('我的作品片段')
    expect(wrapper.text()).toContain('对标样本')
    expect(wrapper.text()).toContain('爆款雷达')
  })

  it('submits benchmark analysis and renders actions', async () => {
    analyzeHotspot.mockResolvedValue({
      market_score: 83,
      verdict: '钩子够狠，兑现还慢半拍',
      radar: [
        { label: '三秒钩子', value: 88 },
        { label: '爽点密度', value: 72 },
      ],
      tropes: ['开局压迫', '十章兑现'],
      gaps: ['第 2 章缺少可感知回报'],
      actions: ['把第 2 章补成一次低成本胜利'],
      samples: [{ title: '同题材样本', lesson: '每章末尾都留一个未兑现问题' }],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="hotspot-content"]').setValue('主角被家族放逐后得到旧神书塔。'.repeat(8))
    await wrapper.find('[data-test="hotspot-benchmark"]').setValue('同题材爆款前三章：压迫、反杀、资源差。')
    await wrapper.find('[data-test="hotspot-submit"]').trigger('click')
    await flushPromises()

    expect(analyzeHotspot).toHaveBeenCalledWith(expect.objectContaining({
      content: expect.stringContaining('旧神书塔'),
      benchmark: expect.stringContaining('爆款前三章'),
      goal: 'new-book',
    }))
    expect(wrapper.text()).toContain('钩子够狠')
    expect(wrapper.text()).toContain('把第 2 章补成一次低成本胜利')
  })
})
