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
            <label class="form-label">参考图（选填）</label>
            <ImageUpload
              v-model="refImage"
              upload-label="上传参考图"
              remove-label="移除参考图"
              hint="支持 PNG、JPG、WebP，最大 20MB；用于自定义封面的构图或风格参考。"
              :max-size="20 * 1024 * 1024"
            />
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
            <p>{{ loadingText }}</p>
          </div>
          <div v-else-if="covers.length" class="grid grid-cols-2 gap-3">
            <figure v-for="(c, index) in covers" :key="c.id" class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-800" style="aspect-ratio: 3 / 4">
              <button
                type="button"
                class="cover-thumb"
                :aria-label="`查看封面 ${index + 1}`"
                :data-test="`cover-thumb-${index}`"
                @click="openCoverViewer(index)"
              >
                <img :src="c.url" :alt="novelTitle || 'cover'" class="h-full w-full object-cover" />
              </button>
            </figure>
          </div>
          <div v-else class="flex min-h-[280px] flex-col items-center justify-center gap-3 text-center text-gray-400">
            <div class="text-5xl">🖼️</div>
            <p>填好左侧，点「生成封面」，候选会出现在这里。</p>
          </div>
        </section>
      </div>

      <section v-if="coverHistory.length" class="cover-history-section">
        <div class="cover-history-head">
          <div>
            <h2>创作时间轴</h2>
            <p>最近生成的封面，按时间从新到旧排列。</p>
          </div>
          <span v-if="historyLoading">同步中</span>
        </div>
        <div class="cover-history-track" data-test="cover-history-timeline">
          <button
            v-for="(item, index) in coverHistory"
            :key="item.id"
            type="button"
            class="cover-history-card"
            data-test="cover-history-card"
            @click="openHistoryViewer(item, 0)"
          >
            <span class="cover-history-thumb" :data-test="`cover-history-thumb-${index}`">
              <img :src="item.covers[0]?.url" :alt="item.title" />
            </span>
            <span class="cover-history-title">{{ item.title }}</span>
            <span class="cover-history-meta">{{ fmtTime(item.created_at) }} · {{ item.covers.length }} 张</span>
          </button>
        </div>
      </section>

      <CoverImageViewer
        v-if="viewerOpen && viewerImages.length"
        :images="viewerImages"
        :initial-index="viewerIndex"
        :title="viewerTitle"
        :download-base-name="viewerTitle"
        @close="closeCoverViewer"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'
import CoverImageViewer from '@/components/studio/CoverImageViewer.vue'
import {
  buildNovelCoverPayload,
  getCoverJob,
  getModelConfig,
  getWork,
  listWorks,
  startCoverJob,
  type CoverImage,
  type CoverJob,
  type WorkDetail,
  type WorkItem,
} from '@/api/studio'
import { useAppStore } from '@/stores/app'

type CoverMode = 'custom' | 'novel'

const COVER_MODE_STORAGE_KEY = 'studio_cover_mode'
const COVER_LAST_RESULT_STORAGE_KEY = 'studio_cover_last_result'
const COVER_ACTIVE_JOB_STORAGE_KEY = 'studio_cover_active_job'
const COVER_HISTORY_LIMIT = 12

interface CoverHistoryItem {
  id: number
  title: string
  model: string
  created_at: string
  covers: CoverImage[]
}

interface StoredCoverResult {
  mode?: CoverMode
  title?: string
  covers?: CoverImage[]
  created_at?: string
}

interface StoredActiveCoverJob {
  job_id: string
  title?: string
  mode?: CoverMode
  created_at?: string
}

const modes = [
  { value: 'custom' as const, label: '自定义' },
  { value: 'novel' as const, label: '小说驱动' },
]
const mode = ref<CoverMode>(readStoredMode())

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
const queueDone = ref(0)
const queueTotal = ref(0)
const appStore = useAppStore()
const COVER_JOB_POLL_INTERVAL_MS = 5000
const jobProgress = ref(0)
const jobMessage = ref('')
let generationRunId = 0
const viewerOpen = ref(false)
const viewerIndex = ref(0)
const viewerImages = ref<CoverImage[]>([])
const viewerTitle = ref('')
const restoredCoverTitle = ref('')
const coverHistory = ref<CoverHistoryItem[]>([])
const historyLoading = ref(false)

const isGPTImageModel = computed(() => imageModel.value.trim().toLowerCase().startsWith('gpt-image'))
const loadingText = computed(() => {
  const progressText = jobProgress.value > 0 ? ` ${jobProgress.value}%` : ''
  if (queueTotal.value > 1) {
    return `正在按队列生成封面 ${queueDone.value}/${queueTotal.value}${progressText}…`
  }
  return jobMessage.value ? `${jobMessage.value}${progressText}` : `正在生成候选封面${progressText}…`
})

