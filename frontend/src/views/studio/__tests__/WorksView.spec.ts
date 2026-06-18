import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const listWorks = vi.hoisted(() => vi.fn())
const getWork = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ listWorks, getWork }))

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

  it('frames the archive as a project-centered works dashboard', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="studio-workbench-shell"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('我的作品')
    expect(wrapper.text()).toContain('章节树')
    expect(wrapper.text()).toContain('设定库')
    expect(wrapper.text()).toContain('概览看板')
    expect(wrapper.text()).toContain('作品归档')

    for (const label of ['概览', '拆书', '爆款', '大纲', '正文', '剧本', '热榜']) {
      expect(wrapper.text()).toContain(label)
    }
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
