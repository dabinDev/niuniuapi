<template>
  <AppLayout>
    <div class="teardown-page studio-wide-shell mx-auto max-w-none">
      <header class="teardown-head">
        <div>
          <span class="eyebrow">Diagnosis Board</span>
          <h1>拆书诊断</h1>
          <p>把章节拆成可执行的质检报告：黄金三章、节奏热区、人物钩子、伏笔追踪和下一步改法分开呈现。</p>
        </div>
        <router-link v-if="isStudioVisible('hotspot')" class="head-link studio-action-link" to="/studio/hotspot">去爆款对标</router-link>
      </header>

      <div class="model-strip model-strip-compact" :class="{ ready: configured }" data-test="model-auto-config-strip">
        <span>{{ configured ? '文案模型已就绪' : '自动配置流程' }}</span>
        <strong>{{ configured ? textModel : '创建第一把密钥后，系统会优先自动选择第一把密钥的最新文案模型' }}</strong>
        <p class="model-strip-copy">{{ configured ? '如需覆盖默认值，可在 API 密钥页手动测试后保存。' : '自动选择最新文案模型；手动测试后保存会记住你的选择，不手动修改时创作台继续使用自动配置。' }}</p>
        <router-link to="/keys">{{ configured ? '模型设置' : '检查密钥' }}</router-link>
      </div>

      <div v-if="bridgeNotice" class="bridge-notice" data-test="teardown-bridge-notice">
        {{ bridgeNotice }}
      </div>

      <section class="diagnosis-map diagnosis-map-wide" aria-label="拆书诊断模块">
        <article v-for="item in diagnosisModules" :key="item.title">
          <span>{{ item.kicker }}</span>
          <strong>{{ item.title }}</strong>
          <p class="model-strip-copy">{{ item.desc }}</p>
        </article>
      </section>

      <section class="teardown-grid teardown-grid-wide">
        <form class="input-panel" @submit.prevent="submit">
          <div class="template-strip" aria-label="拆书模板">
            <button
              v-for="tpl in teardownTemplates"
              :key="tpl.title"
              :data-test="`teardown-template-${tpl.key}`"
              type="button"
              @click="applyTemplate(tpl.key)"
            >
              <strong>{{ tpl.title }}</strong>
              <span>{{ tpl.desc }}</span>
            </button>
          </div>

          <div class="quality-gate" data-test="teardown-quality-gate" aria-label="提交前质量闸门">
            <div>
              <span>提交前看这 4 个点</span>
              <strong>别只贴正文，先确认它能被诊断</strong>
            </div>
            <ul>
              <li>开篇钩子：前 300 字有没有问题、压迫或反常识？</li>
              <li>首次爽点：主角是否已经赢下一次可感知的小胜利？</li>
              <li>章尾钩子：读者有没有“下一章必须看”的未兑现问题？</li>
              <li>代价/伏笔：金手指、反转或设定有没有留下代价和回收点？</li>
            </ul>
          </div>

          <label class="stacked" for="td-content">
            <span>小说正文 / 章节</span>
            <textarea
              id="td-content"
              v-model="content"
              rows="10"
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

            <div class="report-actions" data-test="teardown-report-actions">
              <button v-if="isStudioVisible('hotspot')" data-test="teardown-send-hotspot" type="button" @click="sendReportTo('hotspot')">送去爆款对标</button>
              <button v-if="isStudioVisible('generate')" data-test="teardown-send-generate" type="button" @click="sendReportTo('generate')">送去创作生成</button>
            </div>

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
            <div class="empty-kicker">
              <div class="empty-mark">QC</div>
              <span>先看钩子，再看兑现</span>
            </div>
            <h2>把一章拆成能执行的改稿清单</h2>
            <p>提交章节后，这里会生成评分、烂点、改法，并自动进入“我的作品”。建议先从黄金三章开始，确认读者为什么继续看。</p>
            <button data-test="teardown-empty-sample" type="button" class="empty-action" @click="applyTemplate('golden')">
              先填入黄金三章示例
            </button>
            <div class="empty-guide" aria-label="拆书使用路径">
              <strong>使用路径</strong>
              <ol>
                <li>粘贴 1-3 章正文，先抓开篇钩子和第一次爽点。</li>
                <li>查看烂点、伏笔风险和节奏掉线位置。</li>
                <li>把改法可直接复制到对标和生成，继续产出新版正文。</li>
              </ol>
            </div>
            <div class="empty-preview" aria-label="拆书结果预览">
              <article>
                <span>01</span>
                <strong>黄金三章评分</strong>
                <small>开篇钩子、压迫感、第一次爽点</small>
              </article>
              <article>
                <span>02</span>
                <strong>问题清单</strong>
                <small>拖沓、信息堆叠、伏笔未回收</small>
              </article>
              <article>
                <span>03</span>
                <strong>可执行改法</strong>
                <small>直接带到爆款对标或创作生成</small>
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
import { analyzeTeardown, getModelConfig, type TeardownReport, type TeardownTone } from '@/api/studio'
import { useAppStore, useAuthStore } from '@/stores'
import { consumeStudioBridgePayload, saveStudioBridgePayload, type StudioBridgeTarget } from '@/utils/studioBridge'
import { isStudioFeatureVisible, type StudioFeatureId } from '@/utils/studioFeatures'

