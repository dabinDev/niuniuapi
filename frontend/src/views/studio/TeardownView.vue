<template>
  <AppLayout>
    <div class="teardown-page mx-auto max-w-7xl">
      <header class="teardown-head">
        <div>
          <span class="eyebrow">Diagnosis Board</span>
          <h1>拆书诊断</h1>
          <p>把章节拆成可执行的质检报告：黄金三章、节奏热区、人物钩子、伏笔追踪和下一步改法分开呈现。</p>
        </div>
        <router-link class="head-link" to="/studio/hotspot">去爆款对标</router-link>
      </header>

      <div class="model-strip" :class="{ ready: configured }">
        <span>{{ configured ? '文案模型' : '还没配置文案模型' }}</span>
        <strong>{{ configured ? textModel : '请先到 API 密钥页设置文案模型' }}</strong>
        <router-link to="/keys">模型设置</router-link>
      </div>

      <section class="diagnosis-map" aria-label="拆书诊断模块">
        <article v-for="item in diagnosisModules" :key="item.title">
          <span>{{ item.kicker }}</span>
          <strong>{{ item.title }}</strong>
          <p>{{ item.desc }}</p>
        </article>
      </section>

      <section class="teardown-grid">
        <form class="input-panel" @submit.prevent="submit">
          <label class="stacked" for="td-content">
            <span>小说正文 / 章节</span>
            <textarea
              id="td-content"
              v-model="content"
              rows="14"
              placeholder="粘贴要拆解的小说正文，至少 100 字。建议先从开篇 1-3 章开始。"
            ></textarea>
          </label>

          <div class="char-row">
            <span :class="{ 'char-warn': contentLength > 0 && contentLength < MIN_LEN }">
              {{ contentLength }} 字{{ contentLength < MIN_LEN ? `（至少 ${MIN_LEN}）` : '' }}
            </span>
          </div>

          <div class="field-grid">
            <label>
              <span>书名</span>
              <input id="td-title" v-model="title" type="text" placeholder="例如：北境书塔" />
            </label>
            <label>
              <span>题材</span>
              <input id="td-genre" v-model="genre" type="text" placeholder="玄幻 / 都市 / 女频" />
            </label>
          </div>

          <div class="tone-row" aria-label="点评尺度">
            <button
              v-for="opt in toneOptions"
              :key="opt.value"
              type="button"
              :class="{ active: tone === opt.value }"
              @click="tone = opt.value"
            >
              {{ opt.label }}
            </button>
          </div>

          <button type="button" class="submit-btn" :disabled="!canSubmit" @click="submit">
            {{ loading ? '正在拆书...' : '开始拆书' }}
          </button>

          <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>
        </form>

        <section class="result-panel">
          <div v-if="loading" class="empty-state">
            <div class="scan-line"></div>
            <p>正在检查开篇钩子、节奏热区、爽点兑现和伏笔风险。</p>
          </div>

          <template v-else-if="report">
            <div class="score-board">
              <div>
                <span>综合诊断</span>
                <strong>{{ report.overall_score }}<small>/100</small></strong>
              </div>
              <p>{{ report.verdict }}</p>
            </div>

            <p class="summary">{{ report.summary }}</p>

            <div v-if="report.scores.length" class="score-grid">
              <article v-for="s in report.scores" :key="s.label">
                <span>{{ s.label }}</span>
                <div><i :style="{ width: s.value + '%' }"></i></div>
                <strong>{{ s.value }}</strong>
              </article>
            </div>

            <div class="report-columns">
              <section>
                <h2>亮点</h2>
                <ul><li v-for="(h, i) in report.highlights" :key="i">{{ h }}</li></ul>
              </section>
              <section class="danger">
                <h2>烂点</h2>
                <ul><li v-for="(r, i) in report.rotten_points" :key="i">{{ r }}</li></ul>
              </section>
              <section class="success">
                <h2>改法</h2>
                <ul><li v-for="(g, i) in report.suggestions" :key="i">{{ g }}</li></ul>
              </section>
            </div>
          </template>

          <div v-else class="empty-state">
            <div class="empty-mark">QC</div>
            <p>提交章节后，这里会生成评分、烂点、改法，并自动进入“我的作品”。</p>
          </div>
        </section>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { analyzeTeardown, getModelConfig, type TeardownReport, type TeardownTone } from '@/api/studio'

