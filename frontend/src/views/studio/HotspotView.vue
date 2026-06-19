<template>
  <AppLayout>
    <div class="hotspot-page mx-auto max-w-7xl">
      <header class="studio-hero">
        <div>
          <span class="eyebrow">Benchmark Lab</span>
          <h1>爆款对标</h1>
          <p>把你的章节和对标样本放在同一张质检台上，找出题材卖点、爽点缺口和下一步能直接改的动作。</p>
        </div>
        <router-link class="hero-link studio-action-link" to="/studio/works">回到作品工作台</router-link>
      </header>

      <div
        class="model-strip model-strip-compact"
        :class="{ 'model-strip-ready': configured }"
        data-test="model-auto-config-strip"
      >
        <span>{{ configured ? '文案模型已就绪' : '自动配置流程' }}</span>
        <strong>{{ configured ? textModel : '创建第一把密钥后，系统会优先自动选择第一把密钥的最新文案模型' }}</strong>
        <p class="model-strip-copy">{{ configured ? '如需覆盖默认值，可在 API 密钥页手动测试后保存。' : '自动选择最新文案模型；手动测试后保存会记住你的选择，不手动修改时创作台继续使用自动配置。' }}</p>
        <router-link to="/keys">{{ configured ? '去配置' : '检查密钥' }}</router-link>
      </div>

      <div v-if="bridgeNotice" class="bridge-notice" data-test="hotspot-bridge-notice">
        {{ bridgeNotice }}
      </div>

      <section class="hotspot-grid">
        <form class="input-panel" @submit.prevent="submit">
          <div class="panel-title">
            <span>输入</span>
            <strong>我的作品片段</strong>
          </div>

          <div class="template-strip" aria-label="爆款对标模板">
            <button
              v-for="tpl in benchmarkTemplates"
              :key="tpl.key"
              :data-test="`hotspot-template-${tpl.key}`"
              type="button"
              @click="applyBenchmarkTemplate(tpl.key)"
            >
              <strong>{{ tpl.title }}</strong>
              <span>{{ tpl.desc }}</span>
            </button>
          </div>

          <div class="material-map" data-test="hotspot-material-map" aria-label="对标素材配比">
            <div class="map-head">
              <span>素材配比</span>
              <strong>把“我”和“样本”摆在同一张桌上</strong>
            </div>
            <div class="map-grid">
              <article>
                <b>我的开篇</b>
                <small>1-3 章正文或拆书结论</small>
              </article>
              <article>
                <b>热榜样本</b>
                <small>同题材榜单、简介、前十章观察</small>
              </article>
              <article>
                <b>读者反馈</b>
                <small>评论关键词、追读点、弃文点</small>
              </article>
              <article>
                <b>改写目标</b>
                <small>要补的爽点、钩子或短剧冲突</small>
              </article>
            </div>
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
              rows="8"
              placeholder="粘贴开篇、关键章节、拆书结论，至少 80 字。"
            ></textarea>
          </label>

          <label class="stacked">
            <span>对标样本</span>
            <textarea
              v-model="benchmark"
              data-test="hotspot-benchmark"
              rows="6"
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
            <p class="model-strip-copy">正在拆题材、钩子、爽点兑现和改编潜力。</p>
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

            <div class="result-actions" data-test="hotspot-result-actions">
              <button data-test="hotspot-send-generate" type="button" @click="sendBenchmarkToGenerate">带动作生成正文</button>
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
            <div class="empty-kicker">
              <div class="empty-mark">BM</div>
              <span>先选一个参照系</span>
            </div>
            <h2>别空想爆款，先把样本摆上桌</h2>
            <p>这里会输出爆款雷达、套路库、差距清单和可执行改法。最适合放入你的开篇 + 热榜样本，快速判断差距在哪里。</p>
            <button data-test="hotspot-empty-sample" type="button" class="empty-action" @click="applyBenchmarkTemplate('same-genre')">
              先填入同题材对标示例
            </button>
            <div class="empty-guide" aria-label="爆款对标使用路径">
              <strong>对标路径</strong>
              <ol>
                <li>左侧放你的开篇、拆书结论或改稿目标。</li>
                <li>下方放热榜样本、读者评论或同题材结构。</li>
                <li>生成可复制的改写队列，再送去创作生成。</li>
              </ol>
            </div>
            <div class="empty-preview" aria-label="爆款对标结果预览">
              <article>
                <span>雷达</span>
                <strong>卖点强度</strong>
                <small>题材、钩子、爽点、反转节奏</small>
              </article>
              <article>
                <span>差距</span>
                <strong>对标样本</strong>
                <small>你的片段和样本之间的可复制差异</small>
              </article>
              <article>
                <span>动作</span>
                <strong>改写队列</strong>
                <small>能直接喂给创作生成的下一步</small>
              </article>
            </div>
          </div>
        </section>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  analyzeHotspot,
  getModelConfig,
  type HotspotReport,
  type HotspotRequest,
} from '@/api/studio'
import { consumeStudioBridgePayload, saveStudioBridgePayload } from '@/utils/studioBridge'