const MIN_LEN = 100

const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

const content = ref('')
const title = ref('')
const genre = ref('')
const tone = ref<TeardownTone>('savage')
const loading = ref(false)
const report = ref<TeardownReport | null>(null)
const errorMsg = ref('')
const configured = ref(false)
const textModel = ref('')
const bridgeNotice = ref('')

const diagnosisModules = [
  { kicker: '01', title: '黄金三章', desc: '专看开篇钩子、压迫感和首次爽点兑现。' },
  { kicker: '02', title: '节奏热区', desc: '标出拖沓、信息堆叠和回报偏慢的位置。' },
  { kicker: '03', title: '伏笔追踪', desc: '把未回收伏笔和设定风险拉成清单。' },
  { kicker: '04', title: '改法队列', desc: '把建议变成可喂给创作生成的动作。' },
]

const teardownTemplates = [
  { key: 'golden', title: '黄金三章诊断', desc: '开篇钩子、压迫、首次爽点' },
  { key: 'rhythm', title: '节奏掉线排查', desc: '找拖沓段落和信息堆叠' },
]

const toneOptions: { value: TeardownTone; label: string }[] = [
  { value: 'savage', label: '毒舌' },
  { value: 'neutral', label: '中性' },
  { value: 'gentle', label: '鼓励' },
]

const contentLength = computed(() => content.value.trim().length)
const canSubmit = computed(() => configured.value && contentLength.value >= MIN_LEN && !loading.value)
const studioVisibility = computed(() => appStore.cachedPublicSettings?.studio_feature_visibility)

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
  const payload = consumeStudioBridgePayload('teardown')
  if (!payload) return
  title.value = payload.title
  genre.value = payload.genre
  content.value = payload.content
  bridgeNotice.value = payload.source === 'works'
    ? '已从我的作品带入素材，可直接开始拆书诊断。'
    : '已从番茄热榜带入素材，可直接拆开篇钩子和爽点节奏。'
}

function applyTemplate(key: string) {
  if (key === 'golden') {
    title.value = '雨夜入塔'
    genre.value = '玄幻悬疑'
    content.value = [
      '第 1 章：雨夜里，主角被家族逐出，背着欠债和母亲遗物进入废弃书塔。门缝里传来旧神低语，要求他用一个秘密换一次翻身机会。',
      '第 2 章：主角发现书塔能兑换禁忌知识，但每次兑换都会失去一段记忆。他先用最小代价反杀追债人，却忘记了母亲留下的警告。',
      '第 3 章：家族派来的天才少主登场，公开羞辱主角。主角用书塔知识破局，赢下一次低成本胜利，同时埋下记忆缺口的伏笔。',
      '请重点诊断：开篇钩子是否足够狠、压迫感是否持续、首次爽点是否兑现、代价和伏笔是否能撑住追读。',
    ].join('\n\n')
    return
  }
  title.value = title.value || '节奏排查样本'
  genre.value = genre.value || '都市爽文'
  content.value = [
    '当前问题：第 1 章有冲突，第 2 章开始解释设定，第 3 章才出现第一次小胜利。',
    '请按段落标出拖沓、信息堆叠、情绪断点和爽点兑现过慢的位置，并给出可直接改写的动作清单。',
    '补充正文：主角被上司当众羞辱，获得系统后先查看规则，解释了大量等级、商城和任务，直到章末才准备反击。',
  ].join('\n\n')
}

function isStudioVisible(id: StudioFeatureId) {
  return isStudioFeatureVisible(id, studioVisibility.value, authStore.isAdmin)
}

function sendReportTo(target: StudioBridgeTarget) {
  if (!isStudioVisible(target)) return
  if (!report.value) return
  const workTitle = title.value.trim() || '未命名作品'
  const workGenre = genre.value.trim() || ''
  const reportText = [
    `书名：${workTitle}`,
    `题材：${workGenre || '未填写'}`,
    `综合诊断：${report.value.overall_score}/100`,
    `结论：${report.value.verdict}`,
    `摘要：${report.value.summary}`,
    `亮点：${report.value.highlights.join('；')}`,
    `烂点：${report.value.rotten_points.join('；')}`,
    `改法：${report.value.suggestions.join('；')}`,
  ].join('\n')
  saveStudioBridgePayload({
    source: 'fanqie',
    target,
    title: workTitle,
    genre: workGenre,
    content: [content.value.trim(), reportText].filter(Boolean).join('\n\n'),
    benchmark: reportText,
    brief: target === 'generate'
      ? `根据拆书建议生成改写方案：${report.value.suggestions.join('；')}`
      : '对照同题材爆款样本，找出可复制套路与差距。',
  })
  void router.push(`/studio/${target}`)
}

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
    errorMsg.value = formatTeardownErrorMessage(err)
  } finally {
    loading.value = false
  }
}

