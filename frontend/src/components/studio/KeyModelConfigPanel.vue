<template>
  <div class="model-config-card rounded-2xl border border-orange-100 bg-white p-4 shadow-sm dark:border-dark-800 dark:bg-dark-900">
    <div class="mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div>
        <span class="config-kicker">Model Routing</span>
        <h3 class="mt-1 text-base font-black text-gray-950 dark:text-white">创作模型配置</h3>
        <p class="mt-1 max-w-3xl text-sm leading-6 text-gray-500 dark:text-dark-300">
          系统会优先读取用户创建的第一把密钥，自动填入最新生图模型和最新文案模型；你手动调整并测试保存后，系统会记住上次配置。
        </p>
      </div>
      <div class="config-status" data-test="auto-config-status">
        <span>{{ autoConfigStatus }}</span>
      </div>
    </div>

    <div v-if="!keyLoadFailed && !keys.length" class="empty-key-guide" data-test="empty-key-guide">
      <div class="empty-key-mark">01</div>
      <div>
        <strong>创建第一把密钥，系统会自动完成创作模型配置</strong>
        <p>
          创建密钥后刷新此页，系统会优先使用第一把密钥，自动选择最新的生图模型和文案模型。
          只有你手动调整并测试保存后，系统才会记住你的自定义选择。
        </p>
      </div>
    </div>

    <!-- 获取模型 + 选择 + 标记 -->
    <div class="control-rail">
      <div>
        <label class="mc-label">API 密钥</label>
        <select data-test="key" v-model="keyId" class="mc-input w-44" :disabled="!keys.length">
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
    <div class="mt-4 grid gap-3 sm:grid-cols-2">
      <div class="slot-card image-slot">
        <div class="flex items-center justify-between gap-2">
          <div>
            <div class="text-xs font-black uppercase tracking-wide text-orange-700 dark:text-orange-300">Image Slot</div>
            <div class="mt-0.5 text-sm font-bold text-gray-900 dark:text-white">生图模型</div>
          </div>
          <span class="slot-icon">IMG</span>
        </div>
        <div class="mt-3 flex flex-wrap items-center gap-2 text-sm">
          <span data-test="slot-image" class="slot-value">
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
      <div class="slot-card text-slot">
        <div class="flex items-center justify-between gap-2">
          <div>
            <div class="text-xs font-black uppercase tracking-wide text-emerald-700 dark:text-emerald-300">Copy Slot</div>
            <div class="mt-0.5 text-sm font-bold text-gray-900 dark:text-white">文案模型</div>
          </div>
          <span class="slot-icon">TXT</span>
        </div>
        <div class="mt-3 flex flex-wrap items-center gap-2 text-sm">
          <span data-test="slot-text" class="slot-value">
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

    <div class="mt-4 flex flex-wrap items-center gap-3">
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
const keyLoadFailed = ref(false)

const selectedKey = computed(() => keys.value.find((k) => k.id === keyId.value))
const autoConfigStatus = computed(() => {
  if (keyLoadFailed.value) return '密钥列表加载失败，稍后会重试'
  if (!keys.value.length) return '创建第一把密钥后自动配置'
  if (loadingModels.value) return '正在检查可用模型'
  if (slots.value.image && slots.value.text) return '已准备好创作模型'
  return '等待自动补齐模型'
})
function keyNameOf(id: number) {
  return keys.value.find((k) => k.id === id)?.name || `#${id}`
}

function modelRank(name: string) {
  const lower = name.toLowerCase()
  let score = 0
  const gptMajor = lower.match(/gpt[-_]?(\d+(?:\.\d+)?)/)
  if (gptMajor) score += Number(gptMajor[1]) * 100
  const versionParts = lower.match(/(?:^|[-_])(\d+)(?:\.(\d+))?(?:[-_]|$)/g) ?? []
  for (const part of versionParts) {
    const nums = part.match(/\d+/g) ?? []
    if (nums[0]) score += Number(nums[0]) * 10
    if (nums[1]) score += Number(nums[1])
  }
  if (lower.includes('latest')) score += 1000
  if (lower.includes('preview')) score += 20
  if (lower.includes('mini')) score -= 25
  return score
}

function chooseLatest(candidates: string[]) {
  return [...candidates].sort((a, b) => modelRank(b) - modelRank(a) || b.localeCompare(a))[0] || ''
}

function isImageModel(name: string) {
  const lower = name.toLowerCase()
  return lower.includes('image') || lower.includes('dall-e') || lower.includes('imagen') || lower.includes('flux')
}

function isTextModel(name: string) {
  const lower = name.toLowerCase()
  if (isImageModel(name)) return false
  return !['embedding', 'audio', 'tts', 'whisper', 'moderation', 'rerank'].some((token) => lower.includes(token))
}

