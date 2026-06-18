<template>
  <AppLayout>
    <div class="works-page mx-auto max-w-7xl" data-test="studio-workbench-shell">
      <header class="works-head">
        <div>
          <span class="eyebrow">作品归档</span>
          <h1>我的作品</h1>
          <p>集中查看拆书、爆款对标、创作生成和封面历史；先筛选，再点开右侧详情继续加工。</p>
        </div>
        <div class="quick-actions" aria-label="作品快捷入口">
          <router-link to="/studio/fanqie">查看热榜</router-link>
          <router-link to="/studio/teardown">拆书诊断</router-link>
          <router-link class="primary" to="/studio/generate">创作生成</router-link>
        </div>
      </header>

      <section class="spine-band" aria-label="创作流水线">
        <article v-for="step in spineSteps" :key="step.title">
          <span>{{ step.index }}</span>
          <strong>{{ step.title }}</strong>
          <p>{{ step.desc }}</p>
        </article>
      </section>

      <section class="workbench-grid">
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

      <section class="archive-section" aria-label="最近产出归档">
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

        <div class="archive-grid">
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
                <div class="chapter-summary">
                  <article v-for="chapter in imported.chapters || []" :key="chapter.title + chapter.source">
                    <strong>{{ chapter.title }}</strong>
                    <span>{{ chapter.word_count || '-' }} 字</span>
                    <small>{{ chapter.source || '粘贴内容' }}</small>
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
import AppLayout from '@/components/layout/AppLayout.vue'
import CoverImageViewer from '@/components/studio/CoverImageViewer.vue'
import { getWork, listWorks, type CoverImage, type WorkDetail, type WorkItem } from '@/api/studio'

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

const tabs = [
  { value: 'all', label: '全部' },
  { value: 'import', label: '导入' },
  { value: 'teardown', label: '拆书' },
  { value: 'hotspot', label: '爆款' },
  { value: 'generate', label: '生成' },
  { value: 'script', label: '剧本' },
  { value: 'cover', label: '封面' },
]

const spineSteps = [
  { index: '01', title: '找样本', desc: '从番茄热榜挑选题材和结构参照' },
  { index: '02', title: '诊断', desc: '拆节奏、爽点、人物和伏笔' },
  { index: '03', title: '对标', desc: '比同题材样本，找爆款差距' },
  { index: '04', title: '生成', desc: '产出大纲、正文、改写和脚本' },
]

const projectMetrics = [
  { label: '综合诊断', value: '待分析', hint: '从拆书报告自动汇总' },
  { label: '爆款潜力', value: '待对标', hint: '对标后形成雷达' },
  { label: '设定一致性', value: '待沉淀', hint: '导入章节后建立设定库' },
]

const workbenchTabs = ['概览', '拆书', '爆款', '大纲', '正文', '剧本', '热榜']

const chapterNodes = [
  { index: '01', title: '导入首章', note: '先把开篇送去拆书，确定钩子和节奏。', state: 'good' },
  { index: '02', title: '诊断缺口', note: '用爆款对标确认爽点兑现是否偏慢。', state: 'warn' },
  { index: '03', title: '生成修订', note: '把改法喂给创作生成台，形成新版本。', state: 'good' },
]

const settingGroups = [
  { label: '人物卡', value: '从导入章节抽取' },
  { label: '金手指', value: '拆书后沉淀规则与代价' },
  { label: '世界观', value: '记录地点、势力、限制' },
  { label: '伏笔表', value: '跟踪埋设与回收' },
]

const todoItems = [
  '先从热榜找 3 个同题材样本，确认读者正在追什么。',
  '用拆书报告找出黄金三章的掉线位置。',
  '把爆款对标动作直接带进创作生成，形成下一版正文。',
]

const activeType = ref('all')
const works = ref<WorkItem[]>([])
const loading = ref(false)
const selected = ref<WorkDetail | null>(null)
const detailLoading = ref(false)
const viewerOpen = ref(false)
const viewerIndex = ref(0)

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
  chapters?: { title: string; word_count: number; source?: string }[]
  next_actions?: string[]
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
.works-page {
  display: flex;
  flex-direction: column;
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
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

.spine-band {
  order: 2;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.75rem;
  margin: 1rem 0;
}

.spine-band article,
.panel,
.archive-section {
  border: 1px solid #eaded8;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 18px 50px rgba(54, 32, 24, 0.08);
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
  order: 3;
  display: grid;
  grid-template-columns: minmax(220px, 0.78fr) minmax(0, 1.44fr) minmax(220px, 0.78fr);
  gap: 1rem;
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
  order: 1;
  margin-top: 0;
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
  grid-template-columns: minmax(260px, 0.75fr) minmax(0, 1.25fr);
  gap: 1rem;
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
.dark .chapter-panel p,
.dark .metric-grid p {
  color: #cdbdb5;
}

.dark .spine-band article,
.dark .panel,
.dark .archive-section,
.dark .work-card,
.dark .metric-grid article,
.dark .todo-panel,
.dark .setting-list article,
.dark .list-group,
.dark .section-output,
.dark .chapter-summary article {
  border-color: #312720;
  background: #171311;
}

@media (max-width: 1080px) {
  .workbench-grid,
  .archive-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .works-head,
  .archive-head,
  .spine-band,
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
  .work-card,
  .chapter-summary article,
  .cover-grid {
    grid-template-columns: 1fr;
  }
}
</style>
