<template>
  <div class="space-y-4">
    <!-- Quick Amount Buttons -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ hasTierPackages ? t('payment.rechargePackages') : t('payment.quickAmounts') }}
      </label>
      <div v-if="hasTierPackages" class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <button
          v-for="tier in filteredTiers"
          :key="tier.amount"
          type="button"
          :class="[
            'min-h-[112px] rounded-lg border-2 px-4 py-3 text-left transition-colors',
            modelValue === tier.amount
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/40 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
          ]"
          @click="selectAmount(tier.amount)"
        >
          <span class="flex items-start justify-between gap-2">
            <span class="text-lg font-bold leading-none">¥{{ formatAmount(tier.amount) }}</span>
            <span
              v-if="isBestTier(tier)"
              class="shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-semibold text-amber-700 dark:bg-amber-900/40 dark:text-amber-300"
            >
              {{ t('payment.rechargePackageBestValue') }}
            </span>
            <span
              v-else-if="savingPercent(tier) > 0"
              class="shrink-0 rounded-full bg-green-100 px-2 py-0.5 text-[11px] font-semibold text-green-700 dark:bg-green-900/40 dark:text-green-300"
            >
              {{ t('payment.rechargePackageSave', { percent: savingPercent(tier) }) }}
            </span>
          </span>
          <span class="mt-3 block text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('payment.rechargePackageCredit', { credit: formatAmount(tier.credit) }) }}
          </span>
          <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
            {{ t('payment.rechargePackageUnitPrice', { price: unitPrice(tier) }) }}
          </span>
        </button>
      </div>
      <div v-else class="grid grid-cols-3 gap-2">
        <button
          v-for="amt in filteredAmounts"
          :key="amt"
          type="button"
          :class="[
            'rounded-lg border-2 px-4 py-3 text-center font-medium transition-colors',
            modelValue === amt
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/40 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
          ]"
          @click="selectAmount(amt)"
        >
          {{ amt }}
        </button>
      </div>
    </div>

    <!-- Custom Amount Input -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-dark-500">
          $
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="input w-full py-3 pl-8 pr-4"
          @input="handleInput"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'

interface RechargeTier {
  amount: number
  credit: number
}

const props = withDefaults(defineProps<{
  amounts?: number[]
  tiers?: RechargeTier[]
  modelValue: number | null
  min?: number
  max?: number
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)
const filteredTiers = computed(() =>
  (props.tiers ?? [])
    .map(tier => ({
      amount: roundAmount(Number(tier.amount)),
      credit: roundAmount(Number(tier.credit)),
    }))
    .filter(tier =>
      tier.amount > 0
      && tier.credit > 0
      && (props.min <= 0 || tier.amount >= props.min)
      && (props.max <= 0 || tier.amount <= props.max),
    )
    .sort((a, b) => a.amount - b.amount)
)
const hasTierPackages = computed(() => filteredTiers.value.length > 0)
const baselineUnitPrice = computed(() => {
  const first = filteredTiers.value[0]
  return first ? first.amount / first.credit : 0
})
const bestTierUnitPrice = computed(() => {
  if (!filteredTiers.value.length) return 0
  return Math.min(...filteredTiers.value.map(tier => tier.amount / tier.credit))
})

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function roundAmount(value: number): number {
  if (!Number.isFinite(value)) return 0
  return Math.round(value * 100) / 100
}

function formatAmount(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(2)
}

function unitPrice(tier: RechargeTier): string {
  if (tier.credit <= 0) return '0.00'
  return (tier.amount / tier.credit).toFixed(2)
}

function savingPercent(tier: RechargeTier): number {
  if (baselineUnitPrice.value <= 0 || tier.credit <= 0) return 0
  const currentUnitPrice = tier.amount / tier.credit
  if (currentUnitPrice >= baselineUnitPrice.value) return 0
  return Math.round((1 - currentUnitPrice / baselineUnitPrice.value) * 100)
}

function isBestTier(tier: RechargeTier): boolean {
  if (bestTierUnitPrice.value <= 0 || tier.credit <= 0) return false
  return tier.amount / tier.credit === bestTierUnitPrice.value
}

function handleInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  if (!AMOUNT_PATTERN.test(val)) return
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>
