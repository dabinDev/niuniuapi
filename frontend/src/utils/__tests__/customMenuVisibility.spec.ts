import { describe, expect, it } from 'vitest'

import { customMenuVisibilityFromUserVisible, isCustomMenuUserVisible } from '../customMenuVisibility'

describe('custom menu user visibility mapping', () => {
  it('treats missing or invalid visibility as hidden from regular users', () => {
    expect(isCustomMenuUserVisible(undefined)).toBe(false)
    expect(isCustomMenuUserVisible('')).toBe(false)
    expect(isCustomMenuUserVisible('admin')).toBe(false)
    expect(isCustomMenuUserVisible('public')).toBe(false)
  })

  it('requires explicit user visibility for regular users', () => {
    expect(isCustomMenuUserVisible('user')).toBe(true)
  })

  it('stores unchecked user visibility as admin-only and checked as user-visible', () => {
    expect(customMenuVisibilityFromUserVisible(false)).toBe('admin')
    expect(customMenuVisibilityFromUserVisible(true)).toBe('user')
  })
})
