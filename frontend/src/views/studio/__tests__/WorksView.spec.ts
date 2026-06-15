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

  it('shows empty state when there are no works', async () => {
    listWorks.mockResolvedValue([])
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('还没有作品')
  })
})
