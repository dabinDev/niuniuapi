<template>
  <AppLayout>
    <div class="hotspot-page mx-auto max-w-7xl">
      <header class="studio-hero">
        <div>
          <span class="eyebrow">Benchmark Lab</span>
          <h1>爆款对标</h1>
          <p>把你的章节和对标样本放在同一张质检台上，找出题材卖点、爽点缺口和下一步能直接改的动作。</p>
        </div>
        <router-link class="hero-link" to="/studio/works">回到作品工作台</router-link>
      </header>

      <div
        class="model-strip"
        :class="{ 'model-strip-ready': configured }"
      >
        <span>{{ configured ? '文案模型已就绪' : '需要先配置文案模型' }}</span>
        <strong>{{ configured ? textModel : 'API 密钥页 -> 文案模型 -> 测试保存' }}</strong>
        <router-link to="/keys">去配置</router-link>
      </div>

      <section class="hotspot-grid">
        <form class="input-panel" @submit.prevent="submit">
          <div class="panel-title">
            <span>输入</span>
            <strong>我的作品片段</strong>
          </div>

          <div class="field-grid">
            <label>
              <span>书名</span>
              <input v-model="title" type="text" placeholder="例如：北境书塔" />
            </label>
            <label>
              <span>题材</span>
              <input v-model="genre" type="text" placeholder="玄幻 / 都市 / 女频复仇" />
            </label>
          </div>

          <div class="goal-row" aria-label="分析目标">
            <button
              v-for="opt in goalOptions"
              :key="opt.value"
              type="button"
              :class="{ active: goal === opt.value }"
              @click="goal = opt.value"
            >
              {{ opt.label }}
            </button>
          </div>

          <label class="stacked">
            <span>我的作品片段</span>
            <textarea
              v-model="content"
              data-test="hotspot-content"
              rows="10"
              placeholder="粘贴开篇、关键章节、拆书结论，至少 80 字。"
            ></textarea>
          </label>

          <label class="stacked">
            <span>对标样本</span>
            <textarea
              v-model="benchmark"
              data-test="hotspot-benchmark"
              rows="7"
              placeholder="粘贴同题材爆款片段、榜单观察、读者评论，或你想借鉴的结构。"
            ></textarea>
          </label>

          <div class="submit-row">
            <span :class="{ warn: contentLength > 0 && contentLength < MIN_LEN }">{{ contentLength }} 字</span>
            <button data-test="hotspot-submit" type="button" :disabled="!canSubmit" @click="submit">
              {{ loading ? '正在对标...' : '开始对标' }}
            </button>
          </div>
          <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>
        </form>

        <section class="result-panel">
          <div v-if="loading" class="empty-state">
            <div class="pulse-line"></div>
            <p>正在拆题材、钩子、爽点兑现和改编潜力。</p>
          </div>

          <template v-else-if="report">
            <div class="score-board">
              <div>
                <span>爆款潜力</span>
                <strong>{{ report.market_score }}<small>/100</small></strong>
              </div>
              <p>{{ report.verdict }}</p>
            </div>

            <div class="analysis-block">
              <h2>爆款雷达</h2>
              <div class="radar-list">
                <div v-for="item in report.radar" :key="item.label" class="radar-row">
                  <span>{{ item.label }}</span>
                  <div><i :style="{ width: item.value + '%' }"></i></div>
                  <b>{{ item.value }}</b>
                </div>
              </div>
            </div>

            <div class="three-col">
              <article>
                <h3>可复用套路</h3>
                <ul><li v-for="item in report.tropes" :key="item">{{ item }}</li></ul>
              </article>
              <article>
                <h3>差距</h3>
                <ul><li v-for="item in report.gaps" :key="item">{{ item }}</li></ul>
              </article>
              <article class="action-card">
                <h3>下一步动作</h3>
                <ol><li v-for="item in report.actions" :key="item">{{ item }}</li></ol>
              </article>
            </div>

            <div v-if="report.samples.length" class="sample-strip">
              <article v-for="sample in report.samples" :key="sample.title">
                <strong>{{ sample.title }}</strong>
                <p>{{ sample.lesson }}</p>
              </article>
            </div>
          </template>

          <div v-else class="empty-state">
            <div class="empty-mark">BM</div>
            <p>这里会输出爆款雷达、套路库、差距清单和可执行改法。</p>
          </div>
        </section>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  analyzeHotspot,
  getModelConfig,
  type HotspotReport,
  type HotspotRequest,
} from '@/api/studio'

const MIN_LEN = 80

const title = ref('')
const genre = ref('')
const content = ref('')
const benchmark = ref('')
const goal = ref<NonNullable<HotspotRequest['goal']>>('new-book')
const configured = ref(false)
const textModel = ref('')
const loading = ref(false)
const errorMsg = ref('')
const report = ref<HotspotReport | null>(null)

