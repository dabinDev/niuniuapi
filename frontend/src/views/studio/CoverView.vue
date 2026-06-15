<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl">
      <header class="mb-5">
        <span class="inline-block rounded-full bg-primary-50 px-3 py-1 text-xs font-bold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
          P0 · 创作台
        </span>
        <h1 class="mt-2 text-2xl font-black text-gray-900 dark:text-white sm:text-3xl">小说封面生成</h1>
        <p class="mt-1 text-sm leading-7 text-gray-500 dark:text-gray-400">
          自定义提示词，或用小说简介 + 主角信息，一键生成多版封面。
        </p>
      </header>

      <!-- 生图模型状态：在「API 密钥」页配置 -->
      <div
        class="mb-4 rounded-xl border px-4 py-3 text-sm"
        :class="configured
          ? 'border-gray-200 bg-white text-gray-600 dark:border-dark-800 dark:bg-dark-900 dark:text-gray-300'
          : 'border-dashed border-primary-300 bg-primary-50 text-primary-700 dark:border-primary-700 dark:bg-primary-900/20 dark:text-primary-300'"
      >
        <template v-if="configured">
          🖼️ 生图模型：<span class="font-semibold">{{ imageModel }}</span>
          <router-link to="/keys" class="ml-2 font-semibold text-primary-600 hover:underline dark:text-primary-300">在「API 密钥」页修改</router-link>
        </template>
        <template v-else>
          还没配置生图模型。请先到
          <router-link to="/keys" class="font-semibold underline">API 密钥页</router-link>
          选密钥 → 获取模型 → 设为生图模型 → 测试保存。
        </template>
      </div>

      <!-- 入口切换 -->
      <div class="mb-4 inline-flex rounded-lg border border-gray-200 bg-white p-1 dark:border-dark-700 dark:bg-dark-900">
        <button
          v-for="m in modes"
          :key="m.value"
          type="button"
          class="rounded-md px-4 py-1.5 text-sm font-semibold transition"
          :class="mode === m.value
            ? 'bg-primary-600 text-white'
            : 'text-gray-600 hover:text-primary-600 dark:text-gray-300'"
          @click="mode = m.value"
        >
          {{ m.label }}
        </button>
      </div>

      <div class="grid gap-4 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <!-- 表单 -->
        <section class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-800 dark:bg-dark-900">
          <template v-if="mode === 'custom'">
            <label class="form-label" for="cv-prompt">提示词</label>
            <textarea id="cv-prompt" v-model="prompt" class="form-input" rows="6" placeholder="描述你想要的画面……"></textarea>
            <label class="form-label" for="cv-ref">参考图链接（选填）</label>
            <input id="cv-ref" v-model="refImage" class="form-input" type="text" placeholder="https://… 风格/构图参考" />
          </template>

          <template v-else>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="form-label" for="cv-title">书名（选填）</label>
                <input id="cv-title" v-model="novelTitle" class="form-input" type="text" placeholder="北境书塔" />
              </div>
              <div>
                <label class="form-label" for="cv-genre">题材/画风（选填）</label>
                <input id="cv-genre" v-model="genre" class="form-input" type="text" placeholder="玄幻 / 国风…" />
              </div>
            </div>
            <label class="form-label" for="cv-synopsis">简介</label>
            <textarea id="cv-synopsis" v-model="synopsis" class="form-input" rows="3" placeholder="一句话或一段简介……"></textarea>
            <label class="form-label" for="cv-protagonist">主角信息（外貌/气质）</label>
            <textarea id="cv-protagonist" v-model="protagonist" class="form-input" rows="2" placeholder="林见微，外冷内热，记忆缺口……"></textarea>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="form-label" for="cv-mood">情绪基调（选填）</label>
                <input id="cv-mood" v-model="mood" class="form-input" type="text" placeholder="清冷 / 热血…" />
              </div>
              <div>
                <label class="form-label" for="cv-scene">关键场景/意象（选填）</label>
                <input id="cv-scene" v-model="keyScene" class="form-input" type="text" placeholder="雨夜书塔…" />
              </div>
            </div>
            <label class="form-label" for="cv-cover-title">封面标题文字（选填）</label>
            <input id="cv-cover-title" v-model="coverTitle" class="form-input" type="text" placeholder="封面上要显示的书名" />
          </template>

          <div class="mt-3 grid grid-cols-2 gap-3">
            <div>
              <label class="form-label" for="cv-size">尺寸</label>
              <select id="cv-size" v-model="size" class="form-input">
                <option v-for="s in sizeOptions" :key="s" :value="s">{{ s }}</option>
              </select>
            </div>
            <div>
              <label class="form-label" for="cv-count">数量</label>
              <select id="cv-count" v-model.number="count" class="form-input">
                <option v-for="n in [1, 2, 3, 4]" :key="n" :value="n">{{ n }}</option>
              </select>
            </div>
          </div>

          <button type="button" class="submit-btn" :disabled="!canSubmit" @click="submit">
            {{ loading ? '正在生成…' : '生成封面' }}
          </button>

          <p v-if="errorMsg" class="mt-3 text-sm text-red-500">{{ errorMsg }}</p>
          <div v-if="backendPending" class="mt-3 rounded-lg border border-dashed border-primary-300 bg-primary-50 px-3 py-2 text-sm text-primary-700 dark:border-primary-700 dark:bg-primary-900/20 dark:text-primary-300">
            后端封面生成服务正在开发中，接口就绪后即可在此实时出图。当前页面与调用流程已可用。
          </div>
        </section>

        <!-- 结果 -->
        <section class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-800 dark:bg-dark-900">
          <div v-if="loading" class="flex min-h-[280px] flex-col items-center justify-center gap-3 text-gray-400">
            <div class="spinner"></div>
            <p>正在生成候选封面…</p>
          </div>
          <div v-else-if="covers.length" class="grid grid-cols-2 gap-3">
            <figure v-for="c in covers" :key="c.id" class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-800" style="aspect-ratio: 3 / 4">
              <img :src="c.url" :alt="novelTitle || 'cover'" class="h-full w-full object-cover" />
            </figure>
          </div>
          <div v-else class="flex min-h-[280px] flex-col items-center justify-center gap-3 text-center text-gray-400">
            <div class="text-5xl">🖼️</div>
            <p>填好左侧，点「生成封面」，候选会出现在这里。</p>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { buildNovelCoverPayload, generateCover, getModelConfig, type CoverImage } from '@/api/studio'

