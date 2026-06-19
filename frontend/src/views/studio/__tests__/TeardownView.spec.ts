import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const analyzeTeardown = vi.hoisted(() => vi.fn())
const getModelConfig = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ analyzeTeardown, getModelConfig }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
}))

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
    routerPush.mockReset()
    localStorage.clear()
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

    expect(wrapper.find('.teardown-page').classes()).toContain('studio-wide-shell')
    expect(wrapper.find('.diagnosis-map').classes()).toContain('diagnosis-map-wide')
    expect(wrapper.find('.teardown-grid').classes()).toContain('teardown-grid-wide')
    expect(wrapper.text()).toContain('拆书诊断')
    expect(wrapper.text()).toContain('黄金三章')
    expect(wrapper.text()).toContain('节奏热区')
    expect(wrapper.text()).toContain('伏笔追踪')
    const action = wrapper.findAll('a').find((link) => link.text() === '去爆款对标')
    expect(action?.classes()).toContain('studio-action-link')
  })

  it('fills a golden-three-chapter diagnosis template for authors', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="teardown-template-golden"]').trigger('click')

    expect((wrapper.find('#td-title').element as HTMLInputElement).value).toBe('雨夜入塔')
    expect((wrapper.find('#td-genre').element as HTMLInputElement).value).toBe('玄幻悬疑')
    expect((wrapper.find('#td-content').element as HTMLTextAreaElement).value).toContain('第 1 章')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeUndefined()
  })

  it('offers an empty-state shortcut that loads a usable diagnosis sample', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('先看钩子，再看兑现')
    expect(wrapper.text()).toContain('可直接复制到对标和生成')

    await wrapper.find('[data-test="teardown-empty-sample"]').trigger('click')

    expect((wrapper.find('#td-title').element as HTMLInputElement).value.length).toBeGreaterThan(0)
    expect((wrapper.find('#td-content').element as HTMLTextAreaElement).value.length).toBeGreaterThanOrEqual(100)
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeUndefined()
  })

  it('shows a pre-submit quality gate for practical chapter diagnosis', async () => {
    const wrapper = mountView()
    await flushPromises()

    const gate = wrapper.find('[data-test="teardown-quality-gate"]')
    expect(gate.exists()).toBe(true)
    expect(gate.text()).toContain('开篇钩子')
    expect(gate.text()).toContain('首次爽点')
    expect(gate.text()).toContain('章尾钩子')
    expect(gate.text()).toContain('代价/伏笔')
  })

  it('turns gateway failures into actionable teardown guidance', async () => {
    analyzeTeardown.mockRejectedValue({ response: { status: 502, data: { message: 'Bad Gateway' } } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#td-content').setValue(longText)
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('模型或网关暂时不可用')
    expect(wrapper.text()).not.toContain('拆书失败，请稍后重试')
    expect(wrapper.text()).not.toContain('Bad Gateway')
  })

  it('prefills from a fanqie hotlist bridge payload', async () => {
    localStorage.setItem('studio_bridge_payload', JSON.stringify({
      source: 'fanqie',
      target: 'teardown',
      title: '十日终焉',
      genre: '悬疑脑洞',
      content: '热榜样本：十日终焉\n简介：死亡游戏与规则怪谈。',
      benchmark: '书名：十日终焉',
      brief: '诊断开篇钩子',
    }))

    const wrapper = mountView()
    await flushPromises()

    expect((wrapper.find('#td-title').element as HTMLInputElement).value).toBe('十日终焉')
    expect((wrapper.find('#td-genre').element as HTMLInputElement).value).toBe('悬疑脑洞')
    expect((wrapper.find('#td-content').element as HTMLTextAreaElement).value).toContain('死亡游戏')
    expect(wrapper.find('[data-test="teardown-bridge-notice"]').text()).toContain('已从番茄热榜带入素材')
    expect(localStorage.getItem('studio_bridge_payload')).toBeNull()
  })

  it('blocks teardown when no text model is configured', async () => {
    getModelConfig.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#td-content').setValue(longText)
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('系统会优先自动选择第一把密钥')
  })

  it('explains the automatic model configuration path when the text model is missing', async () => {
    getModelConfig.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()

    const strip = wrapper.find('[data-test="model-auto-config-strip"]')
    expect(strip.exists()).toBe(true)
    expect(strip.classes()).toContain('model-strip-compact')
    expect(strip.find('p').classes()).toContain('model-strip-copy')
    expect(strip.text()).toContain('创建第一把密钥')
    expect(strip.text()).toContain('自动选择最新文案模型')
    expect(strip.text()).toContain('手动测试后保存会记住')
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

  it('sends a finished diagnosis to benchmark and generation workbenches', async () => {
    analyzeTeardown.mockResolvedValue(sampleReport)
    const setItem = vi.spyOn(Storage.prototype, 'setItem')
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#td-title').setValue('北境书塔')
    await wrapper.find('#td-genre').setValue('玄幻悬疑')
    await wrapper.find('#td-content').setValue(longText)
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="teardown-send-hotspot"]').text()).toBe('送去爆款对标')
    expect(wrapper.find('[data-test="teardown-send-generate"]').text()).toBe('送去创作生成')

    await wrapper.find('[data-test="teardown-send-hotspot"]').trigger('click')
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('"target":"hotspot"'))
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('书名：北境书塔'))
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('烂点：签到系统毫无代价'))
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('对照同题材爆款样本'))
    expect(routerPush).toHaveBeenCalledWith('/studio/hotspot')

    await wrapper.find('[data-test="teardown-send-generate"]').trigger('click')
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('"target":"generate"'))
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('根据拆书建议生成改写方案'))
    expect(routerPush).toHaveBeenCalledWith('/studio/generate')
    setItem.mockRestore()
  })
})
