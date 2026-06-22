<template>
  <AppLayout>
    <div class="works-page studio-wide-shell mx-auto max-w-none" data-test="studio-workbench-shell">
      <header class="works-head">
        <div>
          <span class="eyebrow">作品任务台</span>
          <h1>我的作品</h1>
          <p>把导入、诊断、对标、生成和封面历史收进同一个任务台；先看当前动作，再进入归档继续加工。</p>
        </div>
        <div class="quick-actions" aria-label="作品快捷入口">
          <router-link v-if="isStudioVisible('fanqie')" to="/studio/fanqie">查看热榜</router-link>
          <router-link v-if="isStudioVisible('teardown')" to="/studio/teardown">拆书诊断</router-link>
          <router-link v-if="isStudioVisible('generate')" class="primary" to="/studio/generate">创作生成</router-link>
        </div>
      </header>

      <section class="mission-control mission-control-mobile-compact" data-test="works-mission-control" aria-label="作品任务台">
        <article class="mission-panel queue-panel" data-test="works-queue-panel">
          <span class="eyebrow small">Queue</span>
          <strong>作品队列</strong>
          <p>{{ works.length ? `当前归档 ${works.length} 个产出，默认打开最新一条。` : '还没有归档，先从热榜选样本或生成第一稿。' }}</p>
        </article>
        <article class="mission-panel status-panel" data-test="works-status-panel">
          <span class="eyebrow small">Status</span>
          <strong>当前作品状态</strong>
          <p>{{ selected ? selectedStatusText : '选择一个作品后，这里会显示导入、诊断、对标或生成状态。' }}</p>
        </article>
        <article class="mission-panel action-panel" data-test="works-action-panel">
          <span class="eyebrow small">Next Action</span>
          <strong>下一步动作</strong>
          <p>{{ selected ? selectedActionHint : '先创建素材，再把作品推进到拆书、对标或生成。' }}</p>
          <div class="mission-actions">
            <router-link v-if="isStudioVisible('fanqie')" to="/studio/fanqie">热榜选样本</router-link>
            <router-link v-if="isStudioVisible('teardown')" to="/studio/teardown">拆书诊断</router-link>
            <router-link v-if="isStudioVisible('generate')" class="primary" to="/studio/generate">生成下一版</router-link>
          </div>
        </article>
      </section>

      <section class="workbench-grid workbench-grid-wide">
        <aside class="panel chapter-panel" aria-label="章节树">
          <div class="panel-head">
            <span>章节树</span>
            <small>Chapter Map</small>
          </div>
          <ol>
            <li v-for="chapter in chapterNodes" :key="chapter.title" :class="chapter.state">
              <b>{{ chapter.index }}</b>
              <div>
                <strong>{{ chapter.title }}</strong>
                <p>{{ chapter.note }}</p>
              </div>
            </li>
          </ol>
        </aside>

        <main class="panel overview-panel" aria-label="概览看板">
          <div class="panel-head">
            <span>概览看板</span>
            <small>Quality Board</small>
          </div>
          <nav class="workbench-tabs" aria-label="作品功能">
            <button v-for="tab in workbenchTabs" :key="tab" type="button">{{ tab }}</button>
          </nav>

          <div class="metric-grid">
            <article v-for="metric in projectMetrics" :key="metric.label">
              <span>{{ metric.label }}</span>
              <strong>{{ metric.value }}</strong>
              <p>{{ metric.hint }}</p>
            </article>
          </div>

          <div class="todo-panel">
            <div class="panel-head">
              <span>下一步建议</span>
              <small>从归档结果继续推进</small>
            </div>
            <ul>
              <li v-for="todo in todoItems" :key="todo">{{ todo }}</li>
            </ul>
          </div>
        </main>

        <aside class="panel setting-panel" aria-label="设定库">
          <div class="panel-head">
            <span>设定库</span>
            <small>Canon Vault</small>
          </div>
          <div class="setting-list">
            <article v-for="group in settingGroups" :key="group.label">
              <span>{{ group.label }}</span>
              <strong>{{ group.value }}</strong>
            </article>
          </div>
        </aside>
      </section>

      <section class="archive-section archive-section-mobile-stack" data-test="works-archive-section" aria-label="最近产出归档">
        <div class="archive-head">
          <div>
            <span class="eyebrow small">最近产出</span>
            <h2>作品归档</h2>
            <p>所有创作结果按时间进入这里，左侧筛选和选择，右侧查看正文、报告或封面大图。</p>
          </div>
          <div class="type-tabs" aria-label="归档类型筛选">
            <button
              v-for="t in tabs"
              :key="t.value"
              type="button"
              :class="{ active: activeType === t.value }"
              @click="activeType = t.value"
            >
              {{ t.label }}
            </button>
          </div>
        </div>

        <div class="archive-grid archive-grid-wide archive-grid-mobile-flow" data-test="works-archive-grid">
          <section class="archive-list">
            <div v-if="loading" class="archive-empty">加载中...</div>
            <div v-else-if="!works.length" class="archive-empty dashed">
              还没有作品，先去导入章节或生成第一份报告。
            </div>
            <ul v-else>
              <li v-for="w in works" :key="w.id">
                <button
                  type="button"
                  class="work-card"
                  :class="{ active: selected && selected.id === w.id }"
                  @click="select(w)"
                >
                  <span class="work-badge" :class="badgeClass(w.type)">{{ typeLabel(w.type) }}</span>
                  <strong>{{ w.title }}</strong>
                  <small>{{ fmtTime(w.created_at) }}</small>
                </button>
              </li>
            </ul>
          </section>

          <section class="detail-panel">
            <div v-if="detailLoading" class="archive-empty">加载中...</div>
            <div v-else-if="!selected" class="archive-empty">从左侧选一个归档查看详情。</div>

            <template v-else>
              <div class="detail-head">
                <span>{{ typeLabel(selected.type) }} · {{ selected.model }}</span>
                <h2>{{ selected.title }}</h2>
                <p>{{ fmtTime(selected.created_at) }}</p>
              </div>

              <div v-if="selected.type === 'cover'" class="cover-grid">
                <figure v-for="(c, i) in coverList" :key="i">
                  <button
                    type="button"
                    class="work-cover-thumb"
                    :data-test="`work-cover-thumb-${i}`"
                    :aria-label="`查看作品封面 ${i + 1}`"
                    @click="openCoverViewer(i)"
                  >
                    <img :src="c.url" alt="cover" />
                  </button>
                </figure>
              </div>

              <div v-else-if="selected.type === 'teardown'" class="detail-block">
                <div class="score-inline">
                  <strong>{{ report.overall_score || '-' }}</strong>
                  <span>{{ report.verdict }}</span>
                </div>
                <p v-if="report.summary">{{ report.summary }}</p>
                <ListGroup title="烂点" :items="report.rotten_points || []" tone="danger" />
                <ListGroup title="建议" :items="report.suggestions || []" />
              </div>

              <div v-else-if="selected.type === 'hotspot'" class="detail-block">
                <div class="score-inline">
                  <strong>{{ hotspot.market_score || '-' }}</strong>
                  <span>{{ hotspot.verdict }}</span>
                </div>
                <ListGroup title="可复用套路" :items="hotspot.tropes || []" />
                <ListGroup title="下一步动作" :items="hotspot.actions || []" tone="success" />
              </div>

              <div v-else-if="selected.type === 'generate'" class="detail-block">
                <p v-if="creative.summary">{{ creative.summary }}</p>
                <article v-for="(section, i) in creative.sections || []" :key="i" class="section-output">
                  <h3>{{ section.heading }}</h3>
                  <p>{{ section.content }}</p>
                </article>
                <ListGroup title="下一步" :items="creative.next_steps || []" tone="success" />
              </div>

              <div v-else-if="selected.type === 'script'" class="detail-block">
                <article v-for="(s, i) in scriptScenes" :key="i" class="section-output">
                  <h3>{{ s.heading }}</h3>
                  <p>{{ s.content }}</p>
                </article>
              </div>

              <div v-else-if="selected.type === 'import'" class="detail-block">
                <div class="import-overview" data-test="import-overview">
                  <article>
                    <span>章节数</span>
                    <strong>{{ importChapterStats.count }}</strong>
                  </article>
                  <article>
                    <span>已统计字数</span>
                    <strong>{{ importChapterStats.wordsLabel }}</strong>
                  </article>
                  <article>
                    <span>展示策略</span>
                    <strong>预览前 {{ visibleImportChapters.length }} 章</strong>
                  </article>
                </div>
                <p v-if="hasHiddenImportChapters" class="import-note" data-test="import-note">
                  {{ importChapterStats.chaptersWithContent > 0 ? '已导入章节正文' : '已记录完整目录数据' }}
                  {{ importChapterStats.count }} 章。为避免作品页被全本目录撑爆，这里先展示前
                  {{ importPreviewLimit }} 章；{{
                    importChapterStats.chaptersWithContent > 0
                      ? '后续拆书/对标/生成会优先使用已入库正文。'
                      : '当前适合做结构浏览，如需正文级分析请重新导入授权 TXT。'
                  }}
                </p>
                <p
                  v-if="importChapterStats.missingWords > 0"
                  class="import-note import-note-soft"
                  data-test="import-missing-words-note"
                >
                  有 {{ importChapterStats.missingWords }} 章暂未采集到字数，{{
                    importChapterStats.chaptersWithContent > 0
                      ? '章节正文仍已导入，可继续用于拆书、对标和生成。'
                      : '正文待采集，请导入授权 TXT 或等待采集完成后再做正文级分析。'
                  }}
                </p>
                <div class="import-stage-rail" data-test="import-stage-rail" aria-label="导入阶段">
                  <article class="done">
                    <b>01</b>
                    <strong>目录</strong>
                    <span>{{ importChapterStats.count }} 章已记录</span>
                  </article>
                  <article :class="{ done: importChapterStats.chaptersWithContent > 0, warn: importChapterStats.chaptersWithContent === 0 }">
                    <b>02</b>
                    <strong>正文</strong>
                    <span>{{ importChapterStats.chaptersWithContent > 0 ? `${importChapterStats.chaptersWithContent} 章可用` : '待采集或导入 TXT' }}</span>
                  </article>
                  <article :class="{ done: importChapterStats.chaptersWithContent > 0, warn: importChapterStats.chaptersWithContent === 0 }">
                    <b>03</b>
                    <strong>再加工</strong>
                    <span>{{ importChapterStats.chaptersWithContent > 0 ? '可直接拆书/对标/生成' : '先做结构浏览' }}</span>
                  </article>
                </div>
                <div class="import-action-strip" data-test="import-action-strip">
                  <div>
                    <span data-test="import-content-state" :class="importContentStatus.className">
                      {{ importContentStatus.title }}
                    </span>
                    <strong>{{ importContentStatus.description }}</strong>
                  </div>
                  <button v-if="isStudioVisible('teardown')" type="button" data-test="import-send-teardown" @click="sendImportTo('teardown')">送去拆书</button>
                  <button v-if="isStudioVisible('hotspot')" type="button" data-test="import-send-hotspot" @click="sendImportTo('hotspot')">送去对标</button>
                  <button v-if="isStudioVisible('generate')" type="button" data-test="import-send-generate" @click="sendImportTo('generate')">生成续写</button>
                </div>
                <div
                  class="chapter-summary"
                  :class="{ 'scrollable-preview': hasHiddenImportChapters }"
                  data-test="import-chapter-summary"
                >
                  <article v-for="(chapter, index) in visibleImportChapters" :key="chapter.title + chapter.source">
                    <strong>{{ displayChapterTitle(chapter, index) }}</strong>
                    <span v-if="chapter.word_count" class="word-count">{{ formatNumber(chapter.word_count) }} 字</span>
                    <span v-else class="word-count pending" data-test="import-chapter-word-count-pending">待采集</span>
                    <small>{{ sourceLabel(chapter.source) }}</small>
                  </article>
                </div>
                <ListGroup title="下一步" :items="imported.next_actions || []" tone="success" />
              </div>
            </template>
          </section>
        </div>
      </section>

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
import { computed, defineComponent, h, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import CoverImageViewer from '@/components/studio/CoverImageViewer.vue'
import { getWork, listWorks, type CoverImage, type WorkDetail, type WorkItem } from '@/api/studio'
import { useAppStore, useAuthStore } from '@/stores'
import { isStudioFeatureVisible, type StudioFeatureId } from '@/utils/studioFeatures'

const ListGroup = defineComponent({
  name: 'ListGroup',
  props: {
    title: { type: String, required: true },
    items: { type: Array as () => string[], required: true },
    tone: { type: String, default: 'default' },
  },
  setup(props) {
    return () => props.items.length
      ? h('section', { class: ['list-group', `tone-${props.tone}`] }, [
        h('h3', props.title),
        h('ul', props.items.map((item) => h('li', { key: item }, item))),
      ])
      : null
  },
})

const archiveTabs: Array<{ value: string; label: string; feature?: StudioFeatureId }> = [
  { value: 'all', label: '全部' },
  { value: 'import', label: '导入' },
  { value: 'teardown', label: '拆书', feature: 'teardown' },
  { value: 'hotspot', label: '爆款', feature: 'hotspot' },
  { value: 'generate', label: '生成', feature: 'generate' },
  { value: 'script', label: '剧本', feature: 'generate' },
  { value: 'cover', label: '封面', feature: 'cover' },
]

const projectMetricDefinitions: Array<{ label: string; value: string; hint: string; feature?: StudioFeatureId }> = [
  { label: '综合诊断', value: '待分析', hint: '从拆书报告自动汇总', feature: 'teardown' },
  { label: '爆款潜力', value: '待对标', hint: '对标后形成雷达', feature: 'hotspot' },
  { label: '设定一致性', value: '待沉淀', hint: '导入章节后建立设定库' },
]

const workbenchTabDefinitions: Array<{ label: string; feature?: StudioFeatureId }> = [
  { label: '概览' },
  { label: '拆书', feature: 'teardown' },
  { label: '爆款', feature: 'hotspot' },
  { label: '大纲', feature: 'generate' },
  { label: '正文', feature: 'generate' },
  { label: '剧本', feature: 'generate' },
  { label: '热榜', feature: 'fanqie' },
]

const chapterNodeDefinitions: Array<{ index: string; title: string; note: string; state: string; feature?: StudioFeatureId }> = [
  { index: '01', title: '导入首章', note: '先把开篇送去拆书，确定钩子和节奏。', state: 'good', feature: 'teardown' },
  { index: '02', title: '诊断缺口', note: '用爆款对标确认爽点兑现是否偏慢。', state: 'warn', feature: 'hotspot' },
  { index: '03', title: '生成修订', note: '把改法喂给创作台，形成新版本。', state: 'good', feature: 'generate' },
]

const settingGroups = [
  { label: '人物卡', value: '从导入章节抽取' },
  { label: '金手指', value: '拆书后沉淀规则与代价' },
  { label: '世界观', value: '记录地点、势力、限制' },
  { label: '伏笔表', value: '跟踪埋设与回收' },
]

const todoItemDefinitions: Array<{ text: string; feature?: StudioFeatureId }> = [
  { text: '先从热榜找 3 个同题材样本，确认读者正在追什么。', feature: 'fanqie' },
  { text: '用拆书报告找出黄金三章的掉线位置。', feature: 'teardown' },
  { text: '把爆款对标动作直接带进创作生成，形成下一版正文。', feature: 'generate' },
]

const activeType = ref('all')
const works = ref<WorkItem[]>([])
const loading = ref(false)
const selected = ref<WorkDetail | null>(null)
const detailLoading = ref(false)
const viewerOpen = ref(false)
const viewerIndex = ref(0)
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const importPreviewLimit = 60

type ImportedChapter = {
  title: string
  word_count: number
  source?: string
  content?: string
}

const typeLabelMap: Record<string, string> = {
  cover: '封面',
  teardown: '拆书',
  hotspot: '爆款',
  generate: '生成',
  script: '剧本',
  import: '导入',
}

function typeLabel(t: string) {
  return typeLabelMap[t] || t
}

function badgeClass(t: string) {
  return `badge-${t}`
}

function fmtTime(s: string) {
  if (!s) return ''
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? s : d.toLocaleString()
}

const formatNumber = (value: number) => new Intl.NumberFormat('zh-CN').format(value)

function normalizeChapterTitle(title: string, index: number) {
  const normalized = title.replace(/^第\s*(\d+)\s*章/, (_match, chapterNo) => `第 ${String(chapterNo).padStart(2, '0')} 章`)
  return normalized || `第 ${String(index + 1).padStart(2, '0')} 章`
}

function displayChapterTitle(chapter: ImportedChapter, index: number) {
  if (chapter.source?.includes('fanqienovel.com')) return normalizeChapterTitle(chapter.title, index)
  return chapter.title.replace(/^第\s*(\d+)\s*章/, (_match, chapterNo) => `第 ${chapterNo} 章`)
}

function sourceLabel(source?: string) {
  if (!source) return '粘贴内容'
  return source.includes('fanqienovel.com') ? '番茄来源' : '外部来源'
}

function buildImportContent() {
  return importChapters.value
    .map((chapter, index) => {
      const title = chapter.title || `第${index + 1}章`
      return chapter.content ? `${title}\n${chapter.content}` : title
    })
    .join('\n\n')
}

function sendImportTo(target: 'teardown' | 'hotspot' | 'generate') {
  if (!isStudioVisible(target)) return
  if (!selected.value) return
  localStorage.setItem('studio_bridge_payload', JSON.stringify({
    source: 'works',
    target,
    title: selected.value.title,
    genre: '番茄导入',
    content: buildImportContent(),
    brief: `${selected.value.title}：${importChapterStats.value.count} 章，${importContentStatus.value.title}`,
  }))
  void router.push(`/studio/${target}`)
}

const studioVisibility = computed(() => appStore.cachedPublicSettings?.studio_feature_visibility)
const tabs = computed(() => archiveTabs.filter((tab) => !tab.feature || isStudioVisible(tab.feature)))
const projectMetrics = computed(() => projectMetricDefinitions.filter((metric) => !metric.feature || isStudioVisible(metric.feature)))
const workbenchTabs = computed(() => workbenchTabDefinitions.filter((tab) => !tab.feature || isStudioVisible(tab.feature)).map((tab) => tab.label))
const todoItems = computed(() => todoItemDefinitions.filter((todo) => !todo.feature || isStudioVisible(todo.feature)).map((todo) => todo.text))
const chapterNodes = computed(() => chapterNodeDefinitions.filter((node) => !node.feature || isStudioVisible(node.feature)))
const visibleImportHandoffLabels = computed(() => [
  isStudioVisible('teardown') ? '拆书' : '',
  isStudioVisible('hotspot') ? '对标' : '',
  isStudioVisible('generate') ? '续写' : '',
].filter(Boolean))

function isStudioVisible(id: StudioFeatureId) {
  return isStudioFeatureVisible(id, studioVisibility.value, authStore.isAdmin)
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
const hotspot = computed(() => out.value as {
  market_score?: number
  verdict?: string
  tropes?: string[]
  actions?: string[]
})
const creative = computed(() => out.value as {
  summary?: string
  sections?: { heading: string; content: string }[]
  next_steps?: string[]
})
const imported = computed(() => out.value as {
  chapters?: ImportedChapter[]
  next_actions?: string[]
})
const scriptScenes = computed(() => (out.value.scenes as { heading: string; content: string }[]) || [])
const importChapters = computed(() => imported.value.chapters || [])
const visibleImportChapters = computed(() => importChapters.value.slice(0, importPreviewLimit))
const hasHiddenImportChapters = computed(() => importChapters.value.length > visibleImportChapters.value.length)
const importChapterStats = computed(() => {
  const chapters = importChapters.value
  const words = chapters.reduce((sum, chapter) => sum + (Number(chapter.word_count) || 0), 0)
  const chaptersWithContent = chapters.filter((chapter) => Boolean(chapter.content && chapter.content.trim())).length
  const missingWords = chapters.filter((chapter) => !Number(chapter.word_count)).length
  return {
    count: chapters.length,
    words,
    wordsLabel: words > 0 ? `${formatNumber(words)} 字` : '待采集',
    chaptersWithContent,
    missingWords,
  }
})
const importContentStatus = computed(() => {
  if (importChapterStats.value.chaptersWithContent > 0) {
    const handoffText = visibleImportHandoffLabels.value.length
      ? `可继续用于${visibleImportHandoffLabels.value.join('、')}。`
      : '可在作品归档中查看正文。'
    return {
      className: 'content-ready',
      title: '全本内容已入库',
      description: `已有 ${importChapterStats.value.chaptersWithContent} 章正文，${handoffText}`,
    }
  }
  return {
    className: 'catalog-only',
    title: '目录已入库',
    description: '正文待采集，当前适合先做结构浏览和章节规划。',
  }
})
const selectedStatusText = computed(() => {
  if (!selected.value) return ''
  if (selected.value.type === 'import') {
    return `${importContentStatus.value.title}，${importChapterStats.value.count} 章，${importChapterStats.value.wordsLabel}。`
  }
  return `${typeLabel(selected.value.type)}结果已归档，可继续复盘或送入下一步。`
})
const selectedActionHint = computed(() => {
  if (!selected.value) return ''
  if (selected.value.type === 'import') {
    return visibleImportHandoffLabels.value.length
      ? `把导入作品送去${visibleImportHandoffLabels.value.join('、')}继续加工。`
      : '当前下游功能入口已关闭，可先在作品归档中查看导入内容。'
  }
  if (selected.value.type === 'cover') return '封面可放入作品档案，下一步补齐简介和卖点。'
  if (isStudioVisible('fanqie')) return '回到热榜选样本，或把当前结论推进到下一版正文。'
  return '当前结论已归档，可先复制结果或等待管理员开放更多入口。'
})

async function load() {
  loading.value = true
  try {
    works.value = await listWorks(activeType.value === 'all' ? undefined : activeType.value)
    if (works.value.length) {
      await select(works.value[0])
    }
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
.works-page {
  display: flex;
  flex-direction: column;
  width: min(100%, 118rem);
  max-width: calc(100vw - 1.25rem);
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
}

.studio-wide-shell {
  width: min(100%, 118rem);
  max-width: calc(100vw - 1.25rem);
}

.works-head {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-end;
  margin-bottom: 1rem;
}

.eyebrow {
  display: inline-flex;
  border: 1px solid rgba(232, 65, 46, 0.28);
  border-radius: 999px;
  padding: 0.25rem 0.7rem;
  color: #c8351f;
  font-size: 0.74rem;
  font-weight: 900;
  text-transform: uppercase;
}

.eyebrow.small {
  font-size: 0.68rem;
}

.works-head h1 {
  margin: 0.55rem 0 0.35rem;
  font-size: 1.85rem;
  font-weight: 950;
  letter-spacing: 0;
}

.works-head p,
.archive-head p {
  max-width: 50rem;
  color: #665854;
  line-height: 1.75;
}

.quick-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.quick-actions a,
.type-tabs button {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.58rem 0.8rem;
  background: #fffdfb;
  color: #493a35;
  font-weight: 850;
}

.quick-actions a.primary {
  border-color: #e8412e;
  background: #e8412e;
  color: #fff;
}

.mission-control {
  order: 1;
  display: grid;
  grid-template-columns: 0.9fr 1fr 1.15fr;
  gap: 0.75rem;
  margin: 1rem 0;
}

.mission-panel,
.spine-band article,
.panel,
.archive-section {
  border: 1px solid #eaded8;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 18px 50px rgba(54, 32, 24, 0.08);
}

.mission-panel {
  display: grid;
  gap: 0.45rem;
  padding: 0.9rem;
  box-shadow: none;
}

.mission-panel strong {
  color: #241a16;
  font-size: 1rem;
  font-weight: 950;
}

.mission-panel p {
  color: #665854;
  line-height: 1.65;
}

.mission-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.mission-actions a {
  border: 1px solid #eaded8;
  border-radius: 999px;
  padding: 0.45rem 0.65rem;
  background: #fffdfb;
  color: #493a35;
  font-size: 0.78rem;
  font-weight: 900;
}

.mission-actions a.primary {
  border-color: #e8412e;
  background: #e8412e;
  color: #fff;
}

.spine-band article {
  padding: 0.72rem 0.85rem;
  box-shadow: none;
}

.spine-band span {
  color: #c8351f;
  font-size: 0.78rem;
  font-weight: 950;
}

.spine-band strong {
  display: block;
  margin-top: 0.2rem;
  font-weight: 950;
}

.spine-band p {
  margin-top: 0.25rem;
  color: #665854;
  line-height: 1.55;
}

.workbench-grid {
  order: 2;
  display: grid;
  grid-template-columns: minmax(220px, 0.78fr) minmax(0, 1.44fr) minmax(220px, 0.78fr);
  gap: 1rem;
}

.workbench-grid-wide {
  grid-template-columns: minmax(16rem, 0.86fr) minmax(0, 1.58fr) minmax(16rem, 0.86fr);
}

.panel {
  padding: 0.95rem;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: baseline;
  margin-bottom: 0.75rem;
}

.panel-head span {
  font-weight: 950;
}

.panel-head small {
  color: #9f7a6d;
  font-weight: 800;
}

.chapter-panel ol {
  display: grid;
  gap: 0.6rem;
}

.chapter-panel li {
  display: grid;
  grid-template-columns: 2rem 1fr;
  gap: 0.65rem;
  border: 1px solid #f0e5df;
  border-radius: 8px;
  padding: 0.7rem;
}

.chapter-panel b {
  width: 2rem;
  height: 2rem;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: #241a16;
  color: #fff;
  font-size: 0.78rem;
}

.chapter-panel li.warn b {
  background: #e8412e;
}

.chapter-panel p,
.setting-list strong,
.todo-panel li {
  color: #665854;
  line-height: 1.6;
}

.workbench-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin-bottom: 0.85rem;
}

.workbench-tabs button {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.48rem 0.65rem;
  background: #fffdfb;
  color: #493a35;
  font-weight: 850;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.65rem;
}

.metric-grid article,
.todo-panel,
.setting-list article {
  border: 1px solid #f0e5df;
  border-radius: 8px;
  padding: 0.75rem;
  background: #fffdfb;
}

.metric-grid span,
.setting-list span {
  color: #9f7a6d;
  font-size: 0.78rem;
  font-weight: 850;
}

.metric-grid strong {
  display: block;
  margin-top: 0.25rem;
  font-weight: 950;
}

.metric-grid p {
  margin-top: 0.25rem;
  color: #665854;
  line-height: 1.55;
}

.todo-panel {
  margin-top: 0.75rem;
}

.todo-panel ul,
.list-group ul {
  padding-left: 1.05rem;
}

.setting-list {
  display: grid;
  gap: 0.6rem;
}

.setting-list strong {
  display: block;
  margin-top: 0.2rem;
}

.archive-section {
  order: 3;
  margin-top: 1rem;
  padding: 1rem;
}

.archive-head {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-end;
  margin-bottom: 1rem;
}

.archive-head h2 {
  margin-top: 0.35rem;
  font-size: 1.35rem;
  font-weight: 950;
}

.type-tabs {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.45rem;
}

.type-tabs button.active {
  border-color: #241a16;
  background: #241a16;
  color: #fff;
}

.archive-grid {
  display: grid;
  grid-template-columns: minmax(280px, 0.58fr) minmax(0, 1.62fr);
  gap: 1rem;
}

.archive-grid-wide {
  grid-template-columns: minmax(20rem, 0.62fr) minmax(0, 1.78fr);
}

.archive-list ul {
  display: grid;
  gap: 0.5rem;
}

.work-card {
  width: 100%;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 0.6rem;
  align-items: center;
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.68rem;
  background: #fffdfb;
  color: #493a35;
}

.work-card.active {
  border-color: #e8412e;
  box-shadow: 0 0 0 3px rgba(232, 65, 46, 0.1);
}

.work-card strong {
  min-width: 0;
  overflow: hidden;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.work-card small {
  color: #8b7a74;
  font-size: 0.75rem;
}

.work-badge {
  border-radius: 999px;
  padding: 0.22rem 0.48rem;
  background: #fff7f3;
  color: #c8351f;
  font-size: 0.72rem;
  font-weight: 950;
}

.badge-import {
  background: #f8fbff;
  color: #2f5d9f;
}

.badge-hotspot {
  background: #fff7f3;
  color: #c8351f;
}

.badge-generate,
.badge-script {
  background: #f2fbf6;
  color: #1c755b;
}

.badge-cover {
  background: #f6f0ff;
  color: #6c45a6;
}

.archive-empty {
  min-height: 12rem;
  display: grid;
  place-items: center;
  border: 1px solid #f0e5df;
  border-radius: 8px;
  color: #8b7a74;
  text-align: center;
}

.archive-empty.dashed {
  border-style: dashed;
}

.detail-panel {
  min-height: 26rem;
}

.detail-head {
  border-radius: 8px;
  padding: 0.9rem;
  margin-bottom: 0.8rem;
  background: #241a16;
  color: #fff;
}

.detail-head span {
  color: #f0c7bd;
  font-size: 0.78rem;
  font-weight: 900;
}

.detail-head h2 {
  margin-top: 0.25rem;
  font-size: 1.3rem;
  font-weight: 950;
}

.detail-head p {
  margin-top: 0.2rem;
  color: #f7ede4;
}

.cover-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

.cover-grid figure {
  overflow: hidden;
  border: 1px solid #eaded8;
  border-radius: 8px;
  aspect-ratio: 3 / 4;
}

.work-cover-thumb,
.work-cover-thumb img {
  width: 100%;
  height: 100%;
  display: block;
}

.work-cover-thumb img {
  object-fit: cover;
}

.detail-block {
  display: grid;
  gap: 0.75rem;
}

.score-inline {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.score-inline strong {
  color: #e8412e;
  font-size: 2.5rem;
  font-weight: 950;
}

.score-inline span {
  font-weight: 900;
}

.detail-block > p,
.section-output p {
  color: #594843;
  line-height: 1.75;
}

.list-group,
.section-output,
.chapter-summary article {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.8rem;
  background: #fffdfb;
}

.list-group h3,
.section-output h3 {
  margin-bottom: 0.45rem;
  font-size: 0.95rem;
  font-weight: 950;
}

.list-group li {
  color: #594843;
  line-height: 1.75;
}

.tone-danger {
  border-color: rgba(232, 65, 46, 0.28);
  background: #fff7f3;
}

.tone-success {
  border-color: rgba(28, 117, 91, 0.24);
  background: #f2fbf6;
}

.chapter-summary {
  display: grid;
  gap: 0.55rem;
}

.chapter-summary.scrollable-preview {
  max-height: 34rem;
  overflow-y: auto;
  padding-right: 0.35rem;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  mask-image: linear-gradient(to bottom, #000 0, #000 calc(100% - 1.6rem), transparent 100%);
}

.chapter-summary.scrollable-preview::-webkit-scrollbar {
  width: 0.45rem;
}

.chapter-summary.scrollable-preview::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(232, 65, 46, 0.28);
}

.import-overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.6rem;
}

.import-overview article,
.import-stage-rail article {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.75rem;
  background: #fffdfb;
}

.import-overview span {
  display: block;
  color: #9f7a6d;
  font-size: 0.78rem;
  font-weight: 850;
}

.import-overview strong {
  display: block;
  margin-top: 0.25rem;
  color: #241a16;
  font-weight: 950;
}

.import-note {
  border: 1px solid rgba(28, 117, 91, 0.22);
  border-radius: 8px;
  padding: 0.75rem;
  background: #f6fffa;
  color: #476257;
  line-height: 1.75;
}

.import-note-soft {
  border-color: rgba(232, 65, 46, 0.18);
  background: #fff7f3;
  color: #7d4b40;
}

.import-stage-rail {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.55rem;
}

.import-stage-rail article {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.45rem 0.65rem;
  align-items: center;
}

.import-stage-rail b {
  grid-row: span 2;
  width: 2rem;
  height: 2rem;
  display: grid;
  place-items: center;
  border-radius: 999px;
  background: #241a16;
  color: #fff;
  font-size: 0.76rem;
  font-weight: 950;
}

.import-stage-rail strong {
  color: #241a16;
  font-weight: 950;
}

.import-stage-rail span {
  color: #7d625a;
  font-size: 0.78rem;
  font-weight: 850;
}

.import-stage-rail article.done {
  border-color: rgba(28, 117, 91, 0.24);
  background: #f6fffa;
}

.import-stage-rail article.done b {
  background: #1c755b;
}

.import-stage-rail article.warn {
  border-color: rgba(232, 65, 46, 0.2);
  background: #fff7f3;
}

.import-stage-rail article.warn b {
  background: #e8412e;
}

.import-action-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
  align-items: center;
  justify-content: space-between;
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.75rem;
  background: #fffdfb;
}

.import-action-strip div {
  display: grid;
  gap: 0.2rem;
}

.import-action-strip span {
  width: fit-content;
  border-radius: 999px;
  padding: 0.18rem 0.55rem;
  color: #fff;
  font-size: 0.72rem;
  font-weight: 950;
}

.import-action-strip .content-ready {
  background: #1c755b;
}

.import-action-strip .catalog-only {
  background: #e8412e;
}

.import-action-strip strong {
  color: #241a16;
  font-weight: 900;
}

.import-action-strip button {
  border: 1px solid #eaded8;
  border-radius: 999px;
  padding: 0.48rem 0.72rem;
  background: #fff;
  color: #493a35;
  font-weight: 900;
}

.chapter-summary article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 12rem);
  gap: 0.7rem;
  align-items: center;
}

.chapter-summary strong,
.chapter-summary small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chapter-summary span {
  color: #c8351f;
  font-weight: 850;
}

.chapter-summary span.pending {
  color: #9f7a6d;
}

.chapter-summary small {
  color: #8b7a74;
}

.dark .works-page,
.dark .works-head h1,
.dark .archive-head h2 {
  color: #f7ede4;
}

.dark .works-head p,
.dark .archive-head p,
.dark .spine-band p,
.dark .mission-panel p,
.dark .chapter-panel p,
.dark .metric-grid p {
  color: #cdbdb5;
}

.dark .mission-panel strong,
.dark .import-overview strong,
.dark .import-stage-rail strong,
.dark .import-action-strip strong {
  color: #fff7ed;
}

.dark .spine-band article,
.dark .mission-panel,
.dark .panel,
.dark .archive-section,
.dark .work-card,
.dark .metric-grid article,
.dark .todo-panel,
.dark .setting-list article,
.dark .list-group,
.dark .section-output,
.dark .import-overview article,
.dark .import-stage-rail article,
.dark .import-action-strip,
.dark .chapter-summary article {
  border-color: #312720;
  background: #171311;
}

.dark .import-stage-rail span {
  color: #cdbdb5;
}

@media (max-width: 1080px) {
  .workbench-grid,
  .mission-control,
  .archive-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .works-head,
  .archive-head,
  .spine-band,
  .import-overview,
  .import-stage-rail,
  .metric-grid {
    grid-template-columns: 1fr;
  }

  .works-head,
  .archive-head {
    align-items: flex-start;
  }

  .type-tabs {
    justify-content: flex-start;
  }
}

@media (max-width: 640px) {
  .works-page {
    padding: 0.25rem 0 1.25rem;
  }

  .works-head {
    gap: 0.65rem;
    margin-bottom: 0.65rem;
  }

  .works-head h1 {
    font-size: 1.55rem;
  }

  .works-head p,
  .archive-head p {
    font-size: 0.88rem;
    line-height: 1.6;
  }

  .quick-actions,
  .mission-actions {
    display: grid;
    width: 100%;
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .quick-actions a,
  .mission-actions a {
    padding: 0.48rem 0.35rem;
    text-align: center;
    font-size: 0.76rem;
  }

  .mission-control-mobile-compact {
    gap: 0.55rem;
    margin: 0.7rem 0;
  }

  .mission-control-mobile-compact .mission-panel {
    gap: 0.25rem;
    padding: 0.72rem;
  }

  .mission-control-mobile-compact .mission-panel p {
    font-size: 0.84rem;
    line-height: 1.48;
  }

  .workbench-grid,
  .archive-grid-mobile-flow {
    gap: 0.75rem;
  }

  .panel,
  .archive-section-mobile-stack {
    border-radius: 10px;
    padding: 0.75rem;
  }

  .chapter-panel ol,
  .archive-list ul {
    gap: 0.5rem;
  }

  .chapter-panel li {
    grid-template-columns: 1.75rem 1fr;
    padding: 0.55rem;
  }

  .chapter-panel b {
    width: 1.75rem;
    height: 1.75rem;
    font-size: 0.72rem;
  }

  .metric-grid article,
  .todo-panel,
  .setting-list article {
    padding: 0.62rem;
  }

  .archive-head {
    gap: 0.55rem;
    margin-bottom: 0.75rem;
  }

  .archive-head h2 {
    font-size: 1.15rem;
  }

  .type-tabs {
    display: grid;
    width: 100%;
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .type-tabs button {
    padding: 0.42rem 0.3rem;
    font-size: 0.76rem;
  }

  .archive-list ul {
    display: flex;
    overflow-x: auto;
    padding-bottom: 0.25rem;
    scroll-snap-type: x mandatory;
  }

  .archive-list li {
    min-width: 78%;
    scroll-snap-align: start;
  }

  .archive-list .work-card {
    min-height: 5rem;
  }

  .detail-head {
    padding: 0.75rem;
  }

  .import-action-strip {
    display: grid;
  }

  .import-action-strip button {
    width: 100%;
  }

  .chapter-summary.scrollable-preview {
    max-height: 24rem;
  }

  .work-card,
  .chapter-summary article,
  .cover-grid {
    grid-template-columns: 1fr;
  }
}
</style>
