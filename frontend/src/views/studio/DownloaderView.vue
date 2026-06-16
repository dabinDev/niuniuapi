<template>
  <AppLayout>
    <div class="importer-page mx-auto max-w-7xl">
      <header class="importer-head">
        <div>
          <span class="eyebrow">Import Desk</span>
          <h1>番茄导入器</h1>
          <p>用于个人作品备份与授权内容整理。先把章节、安全声明和导入记录做好，再把内容送进拆书、爆款对标和创作生成。</p>
        </div>
        <router-link class="head-link" to="/studio/works">查看导入记录</router-link>
      </header>

      <section class="compliance-band">
        <div>
          <strong>合规边界</strong>
          <p>仅导入你拥有权利或已获授权的内容。链接清单会先记录为任务，不鼓励批量采集、搬运或绕过平台限制。</p>
        </div>
        <label class="consent">
          <input v-model="consent" data-test="import-consent" type="checkbox" />
          <span>我确认仅导入有权备份和处理的内容</span>
        </label>
      </section>

      <section class="importer-grid">
        <form class="import-panel" @submit.prevent="submit">
          <div class="source-switch" aria-label="导入方式">
            <button type="button" :class="{ active: source === 'manual' }" @click="source = 'manual'">粘贴章节</button>
            <button type="button" :class="{ active: source === 'fanqie' }" @click="source = 'fanqie'">链接清单</button>
          </div>

          <label>
            <span>作品名</span>
            <input v-model="title" data-test="import-title" type="text" placeholder="例如：北境书塔" />
          </label>

          <label v-if="source === 'manual'" class="stacked">
            <span>章节内容</span>
            <textarea
              v-model="content"
              data-test="import-content"
              rows="15"
              placeholder="支持按“第 1 章 标题”自动拆分。也可以先粘贴全文，后续再整理。"
            ></textarea>
          </label>

          <label v-else class="stacked">
            <span>链接清单</span>
            <textarea
              v-model="urlText"
              data-test="import-urls"
              rows="15"
              placeholder="每行一个你有权处理的章节或作品链接。当前会先保存任务清单，后续接异步抓取器。"
            ></textarea>
          </label>

          <div class="submit-row">
            <span>{{ source === 'manual' ? `${contentLength} 字` : `${urlList.length} 条链接` }}</span>
            <button data-test="import-submit" type="button" :disabled="!canSubmit" @click="submit">
              {{ loading ? '正在导入...' : '保存导入' }}
            </button>
          </div>

          <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>
        </form>

        <section class="result-panel">
          <div v-if="loading" class="empty-result">
            <div class="scan-box"></div>
            <p>正在整理章节、记录来源和生成下一步动作。</p>
          </div>

          <template v-else-if="result">
            <div class="result-head">
              <span>{{ result.status === 'queued' ? '已入队' : '已完成' }}</span>
              <h2>{{ result.title }}</h2>
              <p>{{ result.source === 'manual' ? '手动章节导入' : '链接清单任务' }}</p>
            </div>

            <div class="chapter-table">
              <div class="chapter-row table-head">
                <span>章节</span>
                <span>字数</span>
                <span>来源</span>
              </div>
              <div v-for="chapter in result.chapters" :key="chapter.title + chapter.source" class="chapter-row">
                <strong>{{ chapter.title }}</strong>
                <span>{{ chapter.word_count || '-' }}</span>
                <small>{{ chapter.source || '粘贴内容' }}</small>
              </div>
            </div>

            <div class="next-grid">
              <article>
                <h3>导入说明</h3>
                <ul><li v-for="note in result.notes" :key="note">{{ note }}</li></ul>
              </article>
              <article>
                <h3>下一步</h3>
                <ol><li v-for="action in result.next_actions" :key="action">{{ action }}</li></ol>
              </article>
            </div>
          </template>

          <div v-else class="empty-result">
            <div class="import-mark">IN</div>
            <p>保存后的导入记录会出现在“我的作品”，可以继续送去拆书、爆款对标或创作生成。</p>
          </div>
        </section>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  importStudioContent,
  type StudioImportResult,
  type StudioImportSource,
} from '@/api/studio'

const source = ref<StudioImportSource>('manual')
const title = ref('')
const content = ref('')
const urlText = ref('')
const consent = ref(false)
const loading = ref(false)
const errorMsg = ref('')
const result = ref<StudioImportResult | null>(null)

const contentLength = computed(() => content.value.trim().length)
const urlList = computed(() => urlText.value
  .split(/\r?\n/)
  .map((line) => line.trim())
  .filter(Boolean))
const hasPayload = computed(() => source.value === 'manual' ? contentLength.value > 0 : urlList.value.length > 0)
const canSubmit = computed(() => consent.value && hasPayload.value && !loading.value)

