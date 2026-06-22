import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const generateCreative = vi.hoisted(() => vi.fn())
const getModelConfig = vi.hoisted(() => vi.fn())
const studioVisibility = vi.hoisted(() => ({ value: {} as Record<string, boolean> }))
const isAdmin = vi.hoisted(() => ({ value: false }))

vi.mock('@/api/studio', () => ({ generateCreative, getModelConfig }))
vi.mock('@/stores', () => ({
  useAppStore: () => ({
    cachedPublicSettings: { studio_feature_visibility: studioVisibility.value },
  }),
  useAuthStore: () => ({
    get isAdmin() {
      return isAdmin.value
    },
  }),
}))

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
    studioVisibility.value = {}
    isAdmin.value = false
    getModelConfig.mockResolvedValue({ text: { api_key_id: 1, model: 'gpt-5.4' } })
  })

  it('presents the page as the creative generation workspace', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.creative-page').classes()).toContain('studio-wide-shell')
    expect(wrapper.find('.creative-grid').classes()).toContain('creative-grid-wide')
    expect(wrapper.find('.mode-grid').classes()).toContain('mode-grid-wide')
    expect(wrapper.text()).toContain('创作生成')
    expect(wrapper.text()).toContain('大纲')
    expect(wrapper.text()).toContain('正文续写')
    expect(wrapper.text()).toContain('改写')
    const action = wrapper.findAll('a').find((link) => link.text() === '查看我的作品')
    expect(action?.classes()).toContain('studio-action-link')
  })

  it('fills a rewrite template and switches to rewrite mode', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="creative-template-rewrite"]').trigger('click')

    expect((wrapper.find('#sc-content').element as HTMLTextAreaElement).value).toContain('原章节')
    expect((wrapper.find('input[placeholder*="把第 2 章"]').element as HTMLInputElement).value).toContain('低成本胜利')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeUndefined()
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()
    expect(generateCreative).toHaveBeenCalledWith(expect.objectContaining({ mode: 'rewrite' }))
  })

  it('offers an empty-state shortcut that loads a rewrite sample', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('先定生成模式')
    expect(wrapper.text()).toContain('输出能复制、能归档、能继续加工')

    await wrapper.find('[data-test="creative-empty-sample"]').trigger('click')

    expect((wrapper.find('#sc-content').element as HTMLTextAreaElement).value.length).toBeGreaterThanOrEqual(50)
    expect((wrapper.find('[data-test="creative-genre"]').element as HTMLInputElement).value.length).toBeGreaterThan(0)
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeUndefined()
  })

  it('shows an output contract for the selected generation mode', async () => {
    const wrapper = mountView()
    await flushPromises()

    const contract = wrapper.find('[data-test="creative-output-contract"]')
    expect(contract.exists()).toBe(true)
    expect(contract.text()).toContain('输出承诺')
    expect(contract.text()).toContain('强冲突分集')
    expect(contract.text()).toContain('生成后检查')

    await wrapper.find('[data-test="creative-mode-outline"]').trigger('click')
    expect(contract.text()).toContain('卷纲')
    expect(contract.text()).toContain('章节钩子')
  })

  it('prefills from a fanqie hotlist bridge payload', async () => {
    localStorage.setItem('studio_bridge_payload', JSON.stringify({
      source: 'fanqie',
      target: 'generate',
      title: '十日终焉',
      genre: '悬疑脑洞',
      content: '热榜样本：十日终焉\n简介：死亡游戏与规则怪谈。',
      benchmark: '书名：十日终焉',
      brief: '参考热榜样本生成新书方向',
    }))

    const wrapper = mountView()
    await flushPromises()

    expect((wrapper.find('#sc-content').element as HTMLTextAreaElement).value).toContain('热榜样本：十日终焉')
    expect((wrapper.find('[data-test="creative-genre"]').element as HTMLInputElement).value).toBe('悬疑脑洞')
    expect(wrapper.find('[data-test="creative-bridge-notice"]').text()).toContain('已从番茄热榜带入素材')
    expect(localStorage.getItem('studio_bridge_payload')).toBeNull()
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
    expect(wrapper.find('[data-test="model-auto-config-strip"]').classes()).toContain('model-strip-compact')
    expect(wrapper.find('[data-test="model-auto-config-strip"] p').classes()).toContain('model-strip-copy')
    expect(wrapper.text()).toContain('系统会优先自动选择第一把密钥')
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

  it('hides works archive entries when the works feature is disabled', async () => {
    studioVisibility.value = { works: false }
    generateCreative.mockResolvedValue({
      title: '生成结果',
      mode: 'outline',
      sections: [{ heading: '第一章', content: '正文结果' }],
      next_steps: [],
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).not.toContain('查看我的作品')

    await wrapper.find('#sc-content').setValue(longText)
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('查看归档')
  })
})
