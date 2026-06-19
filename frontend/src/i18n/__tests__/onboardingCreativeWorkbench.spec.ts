import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('creative workbench onboarding copy', () => {
  it('frames the admin welcome tour around the LanFanQie writing workflow', () => {
    const zhCopy = zh.onboarding.admin.welcome.description
    const enCopy = en.onboarding.admin.welcome.description

    expect(zhCopy).toContain('创作工作台')
    expect(zhCopy).toContain('番茄热榜')
    expect(zhCopy).toContain('封面生成')
    expect(zhCopy).toContain('拆书诊断')
    expect(zhCopy).not.toContain('AI 服务中转平台')
    expect(zhCopy).not.toContain('账号池')
    expect(zhCopy).not.toContain('计费管理')

    expect(enCopy).toContain('writing workbench')
    expect(enCopy).toContain('Fanqie hotlist')
    expect(enCopy).toContain('cover generation')
    expect(enCopy).toContain('chapter diagnosis')
    expect(enCopy).not.toContain('AI service gateway')
    expect(enCopy).not.toContain('Account Pool')
    expect(enCopy).not.toContain('Billing Control')
  })
})
