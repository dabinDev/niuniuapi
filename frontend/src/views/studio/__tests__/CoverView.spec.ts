import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const getModelConfig = vi.hoisted(() => vi.fn())
const getCoverJob = vi.hoisted(() => vi.fn())
const showWarning = vi.hoisted(() => vi.fn())
const startCoverJob = vi.hoisted(() => vi.fn())
const listWorks = vi.hoisted(() => vi.fn())
const getWork = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/studio')>()
  return { ...actual, getCoverJob, getModelConfig, startCoverJob, listWorks, getWork }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showWarning }),
}))

import CoverView from '../CoverView.vue'

const mountView = () =>
  mount(CoverView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { template: '<a><slot /></a>' },
        ImageUpload: {
          props: ['modelValue', 'maxSize', 'hint'],
          emits: ['update:modelValue'],
          template:
            '<div data-test="ref-upload" :data-max-size="maxSize" :data-hint="hint"><input id="cv-ref-upload" type="file" @change="$emit(\'update:modelValue\', \'data:image/png;base64,cmVm\')" /></div>',
        },
      },
    },
  })

const clickByText = async (wrapper: ReturnType<typeof mountView>, text: string) => {
  const btn = wrapper.findAll('button').find((b) => b.text() === text)
  if (!btn) throw new Error(`button not found: ${text}`)
  await btn.trigger('click')
}