const MIN_LEN = 80

const router = useRouter()

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
const bridgeNotice = ref('')

const goalOptions: { value: NonNullable<HotspotRequest['goal']>; label: string }[] = [
  { value: 'new-book', label: '新书立项' },
  { value: 'rewrite', label: '老书修订' },
  { value: 'short-video', label: '短剧/视频' },
]

const benchmarkTemplates = [
  { key: 'same-genre', title: '同题材三章对标', desc: '热榜样本 + 我的开篇差距' },
  { key: 'short-video', title: '短剧化卖点', desc: '提炼强冲突和转折点' },
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
  applyBridgePayload()
})

function applyBridgePayload() {
  const payload = consumeStudioBridgePayload('hotspot')
  if (!payload) return
  title.value = payload.title
  genre.value = payload.genre
  content.value = payload.content
  benchmark.value = payload.benchmark
  goal.value = 'new-book'
  bridgeNotice.value = payload.source === 'works'
    ? '已从我的作品带入素材，可直接对照样本找差距。'
    : '已从番茄热榜带入素材，可直接生成爆款差距清单。'
}

function applyBenchmarkTemplate(key: string) {
  if (key === 'same-genre') {
    title.value = '雨夜入塔'
    genre.value = '玄幻悬疑'
    goal.value = 'new-book'
    content.value = [
      '我的开篇：主角被家族逐出后进入废弃书塔，发现可以用记忆兑换禁忌知识。前两章主要写压迫、规则和第一次反击准备。',
      '当前担心：设定解释偏多，第一次爽点来得慢，读者可能还没看到主角真正赢一次就流失。',
      '希望对标：同题材热榜开篇通常如何安排压迫、金手指代价、第一次低成本胜利和章尾钩子。',
    ].join('\n\n')
    benchmark.value = [
      '对标样本：热榜同题材前三章通常在 800 字内完成压迫，在第 1-2 章给主角一次可感知的小胜利，并把金手指代价做成后续悬念。',
      '读者评论观察：喜欢“开局就被逼到墙角”“能力有爽感但不是白送”“每章末尾都有未兑现问题”。',
    ].join('\n\n')
    return
  }
  goal.value = 'short-video'
  content.value = [
    '我的开篇：主角被当众羞辱，拿到反转能力，但正文节奏偏小说化。',
    '请找出最适合短视频/短剧化的 3 个强冲突场面，并说明每个场面的反转点。',
  ].join('\n\n')
  benchmark.value = '对标样本：短剧开头 15 秒先给羞辱/背叛/危机，30 秒内出现反击信号，结尾卡在更大危机。'
}

function sendBenchmarkToGenerate() {
  if (!report.value) return
  const workTitle = title.value.trim() || '未命名作品'
  const workGenre = genre.value.trim() || ''
  const benchmarkText = [
    `书名：${workTitle}`,
    `题材：${workGenre || '未填写'}`,
    `爆款潜力：${report.value.market_score}/100`,
    `结论：${report.value.verdict}`,
    `可复用套路：${report.value.tropes.join('；')}`,
    `差距：${report.value.gaps.join('；')}`,
    `下一步动作：${report.value.actions.join('；')}`,
  ].join('\n')
  saveStudioBridgePayload({
    source: 'fanqie',
    target: 'generate',
    title: workTitle,
    genre: workGenre,
    content: [content.value.trim(), benchmarkText].filter(Boolean).join('\n\n'),
    benchmark: benchmark.value.trim() || benchmarkText,
    brief: `按爆款对标动作生成可直接改稿的正文方案：${report.value.actions.join('；')}`,
  })
  void router.push('/studio/generate')
}

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
  width: min(100%, 96rem);
  max-width: calc(100vw - 2rem);
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
  font-size: 1.85rem;
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

