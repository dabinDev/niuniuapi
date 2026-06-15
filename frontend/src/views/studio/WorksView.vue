<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl">
      <header class="mb-4">
        <span class="inline-block rounded-full bg-primary-50 px-3 py-1 text-xs font-bold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
          🗂️ 创作台
        </span>
        <h1 class="mt-2 text-2xl font-black text-gray-900 dark:text-white sm:text-3xl">我的作品</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">封面、拆书报告、剧本——每次生成都在这里，可随时回看。</p>
      </header>

      <!-- 类型筛选 -->
      <div class="mb-4 inline-flex flex-wrap gap-1 rounded-lg border border-gray-200 bg-white p-1 dark:border-dark-700 dark:bg-dark-900">
        <button
          v-for="t in tabs"
          :key="t.value"
          type="button"
          class="rounded-md px-3 py-1.5 text-sm font-semibold transition"
          :class="activeType === t.value ? 'bg-primary-600 text-white' : 'text-gray-600 hover:text-primary-600 dark:text-gray-300'"
          @click="activeType = t.value"
        >
          {{ t.label }}
        </button>
      </div>

      <div class="grid gap-4 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
        <!-- 列表 -->
        <section>
          <div v-if="loading" class="py-12 text-center text-gray-400">加载中…</div>
          <div v-else-if="!works.length" class="rounded-xl border border-dashed border-gray-300 py-12 text-center text-gray-400 dark:border-dark-700">
            还没有作品，去创作台生成第一个吧。
          </div>
          <ul v-else class="grid gap-2">
            <li v-for="w in works" :key="w.id">
              <button
                type="button"
                class="work-card"
                :class="{ 'work-card-active': selected && selected.id === w.id }"
                @click="select(w)"
              >
                <span class="work-badge" :class="badgeClass(w.type)">{{ typeLabel(w.type) }}</span>
                <span class="min-w-0 flex-1 truncate text-left font-semibold text-gray-900 dark:text-white">{{ w.title }}</span>
                <span class="shrink-0 text-xs text-gray-400">{{ fmtTime(w.created_at) }}</span>
              </button>
            </li>
          </ul>
        </section>

        <!-- 详情 -->
        <section class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-800 dark:bg-dark-900">
          <div v-if="detailLoading" class="py-12 text-center text-gray-400">加载中…</div>
          <div v-else-if="!selected" class="py-12 text-center text-gray-400">从左侧选一个作品查看详情。</div>

          <template v-else>
            <h2 class="text-lg font-black text-gray-900 dark:text-white">{{ selected.title }}</h2>
            <p class="mb-3 text-xs text-gray-400">{{ typeLabel(selected.type) }} · {{ selected.model }} · {{ fmtTime(selected.created_at) }}</p>

            <!-- 封面 -->
            <div v-if="selected.type === 'cover'" class="grid grid-cols-2 gap-3">
              <figure v-for="(c, i) in coverList" :key="i" class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-800" style="aspect-ratio: 3 / 4">
                <button
                  type="button"
                  class="work-cover-thumb"
                  :data-test="`work-cover-thumb-${i}`"
                  :aria-label="`查看作品封面 ${i + 1}`"
                  @click="openCoverViewer(i)"
                >
                  <img :src="c.url" alt="cover" class="h-full w-full object-cover" />
                </button>
              </figure>
            </div>

            <!-- 拆书 -->
            <div v-else-if="selected.type === 'teardown'">
              <div class="flex items-baseline gap-3">
                <span class="text-4xl font-black text-primary-600 dark:text-primary-400">{{ report.overall_score }}</span>
                <span class="text-sm font-bold text-gray-900 dark:text-white">{{ report.verdict }}</span>
              </div>
              <p class="mt-2 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ report.summary }}</p>
              <div v-if="report.rotten_points && report.rotten_points.length" class="mt-3">
                <h3 class="text-sm font-bold text-gray-900 dark:text-white">🍅 烂点</h3>
                <ul class="mt-1 list-disc pl-5 text-sm text-red-500"><li v-for="(r, i) in report.rotten_points" :key="i">{{ r }}</li></ul>
              </div>
              <div v-if="report.suggestions && report.suggestions.length" class="mt-3">
                <h3 class="text-sm font-bold text-gray-900 dark:text-white">🛠️ 建议</h3>
                <ul class="mt-1 list-disc pl-5 text-sm text-gray-600 dark:text-gray-300"><li v-for="(s, i) in report.suggestions" :key="i">{{ s }}</li></ul>
              </div>
            </div>

            <!-- 剧本 -->
            <div v-else-if="selected.type === 'script'">
              <div v-for="(s, i) in scriptScenes" :key="i" class="mb-3 border-l-2 border-primary-500/50 pl-3">
                <h3 class="text-sm font-bold text-primary-700 dark:text-primary-300">{{ s.heading }}</h3>
                <p class="mt-1 whitespace-pre-wrap text-sm leading-7 text-gray-600 dark:text-gray-300">{{ s.content }}</p>
              </div>
            </div>
          </template>
        </section>
      </div>

      <CoverImageViewer
        v-if="viewerOpen && coverList.length"
        :images="coverList"
        :initial-index="viewerIndex"
        :title="selected?.title || 'cover'"
        :download-base-name="selected?.title || 'cover'"
        @close="closeCoverViewer"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import CoverImageViewer from '@/components/studio/CoverImageViewer.vue'