describe('CoverView', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
    getCoverJob.mockReset()
    getModelConfig.mockReset()
    getWork.mockReset()
    listWorks.mockReset()
    startCoverJob.mockReset()
    showWarning.mockReset()
    localStorage.clear()
    getModelConfig.mockResolvedValue({ image: { api_key_id: 1, model: 'gpt-image-1' } })
    getWork.mockResolvedValue({})
    listWorks.mockResolvedValue([])
    startCoverJob.mockResolvedValue({ job_id: 'job-1', status: 'succeeded', progress: 100, result: { covers: [] } })
  })

  it('disables generate until novel mode required fields are filled', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('#cv-synopsis').setValue('每本书都是一座城')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('#cv-protagonist').setValue('林见微')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeUndefined()
  })

  it('blocks generation when no image model is configured', async () => {
    getModelConfig.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('每本书都是一座城')
    await wrapper.find('#cv-protagonist').setValue('林见微')
    expect(wrapper.find('.submit-btn').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('还没配置生图模型')
  })

  it('submits a novel-mode cover request and renders covers', async () => {
    startCoverJob.mockResolvedValue({ job_id: 'job-1', status: 'succeeded', progress: 100, result: { covers: [{ id: 'a', url: 'a.png' }, { id: 'b', url: 'b.png' }] } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('每本书都是一座城')
    await wrapper.find('#cv-protagonist').setValue('林见微')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(startCoverJob).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'novel', synopsis: '每本书都是一座城', protagonist: '林见微' }),
    )
    expect(wrapper.findAll('figure')).toHaveLength(2)
  })

  it('opens generated covers in a draggable zoomable viewer with download and navigation controls', async () => {
    startCoverJob.mockResolvedValue({
      job_id: 'job-1',
      status: 'succeeded',
      progress: 100,
      result: { covers: [{ id: 'a', url: 'a.png' }, { id: 'b', url: 'b.png' }] },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('viewer synopsis')
    await wrapper.find('#cv-protagonist').setValue('viewer protagonist')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    await wrapper.find('[data-test="cover-thumb-0"]').trigger('click')
    expect(wrapper.find('[data-test="cover-viewer"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cover-viewer-counter"]').text()).toContain('1 / 2')
    expect(wrapper.find('[data-test="cover-download"]').attributes('href')).toBe('a.png')

    await wrapper.find('[data-test="cover-viewer-stage"]').trigger('wheel', { deltaY: -120 })
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('style')).toContain('scale(1.25)')

    await wrapper.find('[data-test="cover-viewer-stage"]').trigger('wheel', { deltaY: 120 })
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('style')).toContain('scale(1)')

    await wrapper.find('[data-test="cover-zoom-in"]').trigger('click')
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('style')).toContain('scale(1.25)')

    await wrapper.find('[data-test="cover-viewer-stage"]').trigger('mousedown', { clientX: 100, clientY: 100 })
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 136, clientY: 124 }))
    window.dispatchEvent(new MouseEvent('mouseup'))
    await flushPromises()
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('style')).toContain('translate3d(36px, 24px')

    await wrapper.find('[data-test="cover-next"]').trigger('click')
    expect(wrapper.find('[data-test="cover-viewer-counter"]').text()).toContain('2 / 2')
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('src')).toBe('b.png')

    await wrapper.find('[data-test="cover-prev"]').trigger('click')
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('src')).toBe('a.png')

    await wrapper.find('[data-test="cover-zoom-reset"]').trigger('click')
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('style')).toContain('scale(1)')

    await wrapper.find('[data-test="cover-close"]').trigger('click')
    await flushPromises()
    await new Promise((resolve) => window.setTimeout(resolve, 260))
    await flushPromises()
    expect(wrapper.find('[data-test="cover-viewer"]').exists()).toBe(false)
  })

  it('submits a custom-mode cover request', async () => {
    const wrapper = mountView()
    await flushPromises()

    await clickByText(wrapper, '自定义')
    await wrapper.find('#cv-prompt').setValue('赛博朋克雨夜街道')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(startCoverJob).toHaveBeenCalledWith(
      expect.objectContaining({ mode: 'custom', prompt: '赛博朋克雨夜街道' }),
    )
  })

  it('uses an upload control for custom reference images and sends the uploaded data URL', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button')[0].trigger('click')
    expect(wrapper.find('#cv-ref').exists()).toBe(false)
    expect(wrapper.find('#cv-ref-upload').exists()).toBe(true)
    expect(wrapper.find('[data-test="ref-upload"]').attributes('data-max-size')).toBe(String(20 * 1024 * 1024))
    expect(wrapper.find('[data-test="ref-upload"]').attributes('data-hint')).toContain('20MB')

    await wrapper.find('#cv-prompt').setValue('uploaded reference prompt')
    await wrapper.find('#cv-ref-upload').trigger('change')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(startCoverJob).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: 'custom',
        prompt: 'uploaded reference prompt',
        ref_image: 'data:image/png;base64,cmVm',
      }),
    )
  })

  it('defaults gpt-image models to one image and queues multi-image requests as one-image jobs', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    startCoverJob.mockImplementation(async () => ({
      job_id: `job-${startCoverJob.mock.calls.length}`,
      status: 'succeeded',
      progress: 100,
      result: { covers: [{ id: `cover-${startCoverJob.mock.calls.length}`, url: `${startCoverJob.mock.calls.length}.png` }] },
    }))
    const wrapper = mountView()
    await flushPromises()

    expect((wrapper.find('#cv-count').element as HTMLSelectElement).value).toBe('1')

    await wrapper.find('#cv-synopsis').setValue('queue synopsis')
    await wrapper.find('#cv-protagonist').setValue('queue protagonist')
    await wrapper.find('#cv-count').setValue('3')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(confirmSpy).toHaveBeenCalledWith(expect.stringContaining('队列'))
    expect(showWarning).toHaveBeenCalledWith(expect.stringContaining('队列'), 6000)
    expect(startCoverJob).toHaveBeenCalledTimes(3)
    expect(startCoverJob.mock.calls.map(([payload]) => payload.count)).toEqual([1, 1, 1])
    expect(wrapper.findAll('figure')).toHaveLength(3)
  })

  it('polls cover job progress every five seconds until the image is ready', async () => {
    vi.useFakeTimers()
    startCoverJob.mockResolvedValue({ job_id: 'job-1', status: 'running', progress: 20, message: 'started' })
    getCoverJob
      .mockResolvedValueOnce({ job_id: 'job-1', status: 'running', progress: 55, message: 'generating' })
      .mockResolvedValueOnce({
        job_id: 'job-1',
        status: 'succeeded',
        progress: 100,
        result: { covers: [{ id: 'done', url: 'done.png' }] },
      })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('polling synopsis')
    await wrapper.find('#cv-protagonist').setValue('polling protagonist')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(startCoverJob).toHaveBeenCalledTimes(1)
    expect(getCoverJob).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('20%')

    await vi.advanceTimersByTimeAsync(4999)
    await flushPromises()
    expect(getCoverJob).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1)
    await flushPromises()
    expect(getCoverJob).toHaveBeenCalledWith('job-1')
    expect(wrapper.text()).toContain('55%')

    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    expect(getCoverJob).toHaveBeenCalledTimes(2)
    expect(wrapper.findAll('figure')).toHaveLength(1)
  })

  it('restores the last selected tab and previous generated covers on the next visit', async () => {
    localStorage.setItem('studio_cover_mode', 'custom')
    localStorage.setItem(
      'studio_cover_last_result',
      JSON.stringify({
        title: 'Saved Cover',
        covers: [{ id: 'saved-cover', url: 'saved-cover.png' }],
        created_at: '2026-06-15T01:00:00Z',
      }),
    )

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('#cv-prompt').exists()).toBe(true)
    expect(wrapper.find('[data-test="cover-thumb-0"] img').attributes('src')).toBe('saved-cover.png')
  })

  it('loads recent cover history as a horizontal timeline and opens records in the viewer', async () => {
    listWorks.mockResolvedValue([
      { id: 3, type: 'cover', title: 'Newest Cover', model: 'gpt-image-2', created_at: '2026-06-15T03:00:00Z' },
      { id: 2, type: 'cover', title: 'Older Cover', model: 'gpt-image-2', created_at: '2026-06-15T02:00:00Z' },
    ])
    getWork.mockImplementation(async (id: number) => ({
      id,
      type: 'cover',
      title: id === 3 ? 'Newest Cover' : 'Older Cover',
      model: 'gpt-image-2',
      created_at: id === 3 ? '2026-06-15T03:00:00Z' : '2026-06-15T02:00:00Z',
      output: {
        covers: [{ id: `history-${id}`, url: `history-${id}.png` }],
      },
    }))

    const wrapper = mountView()
    await flushPromises()
    await flushPromises()

    expect(listWorks).toHaveBeenCalledWith('cover')
    const cards = wrapper.findAll('[data-test="cover-history-card"]')
    expect(cards).toHaveLength(2)
    expect(cards[0].text()).toContain('Newest Cover')
    expect(cards[0].find('img').attributes('src')).toBe('history-3.png')

    await wrapper.find('[data-test="cover-history-thumb-0"]').trigger('click')
    expect(wrapper.find('[data-test="cover-viewer"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('src')).toBe('history-3.png')
  })

  it('shows a readable fallback when a cover history thumbnail fails to load', async () => {
    listWorks.mockResolvedValue([
      { id: 5, type: 'cover', title: 'Missing Cover', model: 'gpt-image-2', created_at: '2026-06-15T05:00:00Z' },
    ])
    getWork.mockResolvedValue({
      id: 5,
      type: 'cover',
      title: 'Missing Cover',
      model: 'gpt-image-2',
      created_at: '2026-06-15T05:00:00Z',
      output: {
        covers: [{ id: 'missing-cover', url: 'missing-cover.png' }],
      },
    })

    const wrapper = mountView()
    await flushPromises()
    await flushPromises()

    await wrapper.find('[data-test="cover-history-thumb-0"] img').trigger('error')

    expect(wrapper.find('[data-test="cover-history-placeholder-0"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cover-history-placeholder-0"]').text()).toContain('封面加载失败')
  })

  it('shows a backend-developing notice on 404', async () => {
    startCoverJob.mockRejectedValue({ response: { status: 404 } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('每本书都是一座城')
    await wrapper.find('#cv-protagonist').setValue('林见微')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('后端封面生成服务正在开发中')
  })

  it('shows a timeout error instead of the backend-developing notice when generation runs too long', async () => {
    startCoverJob.mockRejectedValue({ code: 'ECONNABORTED', message: 'timeout of 30000ms exceeded' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('queue synopsis')
    await wrapper.find('#cv-protagonist').setValue('queue protagonist')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('后端封面生成服务正在开发中')
    expect(wrapper.text()).toContain('生成耗时较长')
  })
})