function getRequestStatus(err: unknown): number | undefined {
  const e = err as { response?: { status?: number }; status?: number }
  return e.response?.status ?? e.status
}

function getRequestMessage(err: unknown): string {
  const e = err as {
    message?: string
    response?: { data?: { message?: string; error?: { message?: string } } }
    error?: { message?: string }
  }
  return e.response?.data?.error?.message || e.response?.data?.message || e.error?.message || e.message || ''
}

function isTimeoutError(err: unknown): boolean {
  const e = err as { code?: string }
  const message = getRequestMessage(err).toLowerCase()
  return e.code === 'ECONNABORTED' || message.includes('timeout') || message.includes('timed out') || message.includes('超时')
}

function readStoredMode(): CoverMode {
  try {
    const stored = localStorage.getItem(COVER_MODE_STORAGE_KEY)
    return stored === 'custom' || stored === 'novel' ? stored : 'novel'
  } catch {
    return 'novel'
  }
}

function fmtTime(s: string) {
  if (!s) return ''
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? s : d.toLocaleString()
}

function normalizeCovers(value: unknown): CoverImage[] {
  if (!Array.isArray(value)) return []
  const nextCovers: CoverImage[] = []
  value.forEach((cover, index) => {
    const c = cover as { id?: unknown; url?: unknown; prompt?: unknown }
    if (typeof c.url !== 'string' || !c.url) return
    nextCovers.push({
      id: typeof c.id === 'string' && c.id ? c.id : `cover-${index}`,
      url: c.url,
      prompt: typeof c.prompt === 'string' ? c.prompt : undefined,
    })
  })
  return nextCovers
}

function currentCoverTitle() {
  return (coverTitle.value || novelTitle.value || restoredCoverTitle.value || 'cover').trim() || 'cover'
}

function readLastCoverResult(): StoredCoverResult | null {
  try {
    return JSON.parse(localStorage.getItem(COVER_LAST_RESULT_STORAGE_KEY) || 'null') as StoredCoverResult | null
  } catch {
    return null
  }
}

function saveLastCoverResult(nextCovers = covers.value) {
  const cleanCovers = normalizeCovers(nextCovers)
  if (!cleanCovers.length) return
  const result: StoredCoverResult = {
    mode: mode.value,
    title: currentCoverTitle(),
    covers: cleanCovers,
    created_at: new Date().toISOString(),
  }
  try {
    localStorage.setItem(COVER_LAST_RESULT_STORAGE_KEY, JSON.stringify(result))
  } catch {
    /* localStorage can be unavailable in private modes. */
  }
}

function restoreLastCoverResult() {
  const stored = readLastCoverResult()
  const storedCovers = normalizeCovers(stored?.covers)
  if (!stored || !storedCovers.length) return
  if (stored.mode === 'custom' || stored.mode === 'novel') {
    mode.value = stored.mode
  }
  restoredCoverTitle.value = stored.title || ''
  covers.value = storedCovers
}

function readActiveCoverJob(): StoredActiveCoverJob | null {
  try {
    return JSON.parse(localStorage.getItem(COVER_ACTIVE_JOB_STORAGE_KEY) || 'null') as StoredActiveCoverJob | null
  } catch {
    return null
  }
}

function saveActiveCoverJob(job: CoverJob) {
  if (!job.job_id) return
  const activeJob: StoredActiveCoverJob = {
    job_id: job.job_id,
    title: currentCoverTitle(),
    mode: mode.value,
    created_at: new Date().toISOString(),
  }
  try {
    localStorage.setItem(COVER_ACTIVE_JOB_STORAGE_KEY, JSON.stringify(activeJob))
  } catch {
    /* localStorage can be unavailable in private modes. */
  }
}

function clearActiveCoverJob() {
  try {
    localStorage.removeItem(COVER_ACTIVE_JOB_STORAGE_KEY)
  } catch {
    /* localStorage can be unavailable in private modes. */
  }
}

function getDetailCovers(detail: WorkDetail) {
  const output = (detail.output as { covers?: unknown } | undefined) || {}
  return normalizeCovers(output.covers)
}

async function loadCoverHistory() {
  historyLoading.value = true
  try {
    const items = await listWorks('cover')
    const recent = [...items]
      .filter((item: WorkItem) => item.type === 'cover')
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .slice(0, COVER_HISTORY_LIMIT)
    const settled = await Promise.allSettled(
      recent.map(async (item) => {
        const detail = await getWork(item.id)
        return {
          id: item.id,
          title: detail.title || item.title,
          model: detail.model || item.model,
          created_at: detail.created_at || item.created_at,
          covers: getDetailCovers(detail),
        }
      }),
    )
    const nextHistory: CoverHistoryItem[] = []
    settled.forEach((result) => {
      if (result.status === 'fulfilled' && result.value.covers.length) {
        nextHistory.push(result.value)
      }
    })
    coverHistory.value = nextHistory
  } catch {
    coverHistory.value = []
  } finally {
    historyLoading.value = false
  }
}