.studio-action-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.42rem;
  border: 1px solid rgba(232, 65, 46, 0.24);
  border-radius: 999px;
  background: #241a16;
  padding: 0.68rem 1rem;
  color: #fff7ed;
  font-weight: 950;
  white-space: nowrap;
  box-shadow: 0 14px 28px rgba(54, 32, 24, 0.14);
  transition: transform 0.18s ease, box-shadow 0.18s ease;
}

.studio-action-link::after {
  content: "→";
  font-weight: 950;
}

.studio-action-link:hover {
  transform: translateY(-1px);
  box-shadow: 0 18px 34px rgba(54, 32, 24, 0.18);
}

.model-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 0.7rem;
  align-items: center;
  justify-content: space-between;
  border: 1px dashed rgba(232, 65, 46, 0.34);
  border-radius: 14px;
  padding: 0.85rem 1rem;
  margin-bottom: 1rem;
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.12), transparent 12rem),
    #fff7f3;
  color: #8c3024;
}

.model-strip span {
  border-radius: 999px;
  padding: 0.18rem 0.55rem;
  background: rgba(232, 65, 46, 0.1);
  color: #c8351f;
  font-size: 0.76rem;
  font-weight: 950;
}

.model-strip-ready {
  border-color: rgba(28, 117, 91, 0.25);
  background:
    radial-gradient(circle at 0% 0%, rgba(28, 117, 91, 0.12), transparent 12rem),
    #f2fbf6;
  color: #1c755b;
}

.model-strip-ready span {
  background: rgba(28, 117, 91, 0.1);
  color: #1c755b;
}

.model-strip p {
  flex-basis: 100%;
  margin: -0.35rem 0 0;
  color: #6b443a;
  line-height: 1.55;
}

.model-strip-ready p {
  color: #3d6e5f;
}

.model-strip-compact {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 0.34rem 0.75rem;
  align-items: center;
  padding: 0.72rem 0.95rem;
  border-radius: 16px;
}

.model-strip-compact strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-strip-compact .model-strip-copy {
  grid-column: 2;
  flex-basis: auto;
  margin: 0;
  font-size: 0.84rem;
  line-height: 1.42;
}

.model-strip-compact a {
  grid-column: 3;
  grid-row: 1 / span 2;
  align-self: center;
}

.bridge-notice {
  margin-bottom: 1rem;
  border: 1px solid rgba(232, 65, 46, 0.22);
  border-radius: 16px;
  padding: 0.8rem 1rem;
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.16), transparent 10rem),
    #fff8f3;
  color: #5a3429;
  font-weight: 850;
  box-shadow: 0 14px 36px rgba(54, 32, 24, 0.07);
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

.template-strip {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.55rem;
  margin-bottom: 0.85rem;
}

.template-strip button {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.68rem;
  background: #fffdfb;
  color: #493a35;
  text-align: left;
}

.template-strip strong,
.template-strip span {
  display: block;
}

.template-strip strong {
  font-weight: 950;
}

.template-strip span {
  margin-top: 0.22rem;
  color: #836f68;
  font-size: 0.78rem;
}

.material-map {
  border: 1px solid rgba(232, 65, 46, 0.2);
  border-radius: 12px;
  padding: 0.85rem;
  margin-bottom: 0.9rem;
  background:
    radial-gradient(circle at 100% 0%, rgba(232, 65, 46, 0.13), transparent 12rem),
    #fffaf6;
}

.map-head {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: baseline;
  margin-bottom: 0.65rem;
}

.map-head span {
  color: #c8351f;
  font-size: 0.76rem;
  font-weight: 950;
}

.map-head strong {
  color: #241a16;
  font-weight: 950;
}

.map-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.45rem;
}

.map-grid article {
  border: 1px solid #eaded8;
  border-radius: 10px;
  padding: 0.62rem;
  background: rgba(255, 255, 255, 0.72);
}

.map-grid b,
.map-grid small {
  display: block;
}

.map-grid b {
  color: #241a16;
  font-size: 0.84rem;
  font-weight: 950;
}

.map-grid small {
  margin-top: 0.22rem;
  color: #77655e;
  line-height: 1.45;
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
  min-height: 34rem;
  padding: 1rem;
}

