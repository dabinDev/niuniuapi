import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const importStudioContent = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ importStudioContent }))

import DownloaderView from '../DownloaderView.vue'

const mountView = () =>
  mount(DownloaderView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })

describe('DownloaderView', () => {
  beforeEach(() => {
    importStudioContent.mockReset()
  })

  it('frames the downloader as a compliant personal importer', () => {
    const wrapper = mountView()

    expect(wrapper.text()).toContain('番茄导入器')
    expect(wrapper.text()).toContain('个人作品备份')
    expect(wrapper.text()).toContain('我确认')
  })

  it('imports pasted chapters after compliance confirmation', async () => {
    importStudioContent.mockResolvedValue({
      title: '北境书塔',
      source: 'manual',
      status: 'completed',
      chapters: [
        { title: '第 1 章 雨夜入塔', word_count: 1200 },
        { title: '第 2 章 旧约', word_count: 980 },
      ],
      notes: ['已按章节标题拆分'],
      next_actions: ['送去拆书诊断', '生成续写大纲'],
    })
    const wrapper = mountView()

    await wrapper.find('[data-test="import-title"]').setValue('北境书塔')
    await wrapper.find('[data-test="import-content"]').setValue('第 1 章 雨夜入塔\n正文……\n\n第 2 章 旧约\n正文……')
    await wrapper.find('[data-test="import-consent"]').setValue(true)
    await wrapper.find('[data-test="import-submit"]').trigger('click')
    await flushPromises()

    expect(importStudioContent).toHaveBeenCalledWith(expect.objectContaining({
      source: 'manual',
      title: '北境书塔',
      consent: true,
    }))
    expect(wrapper.text()).toContain('第 1 章 雨夜入塔')
    expect(wrapper.text()).toContain('送去拆书诊断')
  })
})
