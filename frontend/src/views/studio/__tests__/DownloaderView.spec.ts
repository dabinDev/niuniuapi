import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const getFanqieRank = vi.hoisted(() => vi.fn())
const searchFanqieBooks = vi.hoisted(() => vi.fn())
const downloadFanqieBook = vi.hoisted(() => vi.fn())
const analyzeFanqieBook = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ getFanqieRank, searchFanqieBooks, downloadFanqieBook, analyzeFanqieBook }))

import DownloaderView from '../DownloaderView.vue'

const rankBooks = Array.from({ length: 30 }, (_, index) => ({
  id: String(1000 + index),
  rank: index + 1,
  title: `榜单小说 ${index + 1}`,
  author: `作者 ${index + 1}`,
  category: index % 2 ? '都市日常' : '玄幻脑洞',
  status: '连载中',
  word_count: `${80 + index}万字`,
  score: `${99 - index} 热度`,
  description: `第 ${index + 1} 本榜单书的卖点说明`,
  cover_url: `https://img.example.com/cover-${index + 1}.jpg`,
  source_url: `https://fanqienovel.com/page/${1000 + index}`,
  tags: ['强钩子', '可对标'],
}))

const mountView = () =>
  mount(DownloaderView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })

describe('DownloaderView as FanqieHotlist', () => {
  beforeEach(() => {
    getFanqieRank.mockReset()
    searchFanqieBooks.mockReset()
    downloadFanqieBook.mockReset()
    analyzeFanqieBook.mockReset()
    getFanqieRank.mockResolvedValue({
      channel: 'hot',
      updated_at: '2026-06-17T00:00:00Z',
      source: 'fanqie-rank',
      books: rankBooks,
    })
  })

  it('loads the hotlist by default and frames the page as fanqie rank research', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(getFanqieRank).toHaveBeenCalledWith('hot')
    expect(wrapper.text()).toContain('番茄热榜')
    expect(wrapper.text()).toContain('热榜')
    expect(wrapper.text()).toContain('巅峰榜')
    expect(wrapper.text()).toContain('男生榜')
    expect(wrapper.text()).toContain('女生榜')
    expect(wrapper.findAll('[data-test="fanqie-rank-row"]')).toHaveLength(30)
  })

  it('renders book covers in the list and selected sample card', async () => {
    const wrapper = mountView()
    await flushPromises()

    const listCovers = wrapper.findAll('[data-test="fanqie-cover"]')
    expect(listCovers).toHaveLength(30)
    expect(listCovers[0].attributes('src')).toBe('https://img.example.com/cover-1.jpg')
    expect(listCovers[0].attributes('alt')).toContain(rankBooks[0].title)

    const selectedCover = wrapper.find('[data-test="fanqie-selected-cover"]')
    expect(selectedCover.exists()).toBe(true)
    expect(selectedCover.attributes('src')).toBe('https://img.example.com/cover-1.jpg')
  })

  it('switches rank channels', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-channel-male"]').trigger('click')
    await flushPromises()

    expect(getFanqieRank).toHaveBeenLastCalledWith('male')
  })

  it('searches a named fanqie novel', async () => {
    searchFanqieBooks.mockResolvedValue([
      {
        id: '42',
        rank: 1,
        title: '十日终焉',
        author: '杀虫队队员',
        category: '悬疑脑洞',
        status: '已完结',
        word_count: '240万字',
        score: '搜索命中',
        description: '指定小说搜索结果',
        cover_url: '',
        source_url: 'https://fanqienovel.com/page/42',
        tags: ['搜索'],
      },
    ])
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-search-input"]').setValue('十日终焉')
    await wrapper.find('[data-test="fanqie-search-submit"]').trigger('click')
    await flushPromises()

    expect(searchFanqieBooks).toHaveBeenCalledWith('十日终焉')
    expect(wrapper.text()).toContain('搜索结果')
    expect(wrapper.text()).toContain('十日终焉')
  })

  it('shows backend search failure details', async () => {
    searchFanqieBooks.mockRejectedValue({
      status: 502,
      message: '番茄官方搜索接口触发验证码校验，暂时无法自动按书名搜索；请粘贴番茄作品页链接或作品 ID 后重试',
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-search-input"]').setValue('十日终焉')
    await wrapper.find('[data-test="fanqie-search-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('验证码校验')
    expect(wrapper.text()).toContain('粘贴番茄作品页链接')
  })

  it('downloads the selected book after the user confirms backup rights', async () => {
    downloadFanqieBook.mockResolvedValue({
      title: '榜单小说 1',
      file_name: '榜单小说 1.txt',
      chapter_count: 30,
      text: 'Book 1\n\nChapter 1',
      status: 'completed',
      source: 'fanqie',
      decode_status: 'browser_required',
      notes: ['仅限个人备份'],
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('分析前10章')
    expect(wrapper.text()).toContain('下载/导入完整小说 TXT')
    await wrapper.find('[data-test="fanqie-consent"]').setValue(true)
    await wrapper.find('[data-test="fanqie-download"]').trigger('click')
    await flushPromises()

    expect(downloadFanqieBook).toHaveBeenCalledWith(rankBooks[0], true)
    expect(wrapper.text()).toContain('需要浏览器校验')
    expect(wrapper.text()).not.toContain('browser_required')
    expect(wrapper.text()).toContain('榜单小说 1.txt')
  })

  it('saves the generated fanqie import package as a txt file', async () => {
    const createObjectURL = vi.fn(() => 'blob:fanqie-txt')
    const revokeObjectURL = vi.fn()
    const click = vi.fn()
    Object.defineProperty(window.URL, 'createObjectURL', { value: createObjectURL, configurable: true })
    Object.defineProperty(window.URL, 'revokeObjectURL', { value: revokeObjectURL, configurable: true })
    const realCreateElement = document.createElement.bind(document)
    const createElement = vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
      const element = realCreateElement(tagName)
      if (tagName.toLowerCase() === 'a') {
        Object.defineProperty(element, 'click', { value: click, configurable: true })
      }
      return element
    })
    downloadFanqieBook.mockResolvedValue({
      title: 'Book 1',
      file_name: 'Book 1.txt',
      chapter_count: 30,
      text: 'Book 1\n\nChapter 1',
      status: 'completed',
      source: 'fanqie',
      decode_status: 'catalog_only',
      notes: ['personal backup only'],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-consent"]').setValue(true)
    await wrapper.find('[data-test="fanqie-download"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-test="fanqie-save-txt"]').trigger('click')

    expect(createObjectURL).toHaveBeenCalled()
    expect(click).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:fanqie-txt')
    createElement.mockRestore()
  })

  it('runs first-ten-chapter analysis for the selected book', async () => {
    analyzeFanqieBook.mockResolvedValue({
      title: '榜单小说 1 开篇分析',
      summary: '前三章钩子清晰',
      hooks: ['开局压迫'],
      actions: ['强化章尾钩子'],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-analyze"]').trigger('click')
    await flushPromises()

    expect(analyzeFanqieBook).toHaveBeenCalledWith(rankBooks[0])
    expect(wrapper.text()).toContain('榜单小说 1 开篇分析')
    expect(wrapper.text()).toContain('强化章尾钩子')
  })
})
