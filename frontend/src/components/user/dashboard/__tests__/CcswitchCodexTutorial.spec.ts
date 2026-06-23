import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import CcswitchCodexTutorial from '../CcswitchCodexTutorial.vue'

describe('CcswitchCodexTutorial', () => {
  it('documents CCSWITCH import, Codex desktop, Codex CLI, and manual config paths', () => {
    const wrapper = mount(CcswitchCodexTutorial, {
      global: {
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>',
          },
        },
      },
    })

    const text = wrapper.text()
    expect(text).toContain('CCSWITCH 一键导入')
    expect(text).toContain('Codex 客户端')
    expect(text).toContain('Codex CLI')
    expect(text).toContain('手动配置 config.toml')
    expect(text).toContain('auth.json')
    expect(wrapper.find('#ccswitch-codex-tutorial').exists()).toBe(true)
    expect(wrapper.findAll('img[alt*="真实截图"]')).toHaveLength(4)
  })
})