const MIN_LEN = 100

const content = ref('')
const title = ref('')
const genre = ref('')
const tone = ref<TeardownTone>('savage')
const loading = ref(false)
const report = ref<TeardownReport | null>(null)
const errorMsg = ref('')
const configured = ref(false)
const textModel = ref('')

const diagnosisModules = [
  { kicker: '01', title: '黄金三章', desc: '专看开篇钩子、压迫感和首次爽点兑现。' },
  { kicker: '02', title: '节奏热区', desc: '标出拖沓、信息堆叠和回报偏慢的位置。' },
  { kicker: '03', title: '伏笔追踪', desc: '把未回收伏笔和设定风险拉成清单。' },
  { kicker: '04', title: '改法队列', desc: '把建议变成可喂给创作生成的动作。' },
]

const toneOptions: { value: TeardownTone; label: string }[] = [
  { value: 'savage', label: '毒舌' },
  { value: 'neutral', label: '中性' },
  { value: 'gentle', label: '鼓励' },
]

const contentLength = computed(() => content.value.trim().length)
const canSubmit = computed(() => configured.value && contentLength.value >= MIN_LEN && !loading.value)

onMounted(async () => {
  try {
    const cfg = await getModelConfig()
    if (cfg.text) {
      configured.value = true
      textModel.value = cfg.text.model
    }
  } catch {
    configured.value = false
  }
})

async function submit() {
  if (!canSubmit.value) return
  loading.value = true
  errorMsg.value = ''
  report.value = null
  try {
    report.value = await analyzeTeardown({
      content: content.value.trim(),
      title: title.value.trim() || undefined,
      genre: genre.value.trim() || undefined,
      tone: tone.value,
    })
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
    errorMsg.value = msg || '拆书失败，请稍后重试。'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.teardown-page {
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
}

.teardown-head {
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

.teardown-head h1 {
  margin: 0.55rem 0 0.35rem;
  font-size: 2.05rem;
  font-weight: 950;
  letter-spacing: 0;
}

.teardown-head p {
  max-width: 50rem;
  color: #665854;
  line-height: 1.75;
}

.head-link,
.model-strip a {
  color: #c8351f;
  font-weight: 800;
}

.model-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 0.7rem;
  align-items: center;
  justify-content: space-between;
  border: 1px dashed rgba(232, 65, 46, 0.34);
  border-radius: 8px;
  padding: 0.75rem 0.9rem;
  margin-bottom: 1rem;
  background: #fff7f3;
  color: #8c3024;
}

.model-strip.ready {
  border-color: rgba(28, 117, 91, 0.25);
  background: #f2fbf6;
  color: #1c755b;
}

.diagnosis-map {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.diagnosis-map article,
.input-panel,
.result-panel {
  border: 1px solid #eaded8;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 18px 50px rgba(54, 32, 24, 0.08);
}

.diagnosis-map article {
  padding: 0.9rem;
}

.diagnosis-map span {
  color: #c8351f;
  font-size: 0.78rem;
  font-weight: 950;
}

.diagnosis-map strong {
  display: block;
  margin-top: 0.2rem;
  font-weight: 950;
}

.diagnosis-map p {
  margin-top: 0.25rem;
  color: #665854;
  line-height: 1.55;
}

.teardown-grid {
  display: grid;
  grid-template-columns: minmax(340px, 0.88fr) minmax(0, 1.12fr);
  gap: 1rem;
}

.input-panel,
.result-panel {
  padding: 1rem;
}

label span {
  display: block;
  margin-bottom: 0.35rem;
  color: #6e5a54;
  font-size: 0.8rem;
  font-weight: 800;
}

input,
textarea {
  width: 100%;
  border: 1px solid #e6d6ce;
  border-radius: 8px;
  padding: 0.72rem 0.78rem;
  background: #fffdfb;
  color: #231917;
  outline: none;
}

textarea {
  resize: vertical;
  line-height: 1.75;
}

input:focus,
textarea:focus {
  border-color: #e8412e;
  box-shadow: 0 0 0 3px rgba(232, 65, 46, 0.12);
}

.field-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.7rem;
  margin-top: 0.8rem;
}

.char-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.45rem;
  color: #8b7a74;
  font-size: 0.85rem;
  font-weight: 800;
}

.char-warn {
  color: #c8351f;
}

.tone-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  margin: 0.85rem 0;
}