.empty-state {
  min-height: 28rem;
  display: grid;
  place-content: center;
  gap: 0.9rem;
  color: #7a6962;
  background:
    radial-gradient(circle at 50% 0%, rgba(232, 65, 46, 0.09), transparent 16rem),
    linear-gradient(180deg, rgba(255, 251, 247, 0.9), rgba(255, 255, 255, 0.72));
}

.empty-kicker {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
  justify-self: center;
  color: #c8351f;
  font-size: 0.78rem;
  font-weight: 950;
}

.empty-state h2 {
  margin: 0;
  color: #221a18;
  font-size: 1.25rem;
  font-weight: 950;
  text-align: center;
}

.empty-state > p {
  max-width: 38rem;
  margin: 0 auto;
  text-align: center;
  line-height: 1.75;
}

.empty-mark {
  width: 3rem;
  height: 3rem;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: #241a16;
  color: #fff;
  font-weight: 950;
}

.empty-action {
  width: max-content;
  margin: 0 auto;
  border: 1px solid #e8412e;
  border-radius: 999px;
  padding: 0.62rem 1rem;
  background: #e8412e;
  color: #fff;
  font-weight: 950;
  box-shadow: 0 14px 32px rgba(232, 65, 46, 0.22);
}

.empty-guide {
  max-width: 42rem;
  border: 1px solid rgba(232, 65, 46, 0.22);
  border-radius: 12px;
  padding: 0.85rem 1rem;
  background: rgba(255, 247, 243, 0.92);
  color: #5d4640;
}

.empty-guide strong {
  display: block;
  margin-bottom: 0.45rem;
  color: #241a16;
  font-weight: 950;
}

.empty-guide ol {
  margin: 0;
  padding-left: 1.1rem;
}

.empty-guide li {
  padding: 0.16rem 0;
  line-height: 1.6;
}

.empty-preview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.65rem;
  max-width: 42rem;
}

.empty-preview article {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.75rem;
  background: #fffdfb;
  text-align: left;
}

.empty-preview span {
  color: #c8351f;
  font-size: 0.74rem;
  font-weight: 950;
}

.empty-preview strong,
.empty-preview small {
  display: block;
}

.empty-preview strong {
  margin-top: 0.2rem;
  color: #221a18;
  font-weight: 950;
}

.empty-preview small {
  margin-top: 0.25rem;
  color: #7a6962;
  line-height: 1.55;
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

.result-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.75rem;
}

.result-actions button {
  border: 0;
  border-radius: 8px;
  padding: 0.62rem 0.92rem;
  background: #e8412e;
  color: #fff;
  font-weight: 900;
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
.dark .empty-preview article,
.dark .three-col article,
.dark .sample-strip article {
  border-color: #312720;
  background: #171311;
}

.dark .empty-preview strong {
  color: #f7ede4;
}

.dark .empty-preview small {
  color: #cdbdb5;
}

.dark input,
.dark textarea,
.dark .template-strip button,
.dark .goal-row button {
  border-color: #3a2d26;
  background: #211916;
  color: #f7ede4;
}

.dark .material-map {
  border-color: rgba(232, 65, 46, 0.28);
  background:
    radial-gradient(circle at 100% 0%, rgba(232, 65, 46, 0.14), transparent 12rem),
    #171311;
}

.dark .map-head strong,
.dark .map-grid b {
  color: #f7ede4;
}

.dark .map-grid article {
  border-color: #312720;
  background: #211916;
}

.dark .map-grid small {
  color: #cdbdb5;
}

.dark .studio-action-link {
  border-color: rgba(232, 65, 46, 0.36);
  background: #e8412e;
  color: #fff7ed;
}

.dark .bridge-notice {
  border-color: rgba(232, 65, 46, 0.28);
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.18), transparent 10rem),
    #171311;
  color: #f0c7bd;
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
  .model-strip-compact {
    grid-template-columns: 1fr;
  }

  .model-strip-compact .model-strip-copy,
  .model-strip-compact a {
    grid-column: 1;
    grid-row: auto;
  }

  .field-grid,
  .template-strip,
  .goal-row,
  .map-grid,
  .empty-preview {
    grid-template-columns: 1fr;
  }

  .radar-row {
    grid-template-columns: 1fr;
  }
}
</style>