const modes = [
  { value: 'custom' as const, label: '自定义' },
  { value: 'novel' as const, label: '小说驱动' },
]
const mode = ref<'custom' | 'novel'>('novel')

// 自定义
const prompt = ref('')
const refImage = ref('')
// 小说驱动
const novelTitle = ref('')
const synopsis = ref('')
const protagonist = ref('')
const genre = ref('')
const mood = ref('')
const keyScene = ref('')
const coverTitle = ref('')
// 通用
const sizeOptions = ['1024x1536', '1024x1024', '1536x1024']
const size = ref('1024x1536')
const count = ref(2)

const loading = ref(false)
const covers = ref<CoverImage[]>([])
const errorMsg = ref('')
const backendPending = ref(false)
const configured = ref(false)
const imageModel = ref('')

onMounted(async () => {
  try {
    const cfg = await getModelConfig()
    if (cfg.image) {
      configured.value = true
      imageModel.value = cfg.image.model
    }
  } catch {
    /* 忽略：未配置 */
  }
})

const canSubmit = computed(() => {
  if (loading.value || !configured.value) return false
  return mode.value === 'custom'
    ? prompt.value.trim().length > 0
    : synopsis.value.trim().length > 0 && protagonist.value.trim().length > 0
})

async function submit() {
  if (!canSubmit.value) return
  loading.value = true
  errorMsg.value = ''
  backendPending.value = false
  covers.value = []
  try {
    const payload =
      mode.value === 'custom'
        ? { mode: 'custom' as const, prompt: prompt.value.trim(), ref_image: refImage.value.trim() || undefined, size: size.value, count: count.value }
        : buildNovelCoverPayload({
            title: novelTitle.value,
            synopsis: synopsis.value,
            protagonist: protagonist.value,
            genre: genre.value,
            mood: mood.value,
            keyScene: keyScene.value,
            coverTitle: coverTitle.value,
            size: size.value,
            count: count.value,
          })
    const result = await generateCover(payload)
    covers.value = result.covers
  } catch (err: unknown) {
    const status = (err as { response?: { status?: number } })?.response?.status
    if (status === 404 || status === undefined) {
      backendPending.value = true
    } else {
      errorMsg.value = '生成失败，请稍后重试。'
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.form-label {
  display: block;
  margin: 0.75rem 0 0.35rem;
  font-size: 13px;
  font-weight: 700;
  color: rgb(75 85 99);
}

.dark .form-label {
  color: rgb(209 213 219);
}

.form-input {
  width: 100%;
  border: 1px solid rgb(209 213 219);
  border-radius: 10px;
  background: #fff;
  padding: 0.55rem 0.7rem;
  font-size: 14px;
  color: rgb(17 24 39);
  resize: vertical;
}

.dark .form-input {
  border-color: rgb(55 65 81);
  background: rgb(17 24 39);
  color: rgb(243 244 246);
}

.form-input:focus {
  outline: none;
  border-color: rgb(var(--tw-color-primary-500, 232 65 46));
}

.submit-btn {
  margin-top: 1.1rem;
  width: 100%;
  border-radius: 10px;
  padding: 0.7rem;
  font-size: 15px;
  font-weight: 800;
  color: #fff;
  background: rgb(220 56 31);
  transition: opacity 0.15s ease;
}

.submit-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid rgba(220, 56, 31, 0.25);
  border-top-color: rgb(220 56 31);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
