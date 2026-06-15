import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import CoverImageViewer from '../CoverImageViewer.vue'

describe('CoverImageViewer', () => {
  it('zooms with mouse wheel and keeps navigation, drag, download, and close controls usable', async () => {
    const wrapper = mount(CoverImageViewer, {
      props: {
        images: [
          { id: 'first', url: 'first.png' },
          { id: 'second', url: 'second.png' },
        ],
        initialIndex: 0,
        title: 'Wheel Book',
      },
    })

    expect(wrapper.find('[data-test="cover-viewer-counter"]').text()).toContain('1 / 2')
    expect(wrapper.find('[data-test="cover-download"]').attributes('href')).toBe('first.png')

    await wrapper.find('[data-test="cover-viewer-stage"]').trigger('wheel', { deltaY: -120 })
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('style')).toContain('scale(1.25)')

    await wrapper.find('[data-test="cover-viewer-stage"]').trigger('wheel', { deltaY: 120 })
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('style')).toContain('scale(1)')

    await wrapper.find('[data-test="cover-viewer-stage"]').trigger('mousedown', { clientX: 20, clientY: 40 })
    window.dispatchEvent(new MouseEvent('mousemove', { clientX: 44, clientY: 68 }))
    window.dispatchEvent(new MouseEvent('mouseup'))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('style')).toContain('translate3d(24px, 28px')

    await wrapper.find('[data-test="cover-next"]').trigger('click')
    expect(wrapper.find('[data-test="cover-viewer-counter"]').text()).toContain('2 / 2')
    expect(wrapper.find('[data-test="cover-viewer-image"]').attributes('src')).toBe('second.png')

    await wrapper.find('[data-test="cover-close"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
