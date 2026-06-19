import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const analyzeHotspot = vi.hoisted(() => vi.fn())
const getModelConfig = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ analyzeHotspot, getModelConfig }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
}))

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
    routerPush.mockReset()
    localStorage.clear()
    getModelConfig.mockResolvedValue({ text: { api_key_id: 1, model: 'gpt-5.4' } })
  })

  it('renders hotspot as a usable benchmark workbench', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('爆款对标')
    expect(wrapper.text()).toContain('我的作品片段')
    expect(wrapper.text()).toContain('对标样本')
    expect(wrapper.text()).toContain('爆款雷达')
    const action = wrapper.findAll('a').find((link) => link.text() === '回到作品工作台')
    expect(action?.classes()).toContain('studio-action-link')
  })

  it('fills a same-genre benchmark template', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="hotspot-template-same-genre"]').trigger('click')

    expect((wrapper.find('[data-test="hotspot-content"]').element as HTMLTextAreaElement).value).toContain('我的开篇')
    expect((wrapper.find('[data-test="hotspot-benchmark"]').element as HTMLTextAreaElement).value).toContain('对标样本')
    expect(wrapper.find('[data-test="hotspot-submit"]').attributes('disabled')).toBeUndefined()
  })

  it('offers an empty-state shortcut that loads a benchmark sample', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('先选一个参照系')
    expect(wrapper.text()).toContain('生成可复制的改写队列')

    await wrapper.find('[data-test="hotspot-empty-sample"]').trigger('click')

    expect((wrapper.find('[data-test="hotspot-content"]').element as HTMLTextAreaElement).value.length).toBeGreaterThanOrEqual(80)
    expect((wrapper.find('[data-test="hotspot-benchmark"]').element as HTMLTextAreaElement).value.length).toBeGreaterThan(0)
    expect(wrapper.find('[data-test="hotspot-submit"]').attributes('disabled')).toBeUndefined()
  })

  it('shows a benchmark material map before analysis', async () => {
    const wrapper = mountView()
    await flushPromises()

    const map = wrapper.find('[data-test="hotspot-material-map"]')
    expect(map.exists()).toBe(true)
    expect(map.text()).toContain('我的开篇')
    expect(map.text()).toContain('热榜样本')
    expect(map.text()).toContain('读者反馈')
    expect(map.text()).toContain('改写目标')
  })

  it('uses a compact automatic model configuration strip when the text model is missing', async () => {
    getModelConfig.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()

    const strip = wrapper.find('[data-test="model-auto-config-strip"]')
    expect(strip.exists()).toBe(true)
    expect(strip.classes()).toContain('model-strip-compact')
    expect(strip.find('p').classes()).toContain('model-strip-copy')
    expect(strip.text()).toContain('创建第一把密钥')
    expect(strip.text()).toContain('自动选择最新文案模型')
  })

  it('prefills from a fanqie hotlist bridge payload', async () => {
    localStorage.setItem('studio_bridge_payload', JSON.stringify({
      source: 'fanqie',
      target: 'hotspot',
      title: '十日终焉',
      genre: '悬疑脑洞',
      content: '热榜样本：十日终焉',
      benchmark: '书名：十日终焉\n卖点：规则怪谈',
      brief: '参考热榜样本做对标',
    }))

    const wrapper = mountView()
    await flushPromises()

    expect((wrapper.find('[data-test="hotspot-content"]').element as HTMLTextAreaElement).value).toContain('热榜样本：十日终焉')
    expect((wrapper.find('[data-test="hotspot-benchmark"]').element as HTMLTextAreaElement).value).toContain('规则怪谈')
    expect(wrapper.find('[data-test="hotspot-bridge-notice"]').text()).toContain('已从番茄热榜带入素材')
    expect(localStorage.getItem('studio_bridge_payload')).toBeNull()
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

  it('sends benchmark conclusions to the generation workbench', async () => {
    analyzeHotspot.mockResolvedValue({
      market_score: 91,
      verdict: '卖点够强，开篇兑现还慢半拍',
      radar: [{ label: '三秒钩子', value: 88 }],
      tropes: ['开局压迫', '章末钩子'],
      gaps: ['第 2 章缺少可感知回报'],
      actions: ['把第 2 章补成一次低成本胜利', '章尾追加更大的代价钩子'],
      samples: [{ title: '热榜样本', lesson: '每章末尾都留一个未兑现问题' }],
    })
    const setItem = vi.spyOn(Storage.prototype, 'setItem')
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="hotspot-content"]').setValue('主角被家族逐出后进入旧神书塔，用记忆兑换禁忌知识准备反击。'.repeat(8))
    await wrapper.find('[data-test="hotspot-benchmark"]').setValue('同题材热榜样本：前三章完成压迫、反杀和代价钩子。')
    await wrapper.find('[data-test="hotspot-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="hotspot-send-generate"]').text()).toBe('带动作生成正文')

    await wrapper.find('[data-test="hotspot-send-generate"]').trigger('click')

    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('"target":"generate"'))
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('爆款潜力：91/100'))
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('下一步动作：把第 2 章补成一次低成本胜利'))
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('按爆款对标动作生成可直接改稿的正文方案'))
    expect(routerPush).toHaveBeenCalledWith('/studio/generate')
    setItem.mockRestore()
  })
})