.tone-row button {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.58rem;
  background: #fff;
  color: #614d47;
  font-weight: 800;
}

.tone-row button.active {
  border-color: #241a16;
  background: #241a16;
  color: #fff;
}

.submit-btn {
  width: 100%;
  border: 0;
  border-radius: 8px;
  padding: 0.82rem 1.2rem;
  background: #e8412e;
  color: #fff;
  font-weight: 950;
}

.submit-btn:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.error-text {
  margin-top: 0.8rem;
  color: #c8351f;
  font-weight: 700;
}

.result-panel {
  min-height: 38rem;
}

.empty-state {
  min-height: 32rem;
  display: grid;
  place-content: center;
  gap: 1rem;
  text-align: center;
  color: #7a6962;
}

.empty-mark {
  width: 4.2rem;
  height: 4.2rem;
  display: grid;
  place-items: center;
  margin: 0 auto;
  border-radius: 8px;
  background: #241a16;
  color: #fff;
  font-weight: 950;
}

.scan-line {
  width: 14rem;
  height: 0.55rem;
  overflow: hidden;
  border-radius: 999px;
  background: #f1e4dd;
}

.scan-line::after {
  content: '';
  display: block;
  width: 35%;
  height: 100%;
  border-radius: inherit;
  background: #e8412e;
  animation: scan 1.2s linear infinite;
}

@keyframes scan {
  from { transform: translateX(-100%); }
  to { transform: translateX(300%); }
}

.score-board {
  display: grid;
  grid-template-columns: 12rem 1fr;
  gap: 1rem;
  align-items: center;
  border-radius: 8px;
  padding: 1rem;
  background: #241a16;
  color: #fff;
}

.score-board span {
  color: #f0c7bd;
  font-size: 0.82rem;
  font-weight: 800;
}

.score-board strong {
  display: block;
  font-size: 3.2rem;
  font-weight: 950;
}

.score-board small {
  font-size: 1rem;
  color: #f0c7bd;
}

.score-board p {
  line-height: 1.8;
}

.summary {
  margin: 0.9rem 0;
  color: #594843;
  line-height: 1.8;
}

.score-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.65rem;
}

.score-grid article,
.report-columns section {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.8rem;
  background: #fffdfb;
}

.score-grid span {
  color: #6e5a54;
  font-size: 0.82rem;
  font-weight: 850;
}

.score-grid div {
  height: 0.5rem;
  overflow: hidden;
  border-radius: 999px;
  margin: 0.5rem 0;
  background: #f0e2dc;
}

.score-grid i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #e8412e, #1c755b);
}

.score-grid strong {
  font-weight: 950;
}

.report-columns {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.7rem;
  margin-top: 0.85rem;
}

.report-columns h2 {
  margin-bottom: 0.5rem;
  font-size: 0.95rem;
  font-weight: 950;
}

.report-columns ul {
  padding-left: 1.05rem;
  color: #594843;
  line-height: 1.75;
}

.report-columns .danger {
  border-color: rgba(232, 65, 46, 0.28);
  background: #fff7f3;
}

.report-columns .success {
  border-color: rgba(28, 117, 91, 0.24);
  background: #f2fbf6;
}

.dark .teardown-page,
.dark .teardown-head h1 {
  color: #f7ede4;
}

.dark .teardown-head p,
.dark .diagnosis-map p,
.dark .empty-state,
.dark label span {
  color: #cdbdb5;
}

.dark .diagnosis-map article,
.dark .input-panel,
.dark .result-panel,
.dark .score-grid article,
.dark .report-columns section {
  border-color: #312720;
  background: #171311;
}

.dark input,
.dark textarea,
.dark .tone-row button {
  border-color: #3a2d26;
  background: #211916;
  color: #f7ede4;
}

@media (max-width: 980px) {
  .teardown-head,
  .diagnosis-map,
  .teardown-grid,
  .score-board,
  .report-columns {
    grid-template-columns: 1fr;
  }

  .teardown-head {
    align-items: flex-start;
  }
}

@media (max-width: 640px) {
  .field-grid,
  .tone-row,
  .score-grid {
    grid-template-columns: 1fr;
  }
}
</style>