const goalOptions: { value: NonNullable<HotspotRequest['goal']>; label: string }[] = [
  { value: 'new-book', label: '新书立项' },
  { value: 'rewrite', label: '老书修订' },
  { value: 'short-video', label: '短剧/视频' },
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
    report.value = await analyzeHotspot({
      title: title.value.trim() || undefined,
      genre: genre.value.trim() || undefined,
      content: content.value.trim(),
      benchmark: benchmark.value.trim() || undefined,
      goal: goal.value,
    })
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
    errorMsg.value = msg || '爆款对标失败，请稍后重试。'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.hotspot-page {
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
}

.studio-hero {
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

.studio-hero h1 {
  margin: 0.55rem 0 0.35rem;
  font-size: 2.05rem;
  font-weight: 950;
  letter-spacing: 0;
}

.studio-hero p {
  max-width: 48rem;
  color: #665854;
  line-height: 1.75;
}

.hero-link,
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

.model-strip-ready {
  border-color: rgba(28, 117, 91, 0.25);
  background: #f2fbf6;
  color: #1c755b;
}

.hotspot-grid {
  display: grid;
  grid-template-columns: minmax(320px, 0.85fr) minmax(0, 1.15fr);
  gap: 1rem;
}

.input-panel,
.result-panel {
  border: 1px solid #eaded8;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.86);
  box-shadow: 0 18px 50px rgba(54, 32, 24, 0.08);
}

.input-panel {
  padding: 1rem;
}

.panel-title {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: baseline;
  margin-bottom: 0.8rem;
}

.panel-title span {
  color: #9f7a6d;
  font-size: 0.8rem;
  font-weight: 800;
}

.panel-title strong {
  font-size: 1rem;
}

.field-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.7rem;
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
  line-height: 1.7;
}

input:focus,
textarea:focus {
  border-color: #e8412e;
  box-shadow: 0 0 0 3px rgba(232, 65, 46, 0.12);
}

.stacked {
  display: block;
  margin-top: 0.8rem;
}

.goal-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  margin: 0.85rem 0 0.2rem;
}

.goal-row button {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.58rem;
  background: #fff;
  color: #614d47;
  font-weight: 800;
}

.goal-row button.active {
  border-color: #e8412e;
  background: #e8412e;
  color: #fff;
}

.submit-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 0.9rem;
}

.submit-row span {
  color: #8b7a74;
  font-size: 0.85rem;
  font-weight: 800;
}

.submit-row .warn {
  color: #c8351f;
}

.submit-row button {
  border: 0;
  border-radius: 8px;
  padding: 0.75rem 1.2rem;
  background: #241a16;
  color: #fff;
  font-weight: 900;
}

.submit-row button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.error-text {
  margin-top: 0.8rem;
  color: #c8351f;
  font-weight: 700;
}

.result-panel {
  min-height: 36rem;
  padding: 1rem;
}

.empty-state {
  min-height: 30rem;
  display: grid;
  place-content: center;
  gap: 1rem;
  text-align: center;
  color: #7a6962;
}

.empty-mark {
  width: 4rem;
  height: 4rem;
  display: grid;
  place-items: center;
  margin: 0 auto;
  border-radius: 50%;
  background: #241a16;
  color: #fff;
  font-weight: 950;
}

.pulse-line {
  width: 14rem;
  height: 0.55rem;
  overflow: hidden;
  border-radius: 999px;
  background: #f1e4dd;
}

.pulse-line::after {
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
  color: #f2b4a8;
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
  color: #f2b4a8;
}

.score-board p {
  line-height: 1.8;
}

.analysis-block {
  margin-top: 1rem;
}

.analysis-block h2,
.three-col h3 {
  margin-bottom: 0.7rem;
  font-size: 1rem;
  font-weight: 950;
}

.radar-list {
  display: grid;
  gap: 0.55rem;
}

.radar-row {
  display: grid;
  grid-template-columns: 6.5rem 1fr 2.2rem;
  gap: 0.7rem;
  align-items: center;
  color: #594843;
  font-size: 0.88rem;
  font-weight: 800;
}

.radar-row div {
  height: 0.55rem;
  overflow: hidden;
  border-radius: 999px;
  background: #f0e2dc;
}

.radar-row i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #e8412e, #1c755b);
}

.three-col {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.75rem;
  margin-top: 1rem;
}

.three-col article,
.sample-strip article {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.85rem;
  background: #fffdfb;
}

.three-col ul,
.three-col ol {
  padding-left: 1.05rem;
  color: #665854;
  line-height: 1.75;
}

.action-card {
  border-color: rgba(232, 65, 46, 0.34) !important;
  background: #fff7f3 !important;
}

.sample-strip {
  display: grid;
  gap: 0.7rem;
  margin-top: 1rem;
}

.sample-strip strong {
  display: block;
  margin-bottom: 0.25rem;
}

.sample-strip p {
  color: #665854;
  line-height: 1.65;
}

.dark .hotspot-page,
.dark .studio-hero h1 {
  color: #f7ede4;
}

.dark .studio-hero p,
.dark .empty-state,
.dark label span {
  color: #cdbdb5;
}

.dark .input-panel,
.dark .result-panel,
.dark .three-col article,
.dark .sample-strip article {
  border-color: #312720;
  background: #171311;
}

.dark input,
.dark textarea,
.dark .goal-row button {
  border-color: #3a2d26;
  background: #211916;
  color: #f7ede4;
}

@media (max-width: 980px) {
  .studio-hero,
  .hotspot-grid,
  .score-board,
  .three-col {
    grid-template-columns: 1fr;
  }

  .studio-hero {
    align-items: flex-start;
  }
}

@media (max-width: 640px) {
  .field-grid,
  .goal-row {
    grid-template-columns: 1fr;
  }

  .radar-row {
    grid-template-columns: 1fr;
  }
}
</style>
