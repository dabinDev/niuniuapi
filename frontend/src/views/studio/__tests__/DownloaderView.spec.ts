import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const getFanqieRank = vi.hoisted(() => vi.fn())
const searchFanqieBooks = vi.hoisted(() => vi.fn())
const downloadFanqieBook = vi.hoisted(() => vi.fn())
const analyzeFanqieBook = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/studio', () => ({ getFanqieRank, searchFanqieBooks, downloadFanqieBook, analyzeFanqieBook }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
}))

import DownloaderView from '../DownloaderView.vue'

const rankBooks = Array.from({ length: 30 }, (_, index) => ({
  id: String(1000 + index),
  rank: index + 1,
  title: `Rank Novel ${index + 1}`,
  author: `Author ${index + 1}`,
  category: index % 2 ? 'Urban Daily / System / Business' : 'Suspense Brainstorm / Mystery / Infinite Flow',
  status: index % 2 ? 'Serializing' : 'Completed',
  word_count: `${80 + index}0k words`,
  score: `${99 - index} heat`,
  description: `Selling point for rank novel ${index + 1}`,
  cover_url: `https://img.example.com/cover-${index + 1}.jpg`,
  source_url: `https://fanqienovel.com/page/${1000 + index}`,
  tags: ['hook', 'benchmark'],
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
    routerPush.mockReset()
    localStorage.clear()
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
    expect(wrapper.findAll('[data-test="fanqie-rank-row"]')).toHaveLength(30)
    expect(wrapper.find('[data-test="fanqie-channel-hot"]').classes()).toContain('active')
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
    expect(wrapper.find('[data-test="fanqie-sample-card"]').classes()).toContain('has-cover')
    expect(wrapper.find('[data-test="fanqie-sample-card"]').classes()).toContain('sample-card-compact')
    expect(wrapper.find('[data-test="fanqie-download"]').exists()).toBe(true)
  })

  it('uses a wide topic stat in the selected sample card so long genres do not collapse vertically', async () => {
    const wrapper = mountView()
    await flushPromises()

    const topic = wrapper.find('[data-test="fanqie-sample-topic"]')
    expect(topic.exists()).toBe(true)
    expect(topic.classes()).toContain('sample-profile-topic')
    expect(wrapper.find('[data-test="fanqie-sample-word-count"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-sample-status"]').exists()).toBe(true)
  })

  it('organizes the sample card around analysis, authorized backup, and downstream actions', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="fanqie-sample-panel"]').classes()).toContain('sample-panel-scroll')
    expect(wrapper.find('[data-test="fanqie-sample-fast-lane"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-sample-fast-lane"]').text()).toContain('授权下载')
    expect(wrapper.find('[data-test="fanqie-decision-strip"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-sample-ops"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-sample-ops"]').classes()).toContain('sample-ops-compact')
    expect(wrapper.find('[data-test="fanqie-sample-action-rail"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-backup-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-backup-card"]').classes()).toContain('backup-section-stack')
    expect(wrapper.find('[data-test="fanqie-backup-card"]').text()).toContain('完整小说 TXT')
    expect(wrapper.find('[data-test="fanqie-analyze"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-download"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-send-teardown"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-send-hotspot"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-send-generate"]').exists()).toBe(true)
  })

  it('uses designed cover placeholders when fanqie does not provide images', async () => {
    const noCoverBooks = rankBooks.map((book) => ({ ...book, cover_url: '' }))
    getFanqieRank.mockResolvedValueOnce({
      channel: 'hot',
      updated_at: '2026-06-17T00:00:00Z',
      source: 'fanqie-rank',
      books: noCoverBooks,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.findAll('[data-test="fanqie-cover-placeholder"]')).toHaveLength(30)
    expect(wrapper.find('[data-test="fanqie-cover-placeholder"] b').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-cover-placeholder"] small').text()).toContain('榜样')
    expect(wrapper.find('[data-test="fanqie-selected-cover-placeholder"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-selected-cover-placeholder"] b').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-selected-cover-placeholder"] small').text()).toContain('NO COVER')
    expect(wrapper.find('[data-test="fanqie-selected-cover-placeholder"]').text()).toContain('Rank')
  })

  it('separates sample analysis, downstream routing, and authorized full-book backup', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="fanqie-sample-action-rail"]').text()).toContain('前十章轻拆')
    expect(wrapper.find('[data-test="fanqie-sample-downstream-card"]').text()).toContain('下游加工')
    expect(wrapper.find('[data-test="fanqie-backup-card"]').text()).toContain('授权全本备份')
  })

  it('sends the selected hotlist sample to downstream studio tools', async () => {
    const setItem = vi.spyOn(Storage.prototype, 'setItem')
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-send-hotspot"]').trigger('click')

    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining('"target":"hotspot"'))
    expect(setItem).toHaveBeenCalledWith('studio_bridge_payload', expect.stringContaining(rankBooks[0].title))
    expect(routerPush).toHaveBeenCalledWith('/studio/hotspot')
    setItem.mockRestore()
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
        title: 'Ten Days Final',
        author: 'Search Author',
        category: 'Suspense',
        status: 'Completed',
        word_count: '2400k words',
        score: 'search hit',
        description: 'Named search result',
        cover_url: '',
        source_url: 'https://fanqienovel.com/page/42',
        tags: ['search'],
      },
    ])
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-search-input"]').setValue('Ten Days Final')
    await wrapper.find('[data-test="fanqie-search-submit"]').trigger('click')
    await flushPromises()

    expect(searchFanqieBooks).toHaveBeenCalledWith('Ten Days Final')
    expect(wrapper.text()).toContain('Ten Days Final')
  })

  it('shows backend search failure details', async () => {
    searchFanqieBooks.mockRejectedValue({ status: 502, message: 'fanqie captcha required' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-search-input"]').setValue('blocked novel')
    await wrapper.find('[data-test="fanqie-search-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('fanqie captcha required')
  })

  it('translates generic backend rank failures into a readable local troubleshooting card', async () => {
    getFanqieRank.mockRejectedValue({ response: { status: 500 } })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="fanqie-rank-empty-action"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-sample-empty-guide"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Request failed with status code 500')
  })

  it('downloads the selected book after the user confirms backup rights', async () => {
    downloadFanqieBook.mockResolvedValue({
      title: 'Rank Novel 1',
      file_name: 'Rank Novel 1.txt',
      chapter_count: 30,
      text: 'Book 1\n\nChapter 1',
      status: 'completed',
      source: 'fanqie',
      decode_status: 'browser_required',
      notes: ['personal backup only'],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-consent"]').setValue(true)
    await wrapper.find('[data-test="fanqie-download"]').trigger('click')
    await flushPromises()

    expect(downloadFanqieBook).toHaveBeenCalledWith(rankBooks[0], true)
    expect(wrapper.text()).not.toContain('browser_required')
    expect(wrapper.text()).toContain('Rank Novel 1.txt')
  })

  it('shows a long-running complete novel import notice while download is pending', async () => {
    let resolveDownload!: (value: unknown) => void
    downloadFanqieBook.mockReturnValue(new Promise((resolve) => {
      resolveDownload = resolve
    }))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-consent"]').setValue(true)
    await wrapper.find('[data-test="fanqie-download"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="fanqie-download-progress"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="fanqie-download-progress"]').text()).toContain('完整小说')

    resolveDownload({
      title: 'Rank Novel 1',
      file_name: 'Rank Novel 1.txt',
      chapter_count: 1,
      text: 'Book 1',
      status: 'completed',
      source: 'fanqie',
      decode_status: 'readable',
      notes: [],
    })
    await flushPromises()

    expect(wrapper.find('[data-test="fanqie-download-progress"]').exists()).toBe(false)
  })

  it('translates complete novel download timeouts into actionable guidance', async () => {
    downloadFanqieBook.mockRejectedValue({ code: 'ECONNABORTED', message: 'timeout of 600000ms exceeded' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-consent"]').setValue(true)
    await wrapper.find('[data-test="fanqie-download"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('完整小说下载耗时较长')
    expect(wrapper.text()).not.toContain('timeout of 600000ms exceeded')
  })

  it('keeps analysis scoped to first ten chapters while download imports the complete novel txt', async () => {
    downloadFanqieBook.mockResolvedValue({
      title: 'Rank Novel 1',
      file_name: 'Rank Novel 1.txt',
      chapter_count: 328,
      text: 'Book 1\n\nChapter 1\n...\nChapter 328',
      status: 'completed',
      source: 'fanqie',
      decode_status: 'readable',
      notes: ['complete TXT generated'],
    })
    analyzeFanqieBook.mockResolvedValue({
      title: 'Rank Novel 1 opening analysis',
      summary: 'Only introduction and first ten chapters are analyzed.',
      hooks: ['opening pressure'],
      actions: ['strengthen chapter 10 hook'],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-analyze"]').trigger('click')
    await flushPromises()
    expect(analyzeFanqieBook).toHaveBeenCalledWith(rankBooks[0])
    expect(downloadFanqieBook).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Only introduction and first ten chapters are analyzed.')

    await wrapper.find('[data-test="fanqie-consent"]').setValue(true)
    await wrapper.find('[data-test="fanqie-download"]').trigger('click')
    await flushPromises()
    expect(downloadFanqieBook).toHaveBeenCalledWith(rankBooks[0], true)
    expect(wrapper.text()).toContain('328')
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
      title: 'Rank Novel 1 opening analysis',
      summary: 'Opening hooks are clear.',
      hooks: ['fast opening'],
      actions: ['strengthen chapter ending hook'],
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="fanqie-analyze"]').trigger('click')
    await flushPromises()

    expect(analyzeFanqieBook).toHaveBeenCalledWith(rankBooks[0])
    expect(wrapper.text()).toContain('Rank Novel 1 opening analysis')
    expect(wrapper.text()).toContain('strengthen chapter ending hook')
  })
})