function formatTeardownErrorMessage(err: unknown) {
  const error = err as {
    code?: string
    message?: string
    response?: { status?: number; data?: { message?: string } }
    status?: number
  }
  const status = error.response?.status ?? error.status ?? 0
  const raw = error.response?.data?.message || error.message || ''
  const lower = raw.toLowerCase()
  if (status === 401 || status === 403 || lower.includes('unauthorized') || lower.includes('invalid api key')) {
    return '文案模型密钥不可用，请先检查 API 密钥、分组权限和模型配置。'
  }
  if (status === 429 || lower.includes('rate limit') || lower.includes('quota')) {
    return '模型额度或频率限制已触发，请稍后重试，或切换可用文案模型。'
  }
  if (status === 502 || status === 503 || status === 504 || error.code === 'ECONNABORTED' || lower.includes('bad gateway') || lower.includes('timeout')) {
    return '模型或网关暂时不可用，请稍后重试；也可以先保存章节内容，换一个文案模型再拆。'
  }
  if (lower.includes('json') || lower.includes('format')) {
    return '模型返回格式不完整，请缩短输入或换成黄金三章模板后重试。'
  }
  return raw || '拆书失败，请稍后重试。'
}
</script>

<style scoped>
.teardown-page {
  width: min(100%, 118rem);
  max-width: calc(100vw - 1.25rem);
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
}

.studio-wide-shell {
  width: min(100%, 118rem);
  max-width: calc(100vw - 1.25rem);
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
  font-size: 1.85rem;
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

.model-strip.ready {
  border-color: rgba(28, 117, 91, 0.25);
  background:
    radial-gradient(circle at 0% 0%, rgba(28, 117, 91, 0.12), transparent 12rem),
    #f2fbf6;
  color: #1c755b;
}

.model-strip.ready span {
  background: rgba(28, 117, 91, 0.1);
  color: #1c755b;
}

.model-strip p {
  flex-basis: 100%;
  margin: -0.35rem 0 0;
  color: #6b443a;
  line-height: 1.55;
}

.model-strip.ready p {
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

.diagnosis-map {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.diagnosis-map-wide {
  grid-template-columns: repeat(4, minmax(12rem, 1fr));
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
  padding: 0.68rem 0.8rem;
  box-shadow: none;
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

.teardown-grid-wide {
  grid-template-columns: minmax(28rem, 0.82fr) minmax(0, 1.18fr);
}

.input-panel,
.result-panel {
  padding: 1rem;
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

.quality-gate {
  display: grid;
  grid-template-columns: 0.58fr 1fr;
  gap: 0.8rem;
  border: 1px solid rgba(232, 65, 46, 0.2);
  border-radius: 12px;
  padding: 0.85rem;
  margin-bottom: 0.9rem;
  background:
    linear-gradient(135deg, rgba(232, 65, 46, 0.1), rgba(255, 255, 255, 0.78)),
    #fffaf6;
}

.quality-gate span {
  display: inline-flex;
  margin-bottom: 0.3rem;
  color: #c8351f;
  font-size: 0.76rem;
  font-weight: 950;
}

.quality-gate strong {
  display: block;
  color: #241a16;
  line-height: 1.45;
}

.quality-gate ul {
  display: grid;
  gap: 0.28rem;
  margin: 0;
  padding-left: 1.05rem;
  color: #5d4640;
  font-size: 0.84rem;
  line-height: 1.55;
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
  min-height: 34rem;
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
  border-radius: 8px;
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

.report-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.5rem;
  margin: -0.2rem 0 0.85rem;
}

.report-actions button {
  border: 1px solid #e8412e;
  border-radius: 8px;
  padding: 0.54rem 0.82rem;
  background: #e8412e;
  color: #fff;
  font-weight: 900;
}

.report-actions button:last-child {
  border-color: #241a16;
  background: #241a16;
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
.dark .empty-preview article,
.dark .score-grid article,
.dark .report-columns section {
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
.dark .tone-row button {
  border-color: #3a2d26;
  background: #211916;
  color: #f7ede4;
}

.dark .quality-gate {
  border-color: rgba(232, 65, 46, 0.28);
  background:
    linear-gradient(135deg, rgba(232, 65, 46, 0.14), rgba(23, 19, 17, 0.8)),
    #171311;
}

.dark .quality-gate strong {
  color: #f7ede4;
}

.dark .quality-gate ul {
  color: #d3c1b8;
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
  .model-strip-compact {
    grid-template-columns: 1fr;
  }

  .model-strip-compact .model-strip-copy,
  .model-strip-compact a {
    grid-column: 1;
    grid-row: auto;
  }

  .field-grid,
  .tone-row,
  .template-strip,
  .quality-gate,
  .score-grid,
  .empty-preview {
    grid-template-columns: 1fr;
  }
}
</style>
