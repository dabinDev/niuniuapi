<template>
  <AppLayout>
    <div class="script mx-auto max-w-6xl">
      <header class="sc-head">
        <span class="sc-badge">🎬 P0 · 创作台</span>
        <h1 class="sc-title">剧本生成</h1>
        <p class="sc-sub">小说转剧本、分镜与短剧脚本，一份内容多平台变现，吃满短剧红利。</p>
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

      <div class="sc-grid">
        <!-- 表单 -->
        <section class="card form-card">
          <label class="field-label" for="sc-content">小说正文 / 大纲</label>
          <textarea
            id="sc-content"
            v-model="content"
            class="sc-textarea"
            rows="12"
            placeholder="粘贴要改编的小说章节或大纲，至少 50 字……"
          ></textarea>
          <div class="char-row">
            <span :class="{ 'char-warn': contentLength > 0 && contentLength < MIN_LEN }">
              {{ contentLength }} 字{{ contentLength < MIN_LEN ? `（至少 ${MIN_LEN}）` : '' }}
            </span>
          </div>

          <label class="field-label">目标形态</label>
          <div class="form-row">
            <button
              v-for="opt in formOptions"
              :key="opt.value"
              type="button"
              class="form-btn"
              :class="{ 'form-btn-active': form === opt.value }"
              @click="form = opt.value"
            >
              {{ opt.label }}
            </button>
          </div>

          <button type="button" class="submit-btn" :disabled="!canSubmit" @click="submit">
            {{ loading ? '正在生成…' : '生成剧本' }}
          </button>

          <p v-if="errorMsg" class="err-msg">{{ errorMsg }}</p>
          <div v-if="backendPending" class="pending-msg">
            🍅 后端剧本生成服务正在开发中，接口就绪后即可在此实时生成。当前页面与调用流程已可用。
          </div>
        </section>

        <!-- 结果 -->
        <section class="card result-card">
          <div v-if="loading" class="result-empty">
            <div class="spinner" aria-hidden="true"></div>
            <p>正在改写为剧本 / 分镜…</p>
          </div>

          <template v-else-if="result">
            <h2 class="script-name">{{ result.title }}</h2>
            <div v-for="(scene, i) in result.scenes" :key="i" class="scene">
              <h3 class="scene-heading">{{ scene.heading }}</h3>
              <p class="scene-content">{{ scene.content }}</p>
            </div>
          </template>

          <div v-else class="result-empty">
            <div class="result-emoji" aria-hidden="true">🎬</div>
            <p>粘贴正文，选好形态，点「生成剧本」，这里会输出场景与分镜。</p>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { generateScript, getModelConfig, type ScriptForm, type ScriptResult } from '@/api/studio'

const MIN_LEN = 50

const content = ref('')
const form = ref<ScriptForm>('short')
const loading = ref(false)
const result = ref<ScriptResult | null>(null)
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

const formOptions: { value: ScriptForm; label: string }[] = [
  { value: 'long', label: '长剧本' },
  { value: 'short', label: '短剧' },
  { value: 'storyboard', label: '分镜脚本' },
]

const contentLength = computed(() => content.value.trim().length)
const canSubmit = computed(() => configured.value && contentLength.value >= MIN_LEN && !loading.value)

async function submit() {
  if (!canSubmit.value) return
  loading.value = true
  errorMsg.value = ''
  backendPending.value = false
  result.value = null
  try {
    result.value = await generateScript({
      content: content.value.trim(),
      form: form.value,
    })
  } catch (err: unknown) {
    const status = (err as { response?: { status?: number } })?.response?.status
    if (status === 404 || status === undefined) {
      backendPending.value = true
    } else {
      errorMsg.value = '生成失败，请稍后重试。'
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.script {
  padding: 0.5rem 0 2rem;
}

.sc-head {
  margin-bottom: 1.25rem;
}

.sc-badge {
  display: inline-block;
  border-radius: 999px;
  background: #ffe7e0;
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 800;
  color: #c8351f;
}

.dark .sc-badge {
  background: rgba(232, 65, 46, 0.18);
  color: #ff9d8c;
}

.sc-title {
  margin-top: 0.6rem;
  font-size: clamp(1.6rem, 4vw, 2.2rem);
  font-weight: 900;
  color: #241a16;
}

.dark .sc-title {
  color: #f7ede4;
}

.sc-sub {
  margin-top: 0.4rem;
  font-size: 14px;
  line-height: 1.7;
  color: #83685c;
}

.dark .sc-sub {
  color: #b6a294;
}

.sc-grid {
  display: grid;
  gap: 1rem;
  grid-template-columns: 1fr;
}

@media (min-width: 1024px) {
  .sc-grid {
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

.sc-textarea {
  width: 100%;
  border: 1px solid rgba(58, 28, 18, 0.18);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.85);
  padding: 0.6rem 0.75rem;
  font-size: 14px;
  color: #241a16;
  resize: vertical;
}

.dark .sc-textarea {
  background: rgba(0, 0, 0, 0.25);
  border-color: rgba(255, 255, 255, 0.14);
  color: #f7ede4;
}

.sc-textarea:focus {
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

.form-row {
  display: flex;
  gap: 0.5rem;
}

.form-btn {
  flex: 1;
  border: 1px solid rgba(58, 28, 18, 0.18);
  border-radius: 10px;
  padding: 0.45rem 0;
  font-size: 14px;
  font-weight: 700;
  color: #6b4d42;
  transition: all 0.15s ease;
}

.dark .form-btn {
  border-color: rgba(255, 255, 255, 0.14);
  color: #cdbcae;
}

.form-btn-active {
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

.script-name {
  font-size: 18px;
  font-weight: 900;
  color: #241a16;
}

.dark .script-name {
  color: #f7ede4;
}

.scene {
  margin-top: 1rem;
  border-left: 3px solid rgba(232, 65, 46, 0.5);
  padding-left: 0.85rem;
}

.scene-heading {
  font-size: 14px;
  font-weight: 800;
  color: #b3331f;
}

.dark .scene-heading {
  color: #ff9d8c;
}

.scene-content {
  margin-top: 0.3rem;
  font-size: 13px;
  line-height: 1.8;
  white-space: pre-wrap;
  color: #5c463c;
}

.dark .scene-content {
  color: #cdbcae;
}
</style>
