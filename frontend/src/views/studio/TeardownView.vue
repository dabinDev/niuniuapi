<template>
  <AppLayout>
    <div class="teardown mx-auto max-w-6xl">
      <header class="teardown-head">
        <span class="teardown-badge">🔍 P0 · 创作质检台</span>
        <h1 class="teardown-title">拆书 &amp; 爆款分析</h1>
        <p class="teardown-sub">把小说丢进来，拆成结构报告，并对照爆款标准做毒舌质检——专挑烂梗、烂节奏、烂套路。</p>
      </header>

      <div
        class="mb-4 rounded-xl border px-4 py-3 text-sm"
        :class="configured
          ? 'border-gray-200 bg-white/60 text-gray-600 dark:border-dark-800 dark:bg-dark-900 dark:text-gray-300'
          : 'border-dashed border-primary-300 bg-primary-50 text-primary-700 dark:border-primary-700 dark:bg-primary-900/20 dark:text-primary-300'"
      >
        <template v-if="configured">
          📝 文案模型：<span class="font-semibold">{{ textModel }}</span>
          <router-link to="/keys" class="ml-2 font-semibold text-primary-600 hover:underline dark:text-primary-300">在「API 密钥」页修改</router-link>
        </template>
        <template v-else>
          还没配置文案模型。请先到
          <router-link to="/keys" class="font-semibold underline">API 密钥页</router-link>
          选密钥 → 获取模型 → 设为文案模型 → 测试保存。
        </template>
      </div>

      <div class="teardown-grid">
        <!-- 输入区 -->
        <section class="card form-card">
          <label class="field-label" for="td-content">小说正文 / 章节</label>
          <textarea
            id="td-content"
            v-model="content"
            class="td-textarea"
            rows="12"
            placeholder="在此粘贴要拆解的小说正文，至少 100 字……"
          ></textarea>
          <div class="char-row">
            <span :class="{ 'char-warn': contentLength > 0 && contentLength < MIN_LEN }">
              {{ contentLength }} 字{{ contentLength < MIN_LEN ? `（至少 ${MIN_LEN}）` : '' }}
            </span>
          </div>

          <div class="field-grid">
            <div>
              <label class="field-label" for="td-title">书名（选填）</label>
              <input id="td-title" v-model="title" class="td-input" type="text" placeholder="例如：北境书塔" />
            </div>
            <div>
              <label class="field-label" for="td-genre">题材（选填）</label>
              <input id="td-genre" v-model="genre" class="td-input" type="text" placeholder="例如：玄幻 / 都市 / 短剧" />
            </div>
          </div>

          <label class="field-label">点评尺度</label>
          <div class="tone-row">
            <button
              v-for="opt in toneOptions"
              :key="opt.value"
              type="button"
              class="tone-btn"
              :class="{ 'tone-btn-active': tone === opt.value }"
              @click="tone = opt.value"
            >
              {{ opt.label }}
            </button>
          </div>

          <button type="button" class="submit-btn" :disabled="!canSubmit" @click="submit">
            {{ loading ? '正在拆书…' : '开始拆书' }}
          </button>

          <p v-if="errorMsg" class="err-msg">{{ errorMsg }}</p>
          <div v-if="backendPending" class="pending-msg">
            🍅 后端拆书服务正在开发中，接口就绪后即可在此实时分析。当前页面与调用流程已可用。
          </div>
        </section>

        <!-- 结果区 -->
        <section class="card result-card">
          <div v-if="loading" class="result-empty">
            <div class="spinner" aria-hidden="true"></div>
            <p>正在拆解结构、比对爆款特征…</p>
          </div>

          <template v-else-if="report">
            <div class="score-head">
              <div class="score-num">{{ report.overall_score }}<small>/100</small></div>
              <p class="verdict">{{ report.verdict }}</p>
            </div>
            <p class="summary">{{ report.summary }}</p>

            <div v-if="report.scores.length" class="scores">
              <div v-for="s in report.scores" :key="s.label" class="score-bar">
                <span class="score-bar-label">{{ s.label }}</span>
                <span class="score-bar-track"><span class="score-bar-fill" :style="{ width: s.value + '%' }"></span></span>
                <span class="score-bar-val">{{ s.value }}</span>
              </div>
            </div>

            <div v-if="report.highlights.length" class="block">
              <h3 class="block-title">✅ 亮点</h3>
              <ul><li v-for="(h, i) in report.highlights" :key="i">{{ h }}</li></ul>
            </div>
            <div v-if="report.rotten_points.length" class="block block-rotten">
              <h3 class="block-title">🍅 烂点（毒舌质检）</h3>
              <ul><li v-for="(r, i) in report.rotten_points" :key="i">{{ r }}</li></ul>
            </div>
            <div v-if="report.suggestions.length" class="block">
              <h3 class="block-title">🛠️ 改进建议</h3>
              <ul><li v-for="(g, i) in report.suggestions" :key="i">{{ g }}</li></ul>
            </div>
          </template>

          <div v-else class="result-empty">
            <div class="result-emoji" aria-hidden="true">🍅</div>
            <p>在左侧粘贴正文，点「开始拆书」，这里会给出结构拆解与爆款诊断。</p>
          </div>
        </section>
      </div>
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
const backendPending = ref(false)
const configured = ref(false)
const textModel = ref('')

