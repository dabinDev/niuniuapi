import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import { STUDIO_FEATURES } from '@/utils/studioFeatures'

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
    expect(componentSource).toContain("import { STUDIO_FEATURES, isStudioFeatureVisible }")
    expect(componentSource).toContain('const studioNavItems = computed')
    expect(componentSource).toContain('.filter((feature) => isStudioFeatureVisible')
    expect(componentSource).toContain('path: feature.path')
    expect(componentSource).toContain('label: t(feature.navKey)')
    expect(componentSource).not.toContain("path: '/studio/script', label: t('nav.studioScript')")
  })

  it('puts fanqie hotlist and cover generation before the other studio tools', () => {
    const order = STUDIO_FEATURES.map((feature) => feature.path)
    const fanqieIndex = order.indexOf('/studio/fanqie')
    const coverIndex = order.indexOf('/studio/cover')
    const worksIndex = order.indexOf('/studio/works')
    const teardownIndex = order.indexOf('/studio/teardown')
    const hotspotIndex = order.indexOf('/studio/hotspot')
    const generateIndex = order.indexOf('/studio/generate')

    expect(fanqieIndex).toBeGreaterThanOrEqual(0)
    expect(coverIndex).toBeGreaterThan(fanqieIndex)
    expect(worksIndex).toBeGreaterThan(coverIndex)
    expect(teardownIndex).toBeGreaterThan(coverIndex)
    expect(hotspotIndex).toBeGreaterThan(coverIndex)
    expect(generateIndex).toBeGreaterThan(coverIndex)
  })
})