async function submit() {
  if (!canSubmit.value) return
  loading.value = true
  errorMsg.value = ''
  result.value = null
  try {
    result.value = await importStudioContent({
      source: source.value,
      title: title.value.trim() || undefined,
      content: source.value === 'manual' ? content.value.trim() : undefined,
      urls: source.value === 'fanqie' ? urlList.value : undefined,
      consent: consent.value,
    })
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
    errorMsg.value = msg || '导入失败，请检查内容后重试。'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.importer-page {
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
}

.importer-head {
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

.importer-head h1 {
  margin: 0.55rem 0 0.35rem;
  font-size: 2.05rem;
  font-weight: 950;
  letter-spacing: 0;
}

.importer-head p {
  max-width: 50rem;
  color: #665854;
  line-height: 1.75;
}

.head-link {
  color: #c8351f;
  font-weight: 800;
}

.compliance-band {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 1rem;
  align-items: center;
  border: 1px solid rgba(47, 93, 159, 0.22);
  border-radius: 8px;
  padding: 0.9rem;
  margin-bottom: 1rem;
  background: #f8fbff;
}

.compliance-band strong {
  display: block;
  margin-bottom: 0.25rem;
  color: #2f5d9f;
  font-weight: 950;
}

.compliance-band p {
  color: #5d6c82;
  line-height: 1.65;
}

.consent {
  display: flex;
  gap: 0.45rem;
  align-items: center;
  color: #2f5d9f;
  font-weight: 850;
  white-space: nowrap;
}

.importer-grid {
  display: grid;
  grid-template-columns: minmax(340px, 0.9fr) minmax(0, 1.1fr);
  gap: 1rem;
}

.import-panel,
.result-panel {
  border: 1px solid #eaded8;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 18px 50px rgba(54, 32, 24, 0.08);
}

.import-panel {
  padding: 1rem;
}

.source-switch {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.55rem;
  margin-bottom: 0.9rem;
}

.source-switch button {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.68rem;
  background: #fffdfb;
  color: #493a35;
  font-weight: 900;
}

.source-switch button.active {
  border-color: #241a16;
  background: #241a16;
  color: #fff;
}

label span {
  display: block;
  margin-bottom: 0.35rem;
  color: #6e5a54;
  font-size: 0.8rem;
  font-weight: 800;
}

input[type='text'],
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

.stacked {
  display: block;
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

.submit-row button {
  border: 0;
  border-radius: 8px;
  padding: 0.75rem 1.2rem;
  background: #e8412e;
  color: #fff;
  font-weight: 950;
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

.empty-result {
  min-height: 30rem;
  display: grid;
  place-content: center;
  gap: 1rem;
  text-align: center;
  color: #7a6962;
}

.import-mark {
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

.scan-box {
  width: 5rem;
  height: 5rem;
  margin: 0 auto;
  border: 2px solid #e8412e;
  border-radius: 8px;
  background:
    linear-gradient(transparent 45%, rgba(232, 65, 46, 0.18) 46%, transparent 50%),
    #fff7f3;
  animation: scanBox 1.1s ease-in-out infinite;
}

@keyframes scanBox {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(0.4rem); }
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
  font-size: 1.45rem;
  font-weight: 950;
}

.result-head p {
  margin-top: 0.3rem;
  color: #f7ede4;
}

.chapter-table {
  margin-top: 0.9rem;
  overflow: hidden;
  border: 1px solid #eaded8;
  border-radius: 8px;
}

.chapter-row {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) 5rem minmax(0, 1fr);
  gap: 0.7rem;
  align-items: center;
  padding: 0.7rem 0.8rem;
  border-top: 1px solid #f0e5df;
  color: #594843;
}

.chapter-row:first-child {
  border-top: 0;
}

.table-head {
  background: #fff7f3;
  color: #c8351f;
  font-size: 0.8rem;
  font-weight: 950;
}

.chapter-row strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chapter-row small {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #8b7a74;
}

.next-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin-top: 0.85rem;
}

.next-grid article {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.85rem;
  background: #fffdfb;
}

.next-grid article:last-child {
  background: #f2fbf6;
}

.next-grid h3 {
  margin-bottom: 0.5rem;
  font-size: 0.95rem;
  font-weight: 950;
}

.next-grid ul,
.next-grid ol {
  padding-left: 1.05rem;
  color: #594843;
  line-height: 1.75;
}

.dark .importer-page,
.dark .importer-head h1 {
  color: #f7ede4;
}

.dark .importer-head p,
.dark .empty-result,
.dark label span {
  color: #cdbdb5;
}

.dark .import-panel,
.dark .result-panel,
.dark .next-grid article {
  border-color: #312720;
  background: #171311;
}

.dark input[type='text'],
.dark textarea,
.dark .source-switch button {
  border-color: #3a2d26;
  background: #211916;
  color: #f7ede4;
}

@media (max-width: 980px) {
  .importer-head,
  .compliance-band,
  .importer-grid,
  .next-grid {
    grid-template-columns: 1fr;
  }

  .importer-head {
    align-items: flex-start;
  }

  .consent {
    white-space: normal;
  }
}

@media (max-width: 640px) {
  .chapter-row {
    grid-template-columns: 1fr;
  }
}
</style>