onMounted(async () => {
  try {
    const cfg = await getModelConfig()
    if (cfg.text) {
      configured.value = true
      textModel.value = cfg.text.model
    }
  } catch {
    /* 忽略：未配置 */
  }
})

const toneOptions: { value: TeardownTone; label: string }[] = [
  { value: 'savage', label: '毒舌' },
  { value: 'neutral', label: '中性' },
  { value: 'gentle', label: '鼓励' },
]

const contentLength = computed(() => content.value.trim().length)
const canSubmit = computed(() => configured.value && contentLength.value >= MIN_LEN && !loading.value)

async function submit() {
  if (!canSubmit.value) return
  loading.value = true
  errorMsg.value = ''
  backendPending.value = false
  report.value = null
  try {
    report.value = await analyzeTeardown({
      content: content.value.trim(),
      title: title.value.trim() || undefined,
      genre: genre.value.trim() || undefined,
      tone: tone.value,
    })
  } catch (err: unknown) {
    const status = (err as { response?: { status?: number } })?.response?.status
    if (status === 404 || status === undefined) {
      backendPending.value = true
    } else {
      errorMsg.value = '拆书失败，请稍后重试。'
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.teardown {
  padding: 0.5rem 0 2rem;
}

.teardown-head {
  margin-bottom: 1.25rem;
}

.teardown-badge {
  display: inline-block;
  border-radius: 999px;
  background: #ffe7e0;
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 800;
  color: #c8351f;
}

.dark .teardown-badge {
  background: rgba(232, 65, 46, 0.18);
  color: #ff9d8c;
}

.teardown-title {
  margin-top: 0.6rem;
  font-size: clamp(1.6rem, 4vw, 2.2rem);
  font-weight: 900;
  color: #241a16;
}

.dark .teardown-title {
  color: #f7ede4;
}

.teardown-sub {
  margin-top: 0.4rem;
  font-size: 14px;
  line-height: 1.7;
  color: #83685c;
}

.dark .teardown-sub {
  color: #b6a294;
}

.teardown-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: 1fr;
}

@media (min-width: 1024px) {
  .teardown-grid {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }
}

.card {
  border: 1px solid rgba(58, 28, 18, 0.12);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.7);
  padding: 1.25rem;
}

.dark .card {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
}

.field-label {
  display: block;
  margin: 0.75rem 0 0.4rem;
  font-size: 13px;
  font-weight: 700;
  color: #5c463c;
}

.dark .field-label {
  color: #cdbcae;
}

.field-label:first-child {
  margin-top: 0;
}

.td-textarea,
.td-input {
  width: 100%;
  border: 1px solid rgba(58, 28, 18, 0.18);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.85);
  padding: 0.6rem 0.75rem;
  font-size: 14px;
  color: #241a16;
  resize: vertical;
}