async function autoConfigureMissingSlots() {
  if (!keys.value.length) return
  const needsImage = !slots.value.image
  const needsText = !slots.value.text
  if (!needsImage && !needsText) return

  const firstKey = keys.value[0]
  keyId.value = firstKey.id
  loadingModels.value = true
  error.value = ''
  try {
    const availableModels = await getKeyModels(firstKey.id)
    models.value = availableModels

    const nextSlots = { ...slots.value }
    const imageModel = needsImage ? chooseLatest(availableModels.filter(isImageModel)) : ''
    const textModel = needsText ? chooseLatest(availableModels.filter(isTextModel)) : ''

    if (needsImage && imageModel) {
      nextSlots.image = { api_key_id: firstKey.id, model: imageModel }
      tested.value.image = true
    }
    if (needsText && textModel) {
      nextSlots.text = { api_key_id: firstKey.id, model: textModel }
      tested.value.text = true
    }

    if (nextSlots.image !== slots.value.image || nextSlots.text !== slots.value.text) {
      slots.value = nextSlots
      const cfg: StudioModelConfig = {}
      if (nextSlots.image) cfg.image = nextSlots.image
      if (nextSlots.text) cfg.text = nextSlots.text
      await saveModelConfig(cfg)
      savedMsg.value = '已根据第一把密钥自动配置模型，可手动调整后测试保存。'
    }
  } catch (err) {
    const detail = extractApiErrorMessage(err, '')
    error.value = detail
      ? `自动配置模型失败：${detail}`
      : '自动配置模型失败：请检查第一把密钥是否可用，或手动获取模型列表。'
  } finally {
    loadingModels.value = false
  }
}

async function loadModels() {
  if (!selectedKey.value) return
  loadingModels.value = true
  error.value = ''
  models.value = []
  model.value = ''
  try {
    models.value = await getKeyModels(selectedKey.value.id)
  } catch (err) {
    const detail = extractApiErrorMessage(err, '')
    error.value = detail ? `获取模型列表失败：${detail}` : '获取模型列表失败：该密钥不可用或无权限。'
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
    keyLoadFailed.value = false
  } catch {
    keyLoadFailed.value = true
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

  await autoConfigureMissingSlots()
})
</script>

<style scoped>
.model-config-card {
  background:
    radial-gradient(circle at 10% 0%, rgba(232, 65, 46, 0.08), transparent 18rem),
    linear-gradient(135deg, #fffdfb, #fff);
}

.config-kicker {
  display: inline-flex;
  border: 1px solid rgba(232, 65, 46, 0.22);
  border-radius: 999px;
  padding: 0.18rem 0.55rem;
  color: #c8351f;
  font-size: 0.68rem;
  font-weight: 950;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.config-status {
  align-self: flex-start;
  border: 1px solid rgba(28, 117, 91, 0.18);
  border-radius: 999px;
  padding: 0.35rem 0.7rem;
  background: #f2fbf6;
  color: #1c755b;
  font-size: 0.78rem;
  font-weight: 850;
}

.control-rail {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 0.65rem;
  border: 1px solid #f0e5df;
  border-radius: 14px;
  padding: 0.85rem;
  background: rgba(255, 247, 243, 0.72);
}

.empty-key-guide {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0.8rem;
  margin-bottom: 0.85rem;
  border: 1px dashed rgba(232, 65, 46, 0.28);
  border-radius: 16px;
  padding: 0.9rem;
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.1), transparent 10rem),
    rgba(255, 250, 247, 0.86);
  color: #4d342b;
}

.empty-key-guide strong {
  display: block;
  color: #241a16;
  font-size: 0.92rem;
  font-weight: 950;
}

.empty-key-guide p {
  margin-top: 0.25rem;
  color: #7b6258;
  font-size: 0.82rem;
  line-height: 1.65;
}

.empty-key-mark {
  display: grid;
  width: 2.35rem;
  height: 2.35rem;
  place-items: center;
  border-radius: 999px;
  background: #e8412e;
  color: #fff;
  font-size: 0.75rem;
  font-weight: 950;
  box-shadow: 0 0.75rem 1.5rem rgba(232, 65, 46, 0.22);
}

.slot-card {
  border: 1px solid #f0e5df;
  border-radius: 16px;
  padding: 0.9rem;
  background: #fffdfb;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.78);
}

.image-slot {
  background:
    radial-gradient(circle at 100% 0%, rgba(232, 65, 46, 0.12), transparent 9rem),
    #fffdfb;
}

.text-slot {
  background:
    radial-gradient(circle at 100% 0%, rgba(28, 117, 91, 0.11), transparent 9rem),
    #fffdfb;
}

.slot-icon {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  place-items: center;
  border-radius: 0.75rem;
  background: #241a16;
  color: #fff;
  font-size: 0.68rem;
  font-weight: 950;
  letter-spacing: 0.02em;
}

.slot-value {
  min-width: 0;
  overflow-wrap: anywhere;
  color: rgb(17 24 39);
  font-weight: 850;
}

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

.dark .model-config-card {
  background:
    radial-gradient(circle at 10% 0%, rgba(232, 65, 46, 0.1), transparent 18rem),
    #11100f;
}

.dark .control-rail,
.dark .slot-card {
  border-color: rgb(49 39 32);
  background: rgb(23 19 17);
}

.dark .empty-key-guide {
  border-color: rgba(232, 65, 46, 0.32);
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.14), transparent 10rem),
    rgba(32, 24, 21, 0.9);
  color: #f5d8cf;
}

.dark .empty-key-guide strong {
  color: #fff7f4;
}

.dark .empty-key-guide p {
  color: #d9afa3;
}

.dark .config-status {
  border-color: rgba(28, 117, 91, 0.28);
  background: rgba(28, 117, 91, 0.12);
  color: #78d0b7;
}

.dark .slot-value {
  color: rgb(243 244 246);
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
