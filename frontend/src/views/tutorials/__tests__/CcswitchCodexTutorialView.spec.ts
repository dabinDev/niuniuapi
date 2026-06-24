import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import CcswitchCodexTutorialView from '../CcswitchCodexTutorialView.vue'

describe('CcswitchCodexTutorialView', () => {
  function mountView() {
    return mount(CcswitchCodexTutorialView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          RouterLink: {
            props: ['to'],
            template: '<a :href="to"><slot /></a>',
          },
        },
      },
    })
  }

  it('renders a full written tutorial instead of an image-only gallery', () => {
    const wrapper = mountView()
    const text = wrapper.text()

    expect(text).toContain('CCSWITCH / Codex 使用教程')
    expect(text).toContain('适用场景')
    expect(text).toContain('开始前先准备')
    expect(text).toContain('第一步：在番茄创建并确认 API 密钥')
    expect(text).toContain('第二步：一键导入到 CCSWITCH')
    expect(text).toContain('第三步：配置 Codex 客户端或 Codex CLI')
    expect(text).toContain('第四步：手动写入 config.toml 与 auth.json')
    expect(text).toContain('常见问题排查')
    expect(text).toContain('获取模型列表失败：Upstream model list request failed with HTTP 401')
    expect(text).toContain('Insufficient account balance')
  })

  it('embeds the four screenshots inside the corresponding tutorial sections', () => {
    const wrapper = mountView()
    const images = wrapper.findAll('figure img')

    expect(images).toHaveLength(4)
    expect(images.map((image) => image.attributes('src'))).toEqual([
      '/tutorial/ccswitch/01-keys-import-ccswitch.png',
      '/tutorial/ccswitch/02-use-key-codex-client.png',
      '/tutorial/ccswitch/03-codex-cli-config.png',
      '/tutorial/ccswitch/04-manual-config-files.png',
    ])
    expect(images.every((image) => image.attributes('alt')?.includes('教程截图'))).toBe(true)
  })

  it('keeps direct action links available from the tutorial page', () => {
    const wrapper = mountView()

    expect(wrapper.find('a[href="/keys"]').text()).toContain('去 API 密钥页')
    expect(wrapper.find('a[href="/dashboard"]').text()).toContain('返回创作台')
  })
})
