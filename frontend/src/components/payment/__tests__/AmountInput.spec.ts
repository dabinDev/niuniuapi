import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AmountInput from '../AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string | number>) => {
      const messages: Record<string, string> = {
        'payment.quickAmounts': '快捷金额',
        'payment.rechargePackages': '充值套餐',
        'payment.rechargePackageCredit': `到账 ${params?.credit}U`,
        'payment.rechargePackageUnitPrice': `约 ¥${params?.price}/U`,
        'payment.rechargePackageBestValue': '最划算',
        'payment.rechargePackageSave': `省 ${params?.percent}%`,
        'payment.customAmount': '自定义金额',
        'payment.enterAmount': '输入金额',
      }
      return messages[key] ?? key
    },
  }),
}))

function mountAmountInput(props = {}) {
  return mount(AmountInput, {
    props: {
      modelValue: null,
      ...props,
    },
  })
}

describe('AmountInput', () => {
  it('renders configured recharge tiers as package cards with value cues', () => {
    const wrapper = mountAmountInput({
      tiers: [
        { amount: 10, credit: 20 },
        { amount: 30, credit: 70 },
        { amount: 300, credit: 850 },
      ],
    })

    expect(wrapper.text()).toContain('充值套餐')
    expect(wrapper.text()).toContain('¥30')
    expect(wrapper.text()).toContain('到账 70U')
    expect(wrapper.text()).toContain('约 ¥0.43/U')
    expect(wrapper.text()).toContain('省 14%')
    expect(wrapper.text()).toContain('最划算')
  })
})
