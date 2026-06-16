<template>
  <AppLayout>
    <div class="creative-page mx-auto max-w-7xl">
      <header class="creative-head">
        <div>
          <span class="eyebrow">Writing Bench</span>
          <h1>创作生成</h1>
          <p>围绕同一份素材生成大纲、正文续写、改写增强和改编脚本。每次结果都会进入作品归档，方便回看和继续加工。</p>
        </div>
        <router-link class="head-link" to="/studio/works">查看我的作品</router-link>
      </header>

      <div class="model-strip" :class="{ ready: configured }">
        <span>{{ configured ? '文案模型' : '还没配置文案模型' }}</span>
        <strong>{{ configured ? textModel : '请先到 API 密钥页设置文案模型' }}</strong>
        <router-link to="/keys">模型设置</router-link>
      </div>

      <section class="creative-grid">
        <form class="writer-panel" @submit.prevent="submit">
          <div class="mode-grid" aria-label="创作模式">
            <button
              v-for="opt in modeOptions"
              :key="opt.value"
              :data-test="`creative-mode-${opt.value}`"
              type="button"
              :class="{ active: mode === opt.value }"
              @click="mode = opt.value"
            >
              <b>{{ opt.label }}</b>
              <span>{{ opt.hint }}</span>
            </button>
          </div>

          <div class="field-grid">
            <label>
              <span>题材</span>
              <input v-model="genre" type="text" placeholder="玄幻 / 都市 / 女频 / 悬疑" />
            </label>
            <label>
              <span>风格</span>
              <input v-model="style" type="text" placeholder="毒舌爽文 / 细腻情绪 / 短剧快节奏" />
            </label>
          </div>

          <label class="stacked">
            <span>创作目标</span>
            <input v-model="brief" type="text" placeholder="例如：把第 2 章改成一次低成本胜利，结尾留钩子" />
          </label>

          <label class="stacked">
            <span>小说正文 / 大纲 / 拆书结论</span>
            <textarea
              id="sc-content"
              v-model="content"
              rows="13"
              placeholder="粘贴要继续创作或改写的内容，至少 50 字。"
            ></textarea>
          </label>

          <div class="number-grid">
            <label>
              <span>目标字数</span>
              <input v-model.number="targetWords" type="number" min="0" step="100" placeholder="可选" />
            </label>
            <label>
              <span>集/段数</span>
              <input v-model.number="episodes" type="number" min="0" step="1" placeholder="可选" />
            </label>
          </div>

          <div class="submit-row">
            <span :class="{ warn: contentLength > 0 && contentLength < MIN_LEN }">
              {{ contentLength }} 字{{ contentLength < MIN_LEN ? `（至少 ${MIN_LEN}）` : '' }}
            </span>
            <button class="submit-btn" type="button" :disabled="!canSubmit" @click="submit">
              {{ loading ? '正在生成...' : '生成内容' }}
            </button>
          </div>

          <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>
        </form>

        <section class="output-panel">
          <div v-if="loading" class="empty-output">
            <div class="typing-bars"><i></i><i></i><i></i></div>
            <p>正在检查设定一致性，并生成可继续写的内容。</p>
          </div>

          <template v-else-if="result">
            <div class="result-head">
              <span>{{ modeLabel(result.mode) }}</span>
              <h2>{{ result.title }}</h2>
              <p v-if="result.summary">{{ result.summary }}</p>
            </div>

            <article v-for="(section, i) in result.sections" :key="i" class="scene">
              <h3>{{ section.heading }}</h3>
              <p>{{ section.content }}</p>
            </article>

            <div class="post-grid">
              <section v-if="result.checklist && result.checklist.length">
                <h3>一致性检查</h3>
                <ul><li v-for="item in result.checklist" :key="item">{{ item }}</li></ul>
              </section>
              <section v-if="result.next_steps && result.next_steps.length">
                <h3>下一步</h3>
                <ol><li v-for="item in result.next_steps" :key="item">{{ item }}</li></ol>
              </section>
            </div>
          </template>

          <div v-else class="empty-output">
            <div class="draft-mark">GEN</div>
            <p>选择生成类型，粘贴素材，这里会输出可编辑、可归档、可继续加工的内容。</p>
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
  generateCreative,
  getModelConfig,
  type CreativeMode,
  type CreativeResult,
} from '@/api/studio'

const MIN_LEN = 50

const content = ref('')
const brief = ref('')
const genre = ref('')
const style = ref('')
const targetWords = ref<number | undefined>()
const episodes = ref<number | undefined>()
const mode = ref<CreativeMode>('short')
const loading = ref(false)
const result = ref<CreativeResult | null>(null)
const errorMsg = ref('')
const configured = ref(false)
const textModel = ref('')

