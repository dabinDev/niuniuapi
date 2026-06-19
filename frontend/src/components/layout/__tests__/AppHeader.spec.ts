import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AppHeader repository links', () => {
  it('points admin GitHub menu entry to the maintained niuniuapi repository', () => {
    expect(componentSource).toContain('https://github.com/dabinDev/niuniuapi')
    expect(componentSource).not.toContain('https://github.com/Wei-Shaw/sub2api')
  })
})
