import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const listWorks = vi.hoisted(() => vi.fn())
const getWork = vi.hoisted(() => vi.fn())
const push = vi.hoisted(() => vi.fn())
const studioVisibility = vi.hoisted(() => ({ value: {} as Record<string, boolean> }))
const isAdmin = vi.hoisted(() => ({ value: false }))

vi.mock('@/api/studio', () => ({ listWorks, getWork }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))
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

import WorksView from '../WorksView.vue'

const mountView = () =>
  mount(WorksView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })

describe('WorksView', () => {
  beforeEach(() => {
    listWorks.mockReset()
    getWork.mockReset()
    push.mockReset()
    studioVisibility.value = {}
    isAdmin.value = false
    localStorage.clear()
    listWorks.mockResolvedValue([
      { id: 1, type: 'teardown', title: '拆书报告', model: 'gpt-5.4', created_at: '2026-06-15T00:00:00Z' },
    ])
  })

  it('lists works on mount', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(listWorks).toHaveBeenCalled()
    expect(wrapper.findAll('.work-card')).toHaveLength(1)
    expect(wrapper.text()).toContain('拆书报告')
  })

  it('opens the newest work automatically so the archive has immediate value', async () => {
    getWork.mockResolvedValue({
      id: 1,
      type: 'teardown',
      title: 'Auto Selected Report',
      model: 'gpt-5.4',
      created_at: '2026-06-15T00:00:00Z',
      output: { overall_score: 18, verdict: 'Ready to revise', suggestions: ['Keep chapter-end hooks'] },
    })

    const wrapper = mountView()
    await flushPromises()

    expect(getWork).toHaveBeenCalledWith(1)
    expect(wrapper.text()).toContain('Ready to revise')
    expect(wrapper.find('.work-card').classes()).toContain('active')
  })

  it('frames the archive as a project-centered works dashboard', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="studio-workbench-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="studio-workbench-shell"]').classes()).toContain('studio-wide-shell')
    expect(wrapper.find('.workbench-grid').classes()).toContain('workbench-grid-wide')
    expect(wrapper.find('[data-test="works-archive-grid"]').classes()).toContain('archive-grid-wide')
    expect(wrapper.find('[data-test="works-mission-control"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="works-queue-panel"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="works-status-panel"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="works-action-panel"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('我的作品')
    expect(wrapper.text()).toContain('作品任务台')
    expect(wrapper.text()).toContain('作品队列')
    expect(wrapper.text()).toContain('当前作品状态')
    expect(wrapper.text()).toContain('下一步动作')
    expect(wrapper.text()).toContain('章节树')
    expect(wrapper.text()).toContain('设定库')
    expect(wrapper.text()).toContain('概览看板')
    expect(wrapper.text()).toContain('作品归档')
    expect(wrapper.find('[data-test="works-action-panel"]').text()).toContain('热榜选样本')

    for (const label of ['概览', '拆书', '爆款', '大纲', '正文', '剧本', '热榜']) {
      expect(wrapper.text()).toContain(label)
    }
  })

  it('exposes compact mobile workbench layout hooks for visual regression checks', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="works-mission-control"]').classes()).toContain('mission-control-mobile-compact')
    expect(wrapper.find('[data-test="works-archive-section"]').classes()).toContain('archive-section-mobile-stack')
    expect(wrapper.find('[data-test="works-archive-grid"]').classes()).toContain('archive-grid-mobile-flow')
  })

  it('loads and renders a teardown detail when selected', async () => {
    getWork.mockResolvedValue({
      id: 1,
      type: 'teardown',
      title: '拆书报告',
      model: 'gpt-5.4',
      created_at: '2026-06-15T00:00:00Z',
      output: { overall_score: 18, verdict: '套路堆叠', rotten_points: ['签到无代价'], suggestions: ['加代价'] },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('.work-card').trigger('click')
    await flushPromises()

    expect(getWork).toHaveBeenCalledWith(1)
    expect(wrapper.text()).toContain('套路堆叠')
    expect(wrapper.text()).toContain('签到无代价')
  })

  it('loads and renders hotspot detail cards', async () => {
    listWorks.mockResolvedValue([
      { id: 3, type: 'hotspot', title: '爆款对标', model: 'gpt-5.4', created_at: '2026-06-15T02:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 3,
      type: 'hotspot',
      title: '爆款对标',
      model: 'gpt-5.4',
      created_at: '2026-06-15T02:00:00Z',
      output: {
        market_score: 83,
        verdict: '钩子够狠',
        tropes: ['开局压迫'],
        actions: ['补第 2 章兑现'],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('.work-card').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('钩子够狠')
    expect(wrapper.text()).toContain('补第 2 章兑现')
  })

  it('loads and renders imported chapter summaries', async () => {
    listWorks.mockResolvedValue([
      { id: 4, type: 'import', title: '北境书塔导入', model: 'local-importer', created_at: '2026-06-15T03:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 4,
      type: 'import',
      title: '北境书塔导入',
      model: 'local-importer',
      created_at: '2026-06-15T03:00:00Z',
      output: {
        chapters: [
          { title: '第 1 章 雨夜入塔', word_count: 1200 },
          { title: '第 2 章 旧约', word_count: 980 },
        ],
        next_actions: ['送去拆书诊断'],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('.work-card').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('第 1 章 雨夜入塔')
    expect(wrapper.text()).toContain('送去拆书诊断')
  })

  it('keeps full imported novels usable by summarizing long chapter lists', async () => {
    const chapters = Array.from({ length: 75 }, (_, index) => ({
      title: `第 ${index + 1} 章`,
      word_count: 1000 + index,
      source: `https://fanqienovel.com/reader/${index + 1}`,
    }))
    listWorks.mockResolvedValue([
      { id: 8, type: 'import', title: '完整本导入', model: 'fanqie-importer', created_at: '2026-06-15T03:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 8,
      type: 'import',
      title: '完整本导入',
      model: 'fanqie-importer',
      created_at: '2026-06-15T03:00:00Z',
      output: {
        chapters,
        next_actions: ['送去拆书诊断'],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="import-overview"]').text()).toContain('75')
    expect(wrapper.find('[data-test="import-note"]').text()).toContain('完整目录数据 75 章')
    expect(wrapper.find('[data-test="import-chapter-summary"]').classes()).toContain('scrollable-preview')
    expect(wrapper.findAll('[data-test="import-chapter-summary"] article')).toHaveLength(60)
    expect(wrapper.text()).toContain('第 60 章')
    expect(wrapper.text()).not.toContain('第 61 章')
  })

  it('turns imported chapters into a usable asset map instead of raw source logs', async () => {
    listWorks.mockResolvedValue([
      { id: 9, type: 'import', title: '完整本导入', model: 'fanqie-importer', created_at: '2026-06-15T03:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 9,
      type: 'import',
      title: '完整本导入',
      model: 'fanqie-importer',
      created_at: '2026-06-15T03:00:00Z',
      output: {
        chapters: [
          { title: '第1章 空屋', word_count: 1496, source: 'https://fanqienovel.com/reader/123' },
        ],
        next_actions: [],
      },
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="import-action-strip"]').text()).toContain('目录已入库')
    expect(wrapper.find('[data-test="import-chapter-summary"]').text()).toContain('第 01 章')
    expect(wrapper.find('[data-test="import-chapter-summary"]').text()).toContain('1,496 字')
    expect(wrapper.find('[data-test="import-chapter-summary"]').text()).toContain('番茄来源')
    expect(wrapper.find('[data-test="import-chapter-summary"]').text()).not.toContain('fanqienovel.com/reader')
  })

  it('does not claim full content is ready when imported chapters only contain a catalog', async () => {
    listWorks.mockResolvedValue([
      { id: 12, type: 'import', title: '目录导入', model: 'fanqie-importer', created_at: '2026-06-15T03:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 12,
      type: 'import',
      title: '目录导入',
      model: 'fanqie-importer',
      created_at: '2026-06-15T03:00:00Z',
      output: {
        chapters: Array.from({ length: 75 }, (_, index) => ({
          title: `第${index + 1}章 目录`,
          word_count: 0,
          source: `https://fanqienovel.com/reader/${index + 1}`,
        })),
        next_actions: [],
      },
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="import-action-strip"]').text()).toContain('目录已入库')
    expect(wrapper.find('[data-test="import-action-strip"]').text()).toContain('正文待采集')
    expect(wrapper.find('[data-test="import-content-state"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="import-content-state"]').classes()).toContain('catalog-only')
    expect(wrapper.find('[data-test="import-stage-rail"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="import-stage-rail"]').text()).toContain('目录')
    expect(wrapper.find('[data-test="import-stage-rail"]').text()).toContain('正文')
    expect(wrapper.find('[data-test="import-stage-rail"]').text()).toContain('再加工')
    expect(wrapper.find('[data-test="import-action-strip"]').text()).not.toContain('全本内容已入库')
    expect(wrapper.find('[data-test="import-note"]').text()).toContain('完整目录数据')
    expect(wrapper.find('[data-test="import-missing-words-note"]').text()).toContain('正文待采集')
    expect(wrapper.find('[data-test="import-missing-words-note"]').text()).not.toContain('章节正文仍已导入')
  })

  it('summarizes missing imported chapter word counts instead of repeating noisy row warnings', async () => {
    listWorks.mockResolvedValue([
      { id: 10, type: 'import', title: '目录导入', model: 'fanqie-importer', created_at: '2026-06-15T03:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 10,
      type: 'import',
      title: '目录导入',
      model: 'fanqie-importer',
      created_at: '2026-06-15T03:00:00Z',
      output: {
        chapters: [
          { title: '第1章 空屋', word_count: 0, source: 'https://fanqienovel.com/reader/1' },
          { title: '第2章 说谎', word_count: 0, source: 'https://fanqienovel.com/reader/2' },
          { title: '第3章 有技术的人', word_count: 0, source: 'https://fanqienovel.com/reader/3' },
        ],
        next_actions: [],
      },
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="import-overview"]').text()).toContain('待采集')
    expect(wrapper.find('[data-test="import-missing-words-note"]').text()).toContain('3 章')
    expect(wrapper.findAll('[data-test="import-chapter-word-count-pending"]')).toHaveLength(3)
    expect(wrapper.find('[data-test="import-chapter-summary"]').text()).not.toContain('字数待统计')
  })

  it('sends imported works into downstream studio tools with a reusable bridge payload', async () => {
    listWorks.mockResolvedValue([
      { id: 11, type: 'import', title: '十日终焉导入', model: 'fanqie-importer', created_at: '2026-06-15T03:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 11,
      type: 'import',
      title: '十日终焉导入',
      model: 'fanqie-importer',
      created_at: '2026-06-15T03:00:00Z',
      output: {
        chapters: [
          {
            title: '第1章 空屋',
            word_count: 1496,
            source: 'https://fanqienovel.com/reader/1',
            content: '齐夏在封闭房间里醒来，空气里都是消毒水味。',
          },
          {
            title: '第2章 说谎',
            word_count: 1530,
            source: 'https://fanqienovel.com/reader/2',
            content: '游戏规则开始出现，每个人都必须付出说谎的代价。',
          },
        ],
        next_actions: [],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="import-send-teardown"]').trigger('click')

    expect(push).toHaveBeenCalledWith('/studio/teardown')
    const payload = JSON.parse(localStorage.getItem('studio_bridge_payload') || '{}')
    expect(payload).toMatchObject({
      source: 'works',
      target: 'teardown',
      title: '十日终焉导入',
      genre: '番茄导入',
    })
    expect(payload.content).toContain('第1章 空屋')
    expect(payload.content).toContain('齐夏在封闭房间里醒来')
    expect(payload.content).toContain('第2章 说谎')
    expect(payload.content).toContain('每个人都必须付出说谎的代价')
    expect(payload.brief).toContain('十日终焉导入')
  })

  it('hides disabled studio entries and blocks import handoff to disabled tools', async () => {
    studioVisibility.value = { fanqie: false, teardown: false, hotspot: false, generate: false }
    listWorks.mockResolvedValue([
      { id: 11, type: 'import', title: '十日终焉导入', model: 'fanqie-importer', created_at: '2026-06-15T03:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 11,
      type: 'import',
      title: '十日终焉导入',
      model: 'fanqie-importer',
      created_at: '2026-06-15T03:00:00Z',
      output: {
        chapters: [{ title: '第1章 空屋', word_count: 1496, content: '齐夏醒来。' }],
        next_actions: [],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).not.toContain('查看热榜')
    expect(wrapper.text()).not.toContain('热榜选样本')
    expect(wrapper.text()).not.toContain('拆书诊断')
    expect(wrapper.text()).not.toContain('创作生成')
    expect(wrapper.find('[data-test="import-send-teardown"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="import-send-hotspot"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="import-send-generate"]').exists()).toBe(false)
    expect(push).not.toHaveBeenCalled()
    expect(localStorage.getItem('studio_bridge_payload')).toBeNull()
  })

  it('opens cover works in the shared image viewer with multi-image navigation', async () => {
    listWorks.mockResolvedValue([
      { id: 2, type: 'cover', title: 'Cover Set', model: 'gpt-image-2', created_at: '2026-06-15T01:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 2,
      type: 'cover',
      title: 'Cover Set',
      model: 'gpt-image-2',
      created_at: '2026-06-15T01:00:00Z',
      output: {
        covers: [
          { id: 'cover-a', url: 'cover-a.png' },
          { id: 'cover-b', url: 'cover-b.png' },
        ],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('.work-card').trigger('click')
    await flushPromises()
    await wrapper.find('[data-test="work-cover-thumb-0"]').trigger('click')

    expect(wrapper.find('[data-test="cover-viewer"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cover-viewer-counter"]').text()).toContain('1 / 2')
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('src')).toBe('cover-a.png')

    await wrapper.find('[data-test="cover-next"]').trigger('click')
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('src')).toBe('cover-b.png')
  })

  it('shows empty state when there are no works', async () => {
    listWorks.mockResolvedValue([])
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('还没有作品')
  })
})
