import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import CcswitchCodexTutorial from '../CcswitchCodexTutorial.vue'

describe('CcswitchCodexTutorial', () => {
  it('links to the full CCSWITCH and Codex tutorial from the dashboard', () => {
    const wrapper = mount(CcswitchCodexTutorial, {
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="to"><slot /></a>',
          },
        },
      },
    })

    const text = wrapper.text()
    expect(text).toContain('先看完整教程，再接入本地工具')
    expect(text).toContain('一键导入 CCSWITCH')
    expect(text).toContain('Codex 客户端')
    expect(text).toContain('Codex CLI')
    expect(text).toContain('常见错误排查')
    expect(wrapper.find('#ccswitch-codex-tutorial').exists()).toBe(true)
    expect(wrapper.find('a[href="/tutorials/ccswitch-codex"]').text()).toContain('查看完整教程')
    expect(wrapper.find('a[href="/keys"]').text()).toContain('去 API 密钥页')
    expect(wrapper.findAll('img')).toHaveLength(0)
  })
})