.dark .td-textarea,
.dark .td-input {
  background: rgba(0, 0, 0, 0.25);
  border-color: rgba(255, 255, 255, 0.14);
  color: #f7ede4;
}

.td-textarea:focus,
.td-input:focus {
  outline: none;
  border-color: rgba(232, 65, 46, 0.5);
}

.char-row {
  margin-top: 0.35rem;
  font-size: 12px;
  color: #9a8475;
  text-align: right;
}

.char-warn {
  color: #e8412e;
}

.field-grid {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: 1fr 1fr;
}

.tone-row {
  display: flex;
  gap: 0.5rem;
}

.tone-btn {
  flex: 1;
  border: 1px solid rgba(58, 28, 18, 0.18);
  border-radius: 10px;
  padding: 0.45rem 0;
  font-size: 14px;
  font-weight: 700;
  color: #6b4d42;
  transition: all 0.15s ease;
}

.dark .tone-btn {
  border-color: rgba(255, 255, 255, 0.14);
  color: #cdbcae;
}

.tone-btn-active {
  border-color: #e8412e;
  background: #e8412e;
  color: #fff6f1;
}

.submit-btn {
  margin-top: 1.1rem;
  width: 100%;
  border-radius: 10px;
  background: #e8412e;
  padding: 0.7rem;
  font-size: 15px;
  font-weight: 800;
  color: #fff6f1;
  transition: background 0.15s ease, opacity 0.15s ease;
}

.submit-btn:hover:not(:disabled) {
  background: #d3361f;
}

.submit-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.err-msg {
  margin-top: 0.75rem;
  font-size: 13px;
  color: #e8412e;
}

.pending-msg {
  margin-top: 0.85rem;
  border: 1px dashed rgba(232, 65, 46, 0.4);
  border-radius: 10px;
  padding: 0.7rem 0.85rem;
  font-size: 13px;
  line-height: 1.7;
  color: #a8631a;
  background: #fff8ef;
}

.dark .pending-msg {
  background: rgba(200, 140, 30, 0.1);
  color: #f0c073;
}

.result-empty {
  display: flex;
  min-height: 280px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  text-align: center;
  color: #9a8475;
}

.result-emoji {
  font-size: 48px;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid rgba(232, 65, 46, 0.25);
  border-top-color: #e8412e;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.score-head {
  display: flex;
  align-items: baseline;
  gap: 1rem;
}

.score-num {
  font-size: 44px;
  font-weight: 950;
  color: #e8412e;
  line-height: 1;
}

.score-num small {
  font-size: 16px;
  color: #9a8475;
}

.verdict {
  font-size: 15px;
  font-weight: 700;
  color: #241a16;
}

.dark .verdict {
  color: #f7ede4;
}

.summary {
  margin-top: 0.75rem;
  font-size: 14px;
  line-height: 1.8;
  color: #5c463c;
}

.dark .summary {
  color: #cdbcae;
}

.scores {
  margin-top: 1rem;
  display: grid;
  gap: 0.5rem;
}

.score-bar {
  display: grid;
  grid-template-columns: 56px 1fr 32px;
  align-items: center;
  gap: 0.5rem;
  font-size: 13px;
}

.score-bar-track {
  height: 8px;
  border-radius: 999px;
  background: rgba(232, 65, 46, 0.12);
  overflow: hidden;
}

.score-bar-fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: #e8412e;
}

.score-bar-val {
  text-align: right;
  font-weight: 700;
  color: #6b4d42;
}

.block {
  margin-top: 1.1rem;
}

.block-title {
  font-size: 14px;
  font-weight: 800;
  color: #241a16;
}

.dark .block-title {
  color: #f7ede4;
}

.block ul {
  margin-top: 0.4rem;
  display: grid;
  gap: 0.35rem;
  padding-left: 1.1rem;
  list-style: disc;
  font-size: 13px;
  line-height: 1.7;
  color: #5c463c;
}

.dark .block ul {
  color: #cdbcae;
}

.block-rotten ul {
  color: #b3331f;
}

.dark .block-rotten ul {
  color: #ff9d8c;
}
</style>