const modeOptions: { value: CreativeMode; label: string; hint: string }[] = [
  { value: 'outline', label: '大纲', hint: '卷纲、主线、章节钩子' },
  { value: 'draft', label: '正文续写', hint: '承接原文继续写' },
  { value: 'rewrite', label: '改写', hint: '增强冲突和爽点' },
  { value: 'short', label: '短剧', hint: '竖屏强冲突版本' },
  { value: 'storyboard', label: '分镜', hint: '镜头、旁白、时长' },
  { value: 'long', label: '长剧本', hint: '场景、动作、对白' },
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

function modeLabel(value: string) {
  return modeOptions.find((opt) => opt.value === value)?.label || value
}

async function submit() {
  if (!canSubmit.value) return
  loading.value = true
  errorMsg.value = ''
  result.value = null
  try {
    result.value = await generateCreative({
      content: content.value.trim(),
      mode: mode.value,
      brief: brief.value.trim() || undefined,
      genre: genre.value.trim() || undefined,
      style: style.value.trim() || undefined,
      target_words: targetWords.value || undefined,
      episodes: episodes.value || undefined,
    })
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
    errorMsg.value = msg || '生成失败，请稍后重试。'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.creative-page {
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
}

.creative-head {
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

.creative-head h1 {
  margin: 0.55rem 0 0.35rem;
  font-size: 2.05rem;
  font-weight: 950;
  letter-spacing: 0;
}

.creative-head p {
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

.creative-grid {
  display: grid;
  grid-template-columns: minmax(340px, 0.9fr) minmax(0, 1.1fr);
  gap: 1rem;
}

.writer-panel,
.output-panel {
  border: 1px solid #eaded8;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 18px 50px rgba(54, 32, 24, 0.08);
}

.writer-panel {
  padding: 1rem;
}

.mode-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.55rem;
  margin-bottom: 0.9rem;
}

.mode-grid button {
  min-height: 4.3rem;
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.62rem;
  background: #fffdfb;
  color: #493a35;
  text-align: left;
}

.mode-grid b,
.mode-grid span {
  display: block;
}

.mode-grid b {
  margin-bottom: 0.22rem;
  font-weight: 950;
}

.mode-grid span {
  color: #836f68;
  font-size: 0.78rem;
  line-height: 1.45;
}

.mode-grid button.active {
  border-color: #241a16;
  background: #241a16;
  color: #fff;
}

.mode-grid button.active span {
  color: #f0c7bd;
}

.field-grid,
.number-grid {
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
  line-height: 1.75;
}

input:focus,
textarea:focus {
  border-color: #e8412e;
  box-shadow: 0 0 0 3px rgba(232, 65, 46, 0.12);
}

.stacked,
.number-grid {
  margin-top: 0.8rem;
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

.submit-btn {
  border: 0;
  border-radius: 8px;
  padding: 0.75rem 1.2rem;
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

.output-panel {
  min-height: 38rem;
  padding: 1rem;
}

.empty-output {
  min-height: 32rem;
  display: grid;
  place-content: center;
  gap: 1rem;
  text-align: center;
  color: #7a6962;
}

.draft-mark {
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

.typing-bars {
  display: flex;
  gap: 0.35rem;
  justify-content: center;
}

.typing-bars i {
  width: 0.5rem;
  height: 2.4rem;
  border-radius: 999px;
  background: #e8412e;
  animation: writePulse 0.9s ease-in-out infinite;
}

.typing-bars i:nth-child(2) {
  animation-delay: 0.12s;
  background: #1c755b;
}

.typing-bars i:nth-child(3) {
  animation-delay: 0.24s;
  background: #2f5d9f;
}

@keyframes writePulse {
  0%, 100% { transform: scaleY(0.45); opacity: 0.55; }
  50% { transform: scaleY(1); opacity: 1; }
}

.result-head {
  border-radius: 8px;
  padding: 1rem;
  background: #241a16;
  color: #fff;
}

.result-head span {
  color: #f0c7bd;
  font-size: 0.78rem;
  font-weight: 900;
}

.result-head h2 {
  margin-top: 0.3rem;
  font-size: 1.5rem;
  font-weight: 950;
}

.result-head p {
  margin-top: 0.45rem;
  color: #f7ede4;
  line-height: 1.7;
}

.scene {
  margin-top: 0.75rem;
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.9rem;
  background: #fffdfb;
}

.scene h3 {
  margin-bottom: 0.45rem;
  color: #c8351f;
  font-size: 0.95rem;
  font-weight: 950;
}

.scene p {
  white-space: pre-wrap;
  color: #51423d;
  line-height: 1.8;
}

.post-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin-top: 0.85rem;
}

.post-grid section {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.85rem;
  background: #f8fbff;
}

.post-grid section:last-child {
  background: #f2fbf6;
}

.post-grid h3 {
  margin-bottom: 0.5rem;
  font-size: 0.95rem;
  font-weight: 950;
}

.post-grid ul,
.post-grid ol {
  padding-left: 1.05rem;
  color: #594843;
  line-height: 1.75;
}

.dark .creative-page,
.dark .creative-head h1 {
  color: #f7ede4;
}

.dark .creative-head p,
.dark .empty-output,
.dark label span {
  color: #cdbdb5;
}

.dark .writer-panel,
.dark .output-panel,
.dark .scene,
.dark .post-grid section {
  border-color: #312720;
  background: #171311;
}

.dark input,
.dark textarea,
.dark .mode-grid button {
  border-color: #3a2d26;
  background: #211916;
  color: #f7ede4;
}

@media (max-width: 980px) {
  .creative-head,
  .creative-grid,
  .post-grid {
    grid-template-columns: 1fr;
  }

  .creative-head {
    align-items: flex-start;
  }
}

@media (max-width: 640px) {
  .mode-grid,
  .field-grid,
  .number-grid {
    grid-template-columns: 1fr;
  }
}
</style>
