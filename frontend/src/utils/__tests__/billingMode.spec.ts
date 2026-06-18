import { describe, expect, it } from 'vitest'

import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_TOKEN,
  getDisplayBillingMode,
  isImageUsage,
} from '@/utils/billingMode'

describe('billingMode', () => {
  it('treats legacy image rows without billing_mode as image usage', () => {
    const row = {
      billing_mode: null,
      image_count: 2,
    }

    expect(isImageUsage(row)).toBe(true)
    expect(getDisplayBillingMode(row)).toBe(BILLING_MODE_IMAGE)
  })

  it('keeps explicit token billing as token even when image_count exists', () => {
    const row = {
      billing_mode: BILLING_MODE_TOKEN,
      image_count: 2,
    }

    expect(isImageUsage(row)).toBe(false)
    expect(getDisplayBillingMode(row)).toBe(BILLING_MODE_TOKEN)
  })
})
