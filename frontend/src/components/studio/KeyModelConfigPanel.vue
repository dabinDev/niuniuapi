<template>
  <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-800 dark:bg-dark-900">
    <div class="mb-3 flex items-center gap-2">
      <h3 class="text-sm font-bold text-gray-900 dark:text-white">创作模型配置</h3>
      <span class="text-xs text-gray-400">用某把密钥获取模型，标记为生图 / 文案模型，测试通过后保存</span>
    </div>

    <!-- 获取模型 + 选择 + 标记 -->
    <div class="flex flex-wrap items-end gap-2">
      <div>
        <label class="mc-label">API 密钥</label>
        <select data-test="key" v-model="keyId" class="mc-input w-44">
          <option value="" disabled>选择密钥</option>
          <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name }}</option>
        </select>
      </div>
      <button data-test="fetch" type="button" class="btn btn-secondary" :disabled="keyId === '' || loadingModels" @click="loadModels">
        {{ loadingModels ? '获取中…' : '获取模型列表' }}
      </button>
      <div>
        <label class="mc-label">模型</label>
        <select data-test="model" v-model="model" class="mc-input w-56" :disabled="!models.length">
          <option value="" disabled>{{ models.length ? '选择模型' : '先获取模型列表' }}</option>
          <option v-for="m in models" :key="m" :value="m">{{ m }}</option>
        </select>
      </div>
      <button data-test="set-image" type="button" class="btn btn-secondary" :disabled="!model" @click="assign('image')">设为生图模型</button>
      <button data-test="set-text" type="button" class="btn btn-secondary" :disabled="!model" @click="assign('text')">设为文案模型</button>
    </div>

    <!-- 两个槽位 -->
    <div class="mt-4 grid gap-2 sm:grid-cols-2">
      <div class="rounded-lg border border-gray-100 p-3 dark:border-dark-800">
        <div class="text-xs font-semibold text-gray-500 dark:text-gray-400">🖼️ 生图模型</div>
        <div class="mt-1 flex items-center gap-2 text-sm">
          <span data-test="slot-image" class="font-semibold text-gray-900 dark:text-white">
            {{ slots.image ? `${slots.image.model} (${keyNameOf(slots.image.api_key_id)})` : '未设置' }}
          </span>
          <template v-if="slots.image">
            <button data-test="test-image" type="button" class="btn btn-secondary btn-xs" :disabled="testing.image" @click="testSlot('image')">
              {{ testing.image ? '测试中…' : '测试' }}
            </button>
            <span v-if="tested.image" class="text-green-600 dark:text-green-400">✅</span>
          </template>
        </div>
      </div>
      <div class="rounded-lg border border-gray-100 p-3 dark:border-dark-800">
        <div class="text-xs font-semibold text-gray-500 dark:text-gray-400">📝 文案模型</div>
        <div class="mt-1 flex items-center gap-2 text-sm">
          <span data-test="slot-text" class="font-semibold text-gray-900 dark:text-white">
            {{ slots.text ? `${slots.text.model} (${keyNameOf(slots.text.api_key_id)})` : '未设置' }}
          </span>
          <template v-if="slots.text">
            <button data-test="test-text" type="button" class="btn btn-secondary btn-xs" :disabled="testing.text" @click="testSlot('text')">
              {{ testing.text ? '测试中…' : '测试' }}
            </button>
            <span v-if="tested.text" class="text-green-600 dark:text-green-400">✅</span>
          </template>
        </div>
      </div>
    </div>

    <div class="mt-3 flex items-center gap-3">
      <button data-test="save" type="button" class="btn btn-primary" :disabled="!canSave" @click="save">
        {{ saving ? '保存中…' : '保存' }}
      </button>
      <span v-if="error" class="text-sm text-red-500">{{ error }}</span>
      <span v-else-if="savedMsg" class="text-sm text-green-600 dark:text-green-400">{{ savedMsg }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { list as listKeys } from '@/api/keys'
import { getKeyModels, getModelConfig, saveModelConfig, testModelSlot, type ModelSlot, type StudioModelConfig } from '@/api/studio'
import type { ApiKey } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

type SlotType = 'image' | 'text'

const keys = ref<ApiKey[]>([])
const keyId = ref<number | ''>('')
const models = ref<string[]>([])
const model = ref('')
const loadingModels = ref(false)
const slots = ref<{ image: ModelSlot | null; text: ModelSlot | null }>({ image: null, text: null })
const tested = ref<{ image: boolean; text: boolean }>({ image: false, text: false })
const testing = ref<{ image: boolean; text: boolean }>({ image: false, text: false })
const saving = ref(false)
const error = ref('')
const savedMsg = ref('')

const selectedKey = computed(() => keys.value.find((k) => k.id === keyId.value))
function keyNameOf(id: number) {
  return keys.value.find((k) => k.id === id)?.name || `#${id}`
}

async function loadModels() {
  if (!selectedKey.value) return
  loadingModels.value = true
  error.value = ''
  models.value = []
  model.value = ''
  try {
    models.value = await getKeyModels(selectedKey.value.id)
  } catch {
    error.value = '获取模型列表失败：该密钥不可用或无权限。'
  } finally {
    loadingModels.value = false
  }
}

function assign(type: SlotType) {
  if (keyId.value === '' || !model.value) return
  slots.value[type] = { api_key_id: keyId.value as number, model: model.value }
  tested.value[type] = false
  savedMsg.value = ''
}

async function testSlot(type: SlotType) {
  const s = slots.value[type]
  if (!s) return
  testing.value[type] = true
  error.value = ''
  try {
    await testModelSlot(type, s.api_key_id, s.model)
    tested.value[type] = true
  } catch (err) {
    tested.value[type] = false
    const detail = extractApiErrorMessage(err, '')
    error.value = detail
      ? `${type === 'image' ? '生图' : '文案'}测试未通过：${detail}`
      : `${type === 'image' ? '生图' : '文案'}测试未通过。`
  } finally {
    testing.value[type] = false
  }
}

const canSave = computed(() => {
  const hasAny = !!(slots.value.image || slots.value.text)
  const imgOk = !slots.value.image || tested.value.image
  const txtOk = !slots.value.text || tested.value.text
  return hasAny && imgOk && txtOk && !saving.value
})

async function save() {
  if (!canSave.value) return
  saving.value = true
  error.value = ''
  try {
    const cfg: StudioModelConfig = {}
    if (slots.value.image) cfg.image = slots.value.image
    if (slots.value.text) cfg.text = slots.value.text
    await saveModelConfig(cfg)
    savedMsg.value = '已保存。'
  } catch {
    error.value = '保存失败。'
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    const res = await listKeys(1, 100)
    keys.value = res.items
  } catch {
    /* 忽略 */
  }
  try {
    const cfg = await getModelConfig()
    slots.value.image = cfg.image || null
    slots.value.text = cfg.text || null
    tested.value.image = !!cfg.image
    tested.value.text = !!cfg.text
  } catch {
    /* 忽略 */
  }
})
</script>

<style scoped>
.mc-label {
  display: block;
  margin-bottom: 0.25rem;
  font-size: 12px;
  font-weight: 700;
  color: rgb(107 114 128);
}

.dark .mc-label {
  color: rgb(156 163 175);
}

.mc-input {
  border: 1px solid rgb(209 213 219);
  border-radius: 8px;
  background: #fff;
  padding: 0.45rem 0.55rem;
  font-size: 14px;
  color: rgb(17 24 39);
}

.dark .mc-input {
  border-color: rgb(55 65 81);
  background: rgb(17 24 39);
  color: rgb(243 244 246);
}

.btn-xs {
  padding: 0.2rem 0.6rem;
  font-size: 12px;
}
</style>