async function resumeActiveCoverJob() {
  const activeJob = readActiveCoverJob()
  if (!activeJob?.job_id) return
  restoredCoverTitle.value = activeJob.title || restoredCoverTitle.value
  loading.value = true
  errorMsg.value = ''
  backendPending.value = false
  const runId = (generationRunId += 1)
  try {
    const job = await getCoverJob(activeJob.job_id)
    const result = normalizeCovers(await waitForCoverJob(job, runId))
    if (runId !== generationRunId) return
    if (result.length) {
      covers.value = result
      saveLastCoverResult(result)
      await loadCoverHistory()
    }
    clearActiveCoverJob()
  } catch (err: unknown) {
    if (runId === generationRunId) {
      errorMsg.value = getRequestMessage(err) || '生成任务恢复失败，请稍后重试。'
      clearActiveCoverJob()
    }
  } finally {
    if (runId === generationRunId) {
      loading.value = false
      jobMessage.value = ''
      jobProgress.value = 0
    }
  }
}

onMounted(async () => {
  restoreLastCoverResult()
  await loadCoverHistory()
  await resumeActiveCoverJob()
  try {
    const cfg = await getModelConfig()
    if (cfg.image) {
      configured.value = true
      imageModel.value = cfg.image.model
      if (isGPTImageModel.value) {
        count.value = 1
      }
    }
  } catch {
    /* 忽略：未配置 */
  }
})

onBeforeUnmount(() => {
  generationRunId += 1
})

watch(mode, (value) => {
  try {
    localStorage.setItem(COVER_MODE_STORAGE_KEY, value)
  } catch {
    /* localStorage can be unavailable in private modes. */
  }
})

const canSubmit = computed(() => {
  if (loading.value || !configured.value) return false
  return mode.value === 'custom'
    ? prompt.value.trim().length > 0
    : synopsis.value.trim().length > 0 && protagonist.value.trim().length > 0
})

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function openCoverViewer(index: number) {
  if (!covers.value.length) return
  viewerImages.value = covers.value
  viewerTitle.value = currentCoverTitle()
  viewerIndex.value = clamp(index, 0, covers.value.length - 1)
  viewerOpen.value = true
}

function closeCoverViewer() {
  viewerOpen.value = false
}

function openHistoryViewer(item: CoverHistoryItem, index: number) {
  if (!item.covers.length) return
  viewerImages.value = item.covers
  viewerTitle.value = item.title || 'cover'
  viewerIndex.value = clamp(index, 0, item.covers.length - 1)
  viewerOpen.value = true
}

