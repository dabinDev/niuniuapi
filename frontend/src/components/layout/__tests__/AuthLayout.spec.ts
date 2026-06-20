import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AuthLayout.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AuthLayout brand panel', () => {
  it('uses a lanfanqie tomato seal fallback instead of an empty logo frame', () => {
    expect(componentSource).toContain('TOMATO')
    expect(componentSource).toContain('auth-tomato-mark')
    expect(componentSource).toContain('auth-seal-text')
    expect(componentSource).toContain('番茄创作质检台')
  })

  it('does not use the old API conversion platform subtitle as fallback copy', () => {
    expect(componentSource).not.toContain("|| 'Subscription to API Conversion Platform'")
    expect(componentSource).toContain('热榜、拆书、封面与创作生成一体化工作台')
  })

  it('normalizes legacy public setting subtitles to the current writer-workbench copy', () => {
    expect(componentSource).toContain('LEGACY_AUTH_SUBTITLES')
    expect(componentSource).toContain('normalizeAuthSubtitle')
    expect(componentSource).toContain('Subscription to API Conversion Platform')
  })
})