import { getWork, listWorks, type CoverImage, type WorkDetail, type WorkItem } from '@/api/studio'

const tabs = [
  { value: 'all', label: '全部' },
  { value: 'cover', label: '封面' },
  { value: 'teardown', label: '拆书' },
  { value: 'script', label: '剧本' },
]
const activeType = ref('all')
const works = ref<WorkItem[]>([])
const loading = ref(false)
const selected = ref<WorkDetail | null>(null)
const detailLoading = ref(false)
const viewerOpen = ref(false)
const viewerIndex = ref(0)

const typeLabelMap: Record<string, string> = { cover: '封面', teardown: '拆书', script: '剧本' }
function typeLabel(t: string) {
  return typeLabelMap[t] || t
}
function badgeClass(t: string) {
  return t === 'cover' ? 'badge-cover' : t === 'teardown' ? 'badge-teardown' : 'badge-script'
}
function fmtTime(s: string) {
  if (!s) return ''
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? s : d.toLocaleString()
}

const out = computed<Record<string, unknown>>(() => (selected.value?.output as Record<string, unknown>) || {})
const coverList = computed<CoverImage[]>(() => {
  const raw = (out.value.covers as CoverImage[]) || []
  return raw.filter((cover) => cover && typeof cover.url === 'string' && cover.url.length > 0)
})
const report = computed(() => out.value as {
  overall_score?: number
  verdict?: string
  summary?: string
  rotten_points?: string[]
  suggestions?: string[]
})
const scriptScenes = computed(() => (out.value.scenes as { heading: string; content: string }[]) || [])

async function load() {
  loading.value = true
  try {
    works.value = await listWorks(activeType.value === 'all' ? undefined : activeType.value)
  } catch {
    works.value = []
  } finally {
    loading.value = false
  }
}

async function select(w: WorkItem) {
  detailLoading.value = true
  selected.value = null
  closeCoverViewer()
  try {
    selected.value = await getWork(w.id)
  } catch {
    selected.value = null
  } finally {
    detailLoading.value = false
  }
}

function openCoverViewer(index: number) {
  if (!coverList.value.length) return
  viewerIndex.value = Math.min(Math.max(index, 0), coverList.value.length - 1)
  viewerOpen.value = true
}

function closeCoverViewer() {
  viewerOpen.value = false
}

watch(activeType, () => {
  selected.value = null
  closeCoverViewer()
  void load()
})

onMounted(load)
onBeforeUnmount(closeCoverViewer)
</script>

<style scoped>
.work-card {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 0.6rem;
  border: 1px solid rgb(229 231 235);
  border-radius: 10px;
  background: #fff;
  padding: 0.6rem 0.75rem;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.dark .work-card {
  border-color: rgb(38 38 38);
  background: rgb(17 24 39);
}

.work-card:hover {
  border-color: rgba(232, 65, 46, 0.4);
}

.work-card-active {
  border-color: rgb(232 65 46);
  background: rgba(232, 65, 46, 0.05);
}

.work-badge {
  flex-shrink: 0;
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 800;
}

.badge-cover {
  background: #ffe7e0;
  color: #c8351f;
}

.badge-teardown {
  background: #fff1d9;
  color: #a8631a;
}

.badge-script {
  background: #e0ecff;
  color: #1e51b8;
}

.work-cover-thumb {
  position: relative;
  display: block;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: rgb(17 24 39);
  cursor: zoom-in;
}

.work-cover-thumb::after {
  content: '';
  position: absolute;
  inset: 0;
  border: 1px solid rgba(255, 255, 255, 0);
  background: linear-gradient(180deg, transparent 58%, rgba(0, 0, 0, 0.42));
  opacity: 0;
  transition: opacity 0.18s ease, border-color 0.18s ease;
}

.work-cover-thumb:hover::after,
.work-cover-thumb:focus-visible::after {
  border-color: rgba(255, 255, 255, 0.42);
  opacity: 1;
}

.work-cover-thumb:focus-visible {
  outline: 3px solid rgba(220, 56, 31, 0.5);
  outline-offset: -3px;
}
</style>