function wait(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

function isCoverJobPending(job: CoverJob) {
  return job.status === 'running'
}

function updateCoverJobState(job: CoverJob) {
  jobProgress.value = Math.max(0, Math.min(100, job.progress || 0))
  jobMessage.value = job.message || ''
}

async function waitForCoverJob(initialJob: CoverJob, runId: number): Promise<CoverImage[]> {
  let job = initialJob
  updateCoverJobState(job)
  while (isCoverJobPending(job)) {
    await wait(COVER_JOB_POLL_INTERVAL_MS)
    if (runId !== generationRunId) return []
    job = await getCoverJob(job.job_id)
    updateCoverJobState(job)
  }
  if (job.status === 'failed') {
    throw new Error(job.error || job.message || '生成失败，请稍后重试。')
  }
  return job.result?.covers || []
}

async function runCoverJob(payload: Parameters<typeof startCoverJob>[0], runId: number): Promise<CoverImage[]> {
  const job = await startCoverJob(payload)
  saveActiveCoverJob(job)
  try {
    return await waitForCoverJob(job, runId)
  } finally {
    if (runId === generationRunId) {
      clearActiveCoverJob()
    }
  }
}

async function submit() {
  if (!canSubmit.value) return
  const requestedCount = Math.max(1, Math.min(4, count.value || 1))
  const useQueue = isGPTImageModel.value && requestedCount > 1
  if (useQueue) {
    const ok = window.confirm(`OpenAI ${imageModel.value} 每次只支持生成 1 张图。将按队列连续生成 ${requestedCount} 张，是否继续？`)
    if (!ok) return
    appStore.showWarning(`已切换为队列生成：${requestedCount} 张封面会按 1 张一批依次生成。`, 6000)
  }

  loading.value = true
  errorMsg.value = ''
  backendPending.value = false
  covers.value = []
  closeCoverViewer()
  queueDone.value = 0
  queueTotal.value = useQueue ? requestedCount : 0
  jobProgress.value = 0
  jobMessage.value = ''
  const runId = (generationRunId += 1)
  try {
    const buildPayload = (jobCount: number) =>
      mode.value === 'custom'
        ? { mode: 'custom' as const, prompt: prompt.value.trim(), ref_image: refImage.value.trim() || undefined, size: size.value, count: jobCount }
        : buildNovelCoverPayload({
            title: novelTitle.value,
            synopsis: synopsis.value,
            protagonist: protagonist.value,
            genre: genre.value,
            mood: mood.value,
            keyScene: keyScene.value,
            coverTitle: coverTitle.value,
            size: size.value,
            count: jobCount,
          })

    if (useQueue) {
      for (let i = 0; i < requestedCount; i += 1) {
        const result = await runCoverJob(buildPayload(1), runId)
        const batch = result.map((cover, index) => ({
          ...cover,
          id: `${cover.id || 'cover'}-${i}-${index}`,
        }))
        if (runId !== generationRunId) return
        covers.value = [...covers.value, ...batch]
        saveLastCoverResult(covers.value)
        queueDone.value = i + 1
      }
    } else {
      const result = await runCoverJob(buildPayload(requestedCount), runId)
      if (runId !== generationRunId) return
      covers.value = result
      saveLastCoverResult(result)
    }
    if (covers.value.length) {
      await loadCoverHistory()
    }
  } catch (err: unknown) {
    const status = getRequestStatus(err)
    if (status === 404) {
      backendPending.value = true
    } else if (isTimeoutError(err)) {
      errorMsg.value = '生成耗时较长，请稍后重试，或换一张更小的参考图。'
    } else {
      errorMsg.value = getRequestMessage(err) || '生成失败，请稍后重试。'
    }
  } finally {
    if (runId === generationRunId) {
      loading.value = false
      queueTotal.value = 0
      jobMessage.value = ''
      jobProgress.value = 0
    }
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

.cover-thumb {
  position: relative;
  display: block;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: rgb(17 24 39);
  cursor: zoom-in;
}

.cover-thumb::after {
  content: '';
  position: absolute;
  inset: 0;
  border: 1px solid rgba(255, 255, 255, 0);
  background: linear-gradient(180deg, transparent 58%, rgba(0, 0, 0, 0.42));
  opacity: 0;
  transition: opacity 0.18s ease, border-color 0.18s ease;
}

.cover-thumb:hover::after,
.cover-thumb:focus-visible::after {
  border-color: rgba(255, 255, 255, 0.42);
  opacity: 1;
}

.cover-thumb:focus-visible {
  outline: 3px solid rgba(220, 56, 31, 0.5);
  outline-offset: -3px;
}

.cover-history-section {
  margin-top: 1rem;
  border: 1px solid rgb(229 231 235);
  border-radius: 12px;
  background: #fff;
  padding: 1rem;
}

.dark .cover-history-section {
  border-color: rgb(31 41 55);
  background: rgb(17 24 39);
}

.cover-history-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.8rem;
}

.cover-history-head h2 {
  font-size: 16px;
  font-weight: 900;
  color: rgb(17 24 39);
}

.dark .cover-history-head h2 {
  color: #fff;
}

.cover-history-head p,
.cover-history-head span {
  margin-top: 0.15rem;
  font-size: 12px;
  color: rgb(107 114 128);
}

.dark .cover-history-head p,
.dark .cover-history-head span {
  color: rgb(156 163 175);
}

.cover-history-track {
  display: flex;
  gap: 0.75rem;
  overflow-x: auto;
  overscroll-behavior-x: contain;
  padding-bottom: 0.35rem;
  scrollbar-width: thin;
}

.cover-history-card {
  display: flex;
  width: 138px;
  flex: 0 0 138px;
  flex-direction: column;
  gap: 0.45rem;
  border: 1px solid rgb(229 231 235);
  border-radius: 10px;
  background: rgb(249 250 251);
  padding: 0.45rem;
  text-align: left;
  transition: transform 0.16s ease, border-color 0.16s ease, background 0.16s ease;
}

.dark .cover-history-card {
  border-color: rgb(31 41 55);
  background: rgb(11 18 32);
}

.cover-history-card:hover,
.cover-history-card:focus-visible {
  border-color: rgba(220, 56, 31, 0.5);
  background: rgba(220, 56, 31, 0.06);
  transform: translateY(-2px);
}

.cover-history-card:focus-visible {
  outline: 3px solid rgba(220, 56, 31, 0.25);
  outline-offset: 2px;
}

.cover-history-thumb {
  display: block;
  width: 100%;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  border-radius: 8px;
  background: rgb(17 24 39);
}

.cover-history-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.cover-history-title {
  overflow: hidden;
  color: rgb(17 24 39);
  font-size: 13px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dark .cover-history-title {
  color: #fff;
}

.cover-history-meta {
  overflow: hidden;
  color: rgb(107 114 128);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dark .cover-history-meta {
  color: rgb(156 163 175);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
