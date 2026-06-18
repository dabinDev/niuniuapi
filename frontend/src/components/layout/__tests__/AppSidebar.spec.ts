import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar studio navigation', () => {
  it('uses the new studio IA and keeps legacy script out of the visible menu', () => {
    expect(componentSource).toContain("path: '/studio/hotspot'")
    expect(componentSource).toContain("label: t('nav.studioHotspot')")
    expect(componentSource).toContain("path: '/studio/generate'")
    expect(componentSource).toContain("label: t('nav.studioGenerate')")
    expect(componentSource).not.toContain("path: '/studio/script', label: t('nav.studioScript')")
  })

  it('puts fanqie hotlist and cover generation before the other studio tools', () => {
    const navBlock = componentSource.match(/const studioNavItems = computed\(\(\): NavItem\[\] => \[([\s\S]*?)\]\)/)?.[1] ?? ''

    const fanqieIndex = navBlock.indexOf("path: '/studio/fanqie'")
    const coverIndex = navBlock.indexOf("path: '/studio/cover'")
    const worksIndex = navBlock.indexOf("path: '/studio/works'")
    const teardownIndex = navBlock.indexOf("path: '/studio/teardown'")
    const hotspotIndex = navBlock.indexOf("path: '/studio/hotspot'")
    const generateIndex = navBlock.indexOf("path: '/studio/generate'")

    expect(fanqieIndex).toBeGreaterThanOrEqual(0)
    expect(coverIndex).toBeGreaterThan(fanqieIndex)
    expect(worksIndex).toBeGreaterThan(coverIndex)
    expect(teardownIndex).toBeGreaterThan(coverIndex)
    expect(hotspotIndex).toBeGreaterThan(coverIndex)
    expect(generateIndex).toBeGreaterThan(coverIndex)
  })
})
