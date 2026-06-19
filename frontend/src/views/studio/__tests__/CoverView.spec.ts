import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const getModelConfig = vi.hoisted(() => vi.fn())
const getCoverJob = vi.hoisted(() => vi.fn())
const showWarning = vi.hoisted(() => vi.fn())
const startCoverJob = vi.hoisted(() => vi.fn())
const polishCoverPrompt = vi.hoisted(() => vi.fn())
const listWorks = vi.hoisted(() => vi.fn())
const getWork = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/studio')>()
  return { ...actual, getCoverJob, getModelConfig, startCoverJob, polishCoverPrompt, listWorks, getWork }
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
    polishCoverPrompt.mockReset()
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
    expect(wrapper.find('[data-test="model-auto-config-strip"]').classes()).toContain('cover-hero-card-compact')
    expect(wrapper.find('[data-test="model-auto-config-strip"] p').classes()).toContain('cover-hero-copy')
    expect(wrapper.find('[data-test="cover-key-setup-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cover-key-setup-card"]').text()).toContain('去 API 密钥配置')
    expect(wrapper.text()).toContain('系统会优先自动选择第一把密钥')
  })

  it('loads the image model even when cover history fails', async () => {
    listWorks.mockRejectedValue(new Error('history unavailable'))
    getModelConfig.mockResolvedValue({ image: { api_key_id: 1, model: 'gpt-image-2' } })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('gpt-image-2')
    expect(wrapper.text()).not.toContain('系统会优先自动选择第一把密钥')
  })

  it('shows a three-step cover workflow and practical input checklist', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.cover-lab').classes()).toContain('studio-wide-shell')
    expect(wrapper.find('.cover-workbench').classes()).toContain('cover-workbench-wide')
    expect(wrapper.find('[data-test="cover-workflow-guide"]').classes()).toContain('cover-flow-mobile-readable')
    expect(wrapper.find('[data-test="cover-workflow-guide"]').classes()).toContain('cover-flow-wide')
    expect(wrapper.text()).toContain('Brief')
    expect(wrapper.text()).toContain('Model')
    expect(wrapper.text()).toContain('Result')
    expect(wrapper.text()).toContain('输入书名与题材')
    expect(wrapper.find('[data-test="cover-workflow-guide"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cover-empty-brief"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cover-empty-brief"]').text()).toContain('先定点击承诺')
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

  it('polishes a custom prompt with the configured text model', async () => {
    getModelConfig.mockResolvedValue({
      image: { api_key_id: 1, model: 'gpt-image-1' },
      text: { api_key_id: 2, model: 'gpt-5.5' },
    })
    polishCoverPrompt.mockResolvedValue({
      prompt: '竖版网络小说封面，赛博雨夜街道，主标题区醒目，封面文字清晰有设计感。',
    })
    const wrapper = mountView()
    await flushPromises()

    await clickByText(wrapper, '自定义')
    await wrapper.find('#cv-prompt').setValue('赛博朋克雨夜街道')
    await wrapper.find('[data-test="cover-polish-custom"]').trigger('click')
    await flushPromises()

    expect(polishCoverPrompt).toHaveBeenCalledWith(expect.objectContaining({ mode: 'custom', prompt: '赛博朋克雨夜街道' }))
    expect((wrapper.find('#cv-prompt').element as HTMLTextAreaElement).value).toContain('竖版网络小说封面')
    expect((wrapper.find('#cv-prompt').element as HTMLTextAreaElement).value).toContain('封面文字')
  })

  it('polishes novel-driven cover fields from synopsis', async () => {
    getModelConfig.mockResolvedValue({
      image: { api_key_id: 1, model: 'gpt-image-1' },
      text: { api_key_id: 2, model: 'gpt-5.5' },
    })
    polishCoverPrompt.mockResolvedValue({
      protagonist: '林见微，冷感书塔修复师，红斗篷',
      genre: '玄幻 / 悬疑',
      mood: '冷色悬疑，雨夜压迫感',
      key_scene: '雨夜书塔门前，书页化作群鸟',
      cover_title: '北境书塔',
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-title').setValue('北境书塔')
    await wrapper.find('#cv-synopsis').setValue('少年在雨夜进入一座会吞掉书名的书塔。')
    await wrapper.find('[data-test="cover-polish-novel"]').trigger('click')
    await flushPromises()

    expect(polishCoverPrompt).toHaveBeenCalledWith(expect.objectContaining({
      mode: 'novel',
      title: '北境书塔',
      synopsis: '少年在雨夜进入一座会吞掉书名的书塔。',
    }))
    expect((wrapper.find('#cv-protagonist').element as HTMLTextAreaElement).value).toContain('林见微')
    expect((wrapper.find('#cv-genre').element as HTMLInputElement).value).toBe('玄幻 / 悬疑')
    expect((wrapper.find('#cv-mood').element as HTMLInputElement).value).toContain('雨夜')
    expect((wrapper.find('#cv-scene').element as HTMLInputElement).value).toContain('书塔')
    expect((wrapper.find('#cv-cover-title').element as HTMLInputElement).value).toBe('北境书塔')
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

  it('localizes insufficient balance errors for cover generation', async () => {
    startCoverJob.mockRejectedValue({ message: 'Insufficient account balance' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('#cv-synopsis').setValue('balance synopsis')
    await wrapper.find('#cv-protagonist').setValue('balance protagonist')
    await wrapper.find('.submit-btn').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('账户余额不足，请先充值或联系管理员增加余额')
    expect(wrapper.text()).not.toContain('Insufficient account balance')
  })
})
