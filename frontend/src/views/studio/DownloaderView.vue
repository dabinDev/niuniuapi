<template>
  <AppLayout>
    <div class="fanqie-page mx-auto max-w-7xl">
      <header class="page-head">
        <div>
          <span class="eyebrow">Rank Research</span>
          <h1>番茄热榜</h1>
          <p>把榜单当作素材雷达：看热榜、巅峰榜、男生榜、女生榜前 30，搜索指定小说，并在授权范围内下载完整小说做个人备份与拆解。</p>
        </div>
        <div class="head-actions">
          <router-link to="/studio/hotspot">送去爆款对标</router-link>
          <router-link class="primary" to="/studio/generate">生成创作方向</router-link>
        </div>
      </header>

      <section class="metric-strip" aria-label="榜单概览">
        <article>
          <span>当前榜单</span>
          <strong>{{ activeChannelLabel }}</strong>
          <p>{{ rankBooks.length }} 条可对标素材</p>
        </article>
        <article>
          <span>更新时间</span>
          <strong>{{ updatedAtLabel }}</strong>
          <p>{{ rankSourceLabel }}</p>
        </article>
        <article>
          <span>搜索结果</span>
          <strong>{{ searchResults.length }}</strong>
          <p>支持书名、作者、题材关键词</p>
        </article>
        <article>
          <span>下一步</span>
          <strong>下载全本</strong>
          <p>授权后导入完整小说 TXT</p>
        </article>
      </section>

      <section class="toolbar-panel">
        <div class="channel-tabs" aria-label="番茄榜单频道">
          <button
            v-for="channel in channels"
            :key="channel.value"
            :data-test="`fanqie-channel-${channel.value}`"
            type="button"
            :class="{ active: activeChannel === channel.value }"
            @click="switchChannel(channel.value)"
          >
            <strong>{{ channel.label }}</strong>
            <span>{{ channel.hint }}</span>
          </button>
        </div>

        <form class="search-box" @submit.prevent="submitSearch">
          <label for="fanqie-search">搜索指定小说</label>
          <div>
            <input
              id="fanqie-search"
              v-model="searchQuery"
              data-test="fanqie-search-input"
              type="search"
              placeholder="输入书名或作者，例如：十日终焉"
            />
            <button data-test="fanqie-search-submit" type="submit" :disabled="searchLoading" @click.prevent="submitSearch">
              {{ searchLoading ? '搜索中...' : '搜索' }}
            </button>
          </div>
        </form>
      </section>

      <section class="rank-workbench">
        <main class="rank-panel">
          <div class="panel-head">
            <div>
              <span class="eyebrow small">{{ showingSearch ? 'Search Result' : 'Top 30' }}</span>
              <h2>{{ showingSearch ? '搜索结果' : `${activeChannelLabel}前 30` }}</h2>
            </div>
            <button type="button" class="text-action" :disabled="rankLoading" @click="loadRank">
              {{ rankLoading ? '刷新中...' : '刷新榜单' }}
            </button>
          </div>

          <div v-if="rankLoading" class="state-panel">正在获取榜单数据...</div>
          <div v-else-if="currentBooks.length === 0" class="state-panel dashed">暂无榜单数据</div>
          <div v-else class="rank-list">
            <button
              v-for="book in currentBooks"
              :key="book.id + book.title"
              data-test="fanqie-rank-row"
              type="button"
              class="rank-row"
              :class="{ active: selectedBook?.id === book.id }"
              @click="selectBook(book)"
            >
              <span class="rank-num">{{ String(book.rank).padStart(2, '0') }}</span>
              <span class="book-cover-thumb">
                <img
                  v-if="book.cover_url"
                  data-test="fanqie-cover"
                  :src="book.cover_url"
                  :alt="`${book.title}封面`"
                  loading="lazy"
                  referrerpolicy="no-referrer"
                />
                <span v-else class="book-cover-empty">{{ book.title.slice(0, 1) }}</span>
              </span>
              <span class="book-main">
                <strong>{{ book.title }}</strong>
                <small>{{ book.author }} · {{ book.category }} · {{ book.status }}</small>
              </span>
              <span class="book-score">{{ book.score }}</span>
            </button>
          </div>
        </main>

        <aside class="sample-panel">
          <div class="panel-head">
            <div>
              <span class="eyebrow small">Benchmark Sample</span>
              <h2>样本卡</h2>
            </div>
          </div>

          <div v-if="!selectedBook" class="state-panel">选择一本榜单小说，右侧会生成可复制的对标样本卡。</div>
          <template v-else>
            <div class="sample-card" :class="{ 'has-cover': selectedBook.cover_url }" data-test="fanqie-sample-card">
              <div class="sample-cover-frame">
                <img
                  v-if="selectedBook.cover_url"
                  data-test="fanqie-selected-cover"
                  class="sample-cover"
                  :src="selectedBook.cover_url"
                  :alt="`${selectedBook.title}封面`"
                  loading="lazy"
                  referrerpolicy="no-referrer"
                />
                <span v-else class="sample-cover-empty">{{ selectedBook.title.slice(0, 1) }}</span>
              </div>
              <div class="sample-copy">
                <div class="sample-meta-line">
                  <span>#{{ selectedBook.rank }}</span>
                  <span>{{ selectedBook.category }}</span>
                  <span>{{ selectedBook.score }}</span>
                </div>
                <h3>{{ selectedBook.title }}</h3>
                <dl>
                  <div>
                    <dt>作者</dt>
                    <dd>{{ selectedBook.author }}</dd>
                  </div>
                  <div>
                    <dt>字数</dt>
                    <dd>{{ selectedBook.word_count }}</dd>
                  </div>
                  <div>
                    <dt>状态</dt>
                    <dd>{{ selectedBook.status }}</dd>
                  </div>
                </dl>
                <p>{{ selectedBook.description }}</p>
              </div>
            </div>

            <div class="tag-list">
              <span v-for="tag in selectedBook.tags" :key="tag">{{ tag }}</span>
            </div>

            <textarea class="copy-area" readonly :value="benchmarkText"></textarea>

            <div class="sample-actions">
              <button type="button" @click="copyBenchmark">复制样本卡</button>
              <button
                data-test="fanqie-analyze"
                type="button"
                class="secondary"
                :disabled="analyzeLoading"
                @click="analyzeSelectedBook"
              >
                {{ analyzeLoading ? '分析中...' : '分析前10章' }}
              </button>
              <a :href="selectedBook.source_url" target="_blank" rel="noopener">打开番茄搜索</a>
            </div>
            <label class="consent-row">
              <input data-test="fanqie-consent" v-model="consent" type="checkbox" />
              <span>我确认该作品为本人作品或已获得授权，仅用于个人备份/学习分析。</span>
            </label>
            <div class="sample-actions bottom-actions">
              <button
                data-test="fanqie-download"
                type="button"
                :disabled="downloadLoading || !consent"
                @click="downloadSelectedBook"
              >
                {{ downloadLoading ? '导入中...' : '下载完整小说 / 导入 TXT' }}
              </button>
            </div>
            <p v-if="copyMsg" class="copy-msg">{{ copyMsg }}</p>
            <p v-if="actionMsg" class="copy-msg">{{ actionMsg }}</p>

            <section v-if="downloadResult" class="action-result">
              <span>导入包</span>
              <strong>{{ downloadResult.file_name }}</strong>
              <p>{{ downloadResult.chapter_count }} 章 · {{ formatDownloadStatus(downloadResult.decode_status) }}</p>
              <ul>
                <li v-for="note in downloadResult.notes" :key="note">{{ note }}</li>
              </ul>
              <div class="result-actions">
                <button data-test="fanqie-save-txt" type="button" @click="saveDownloadText">保存 TXT</button>
              </div>
            </section>

            <section v-if="analysisResult" class="analysis-result">
              <span>开篇分析</span>
              <h3>{{ analysisResult.title }}</h3>
              <p>{{ analysisResult.summary }}</p>
              <div v-if="analysisResult.hooks?.length" class="mini-list">
                <strong>钩子</strong>
                <b v-for="hook in analysisResult.hooks" :key="hook">{{ hook }}</b>
              </div>
              <div v-if="analysisResult.actions?.length" class="mini-list">
                <strong>动作</strong>
                <b v-for="action in analysisResult.actions" :key="action">{{ action }}</b>
              </div>
            </section>
          </template>
        </aside>
      </section>

      <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  analyzeFanqieBook,
  downloadFanqieBook,
  getFanqieRank,
  searchFanqieBooks,
  type FanqieAnalysisReport,
  type FanqieBook,
  type FanqieDownloadResult,
  type FanqieRankChannel,
} from '@/api/studio'

const channels: { value: FanqieRankChannel; label: string; hint: string }[] = [
  { value: 'hot', label: '热榜', hint: '综合热度样本' },
  { value: 'peak', label: '巅峰榜', hint: '长期高讨论作品' },
  { value: 'male', label: '男生榜', hint: '男频题材观察' },
  { value: 'female', label: '女生榜', hint: '女频题材观察' },
]

const activeChannel = ref<FanqieRankChannel>('hot')
const rankBooks = ref<FanqieBook[]>([])
const searchResults = ref<FanqieBook[]>([])
const selectedBook = ref<FanqieBook | null>(null)
const searchQuery = ref('')
const updatedAt = ref('')
const rankSource = ref('')
const rankLoading = ref(false)
const searchLoading = ref(false)
const downloadLoading = ref(false)
const analyzeLoading = ref(false)
const errorMsg = ref('')
const copyMsg = ref('')
const actionMsg = ref('')
const showingSearch = ref(false)
const consent = ref(false)
const downloadResult = ref<FanqieDownloadResult | null>(null)
const analysisResult = ref<FanqieAnalysisReport | null>(null)

const activeChannelLabel = computed(() => channels.find((item) => item.value === activeChannel.value)?.label || '热榜')
const currentBooks = computed(() => showingSearch.value ? searchResults.value : rankBooks.value)
const updatedAtLabel = computed(() => {
  if (!updatedAt.value) return '待刷新'
  const d = new Date(updatedAt.value)
  return Number.isNaN(d.getTime()) ? updatedAt.value : d.toLocaleString()
})
const rankSourceLabel = computed(() => rankSource.value === 'fanqie-rank-cache' ? '后端榜单缓存' : (rankSource.value || '后端聚合接口'))
const benchmarkText = computed(() => {
  if (!selectedBook.value) return ''
  const book = selectedBook.value
  return [
    `书名：${book.title}`,
    `作者：${book.author}`,
    `题材：${book.category}`,
    `状态/字数：${book.status} / ${book.word_count}`,
    `榜单：${activeChannelLabel.value} #${book.rank}`,
    `卖点：${book.description}`,
    `标签：${book.tags.join('、')}`,
  ].join('\n')
})

function resetBookActions() {
  copyMsg.value = ''
  actionMsg.value = ''
  downloadResult.value = null
  analysisResult.value = null
}

function extractApiErrorMessage(err: unknown) {
  const error = err as { message?: string; response?: { data?: { message?: string } } }
  return error?.response?.data?.message || error?.message || ''
}

function formatDownloadStatus(status: string) {
  switch (status) {
    case 'readable':
      return '已采集正文'
    case 'font_encoded':
      return '已采集正文（网页字体编码）'
    case 'browser_required':
      return '需要浏览器校验'
    case 'official_blocked':
      return '官方验证码拦截'
    case 'web_preview':
      return '网页预览段落'
    case 'catalog_only':
      return '仅目录'
    default:
      return status || '未知状态'
  }
}

function selectBook(book: FanqieBook) {
  selectedBook.value = book
  resetBookActions()
}

async function loadRank() {
  rankLoading.value = true
  errorMsg.value = ''
  showingSearch.value = false
  try {
    const result = await getFanqieRank(activeChannel.value)
    rankBooks.value = result.books || []
    updatedAt.value = result.updated_at
    rankSource.value = result.source || ''
    selectedBook.value = rankBooks.value[0] || null
    resetBookActions()
  } catch (err: unknown) {
    rankBooks.value = []
    selectedBook.value = null
    const msg = extractApiErrorMessage(err)
    errorMsg.value = msg || '番茄榜单获取失败，请稍后重试。'
  } finally {
    rankLoading.value = false
  }
}

async function switchChannel(channel: FanqieRankChannel) {
  if (activeChannel.value === channel && !showingSearch.value) return
  activeChannel.value = channel
  await loadRank()
}

async function submitSearch() {
  const q = searchQuery.value.trim()
  if (!q) return
  searchLoading.value = true
  errorMsg.value = ''
  try {
    searchResults.value = await searchFanqieBooks(q)
    showingSearch.value = true
    selectedBook.value = searchResults.value[0] || null
    resetBookActions()
  } catch (err: unknown) {
    searchResults.value = []
    selectedBook.value = null
    const msg = extractApiErrorMessage(err)
    errorMsg.value = msg || '搜索失败，请稍后重试。'
  } finally {
    searchLoading.value = false
  }
}

async function copyBenchmark() {
  if (!benchmarkText.value) return
  copyMsg.value = ''
  try {
    await navigator.clipboard?.writeText(benchmarkText.value)
    copyMsg.value = '已复制，可粘贴到爆款对标。'
  } catch {
    copyMsg.value = '当前浏览器不允许自动复制，请手动选中样本卡内容。'
  }
}

async function downloadSelectedBook() {
  if (!selectedBook.value) return
  downloadLoading.value = true
  actionMsg.value = ''
  downloadResult.value = null
  try {
    downloadResult.value = await downloadFanqieBook(selectedBook.value, consent.value)
    actionMsg.value = `已生成 ${downloadResult.value.file_name}`
  } catch (err: unknown) {
    const msg = extractApiErrorMessage(err)
    actionMsg.value = msg || '下载/导入失败，请稍后重试。'
  } finally {
    downloadLoading.value = false
  }
}

function saveDownloadText() {
  if (!downloadResult.value) return
  const blob = new Blob([downloadResult.value.text || ''], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = downloadResult.value.file_name || `${downloadResult.value.title || 'fanqie-book'}.txt`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
  actionMsg.value = `已开始下载 ${link.download}`
}

async function analyzeSelectedBook() {
  if (!selectedBook.value) return
  analyzeLoading.value = true
  actionMsg.value = ''
  analysisResult.value = null
  try {
    analysisResult.value = await analyzeFanqieBook(selectedBook.value)
  } catch (err: unknown) {
    const msg = extractApiErrorMessage(err)
    actionMsg.value = msg || '分析失败，请先确认文案模型已配置。'
  } finally {
    analyzeLoading.value = false
  }
}

onMounted(loadRank)
</script>

<style scoped>
@font-face {
  font-family: "FanqieRankOfficial";
  src:
    url("https://lf6-awef.bytetos.com/obj/awesome-font/c/dc027189e0ba4cd.woff2") format("woff2"),
    url("https://lf3-awef.bytetos.com/obj/awesome-font/c/dc027189e0ba4cd.woff2") format("woff2");
  font-display: swap;
}

.fanqie-page {
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
}

.page-head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 1rem;
  align-items: end;
  margin-bottom: 1rem;
}

.eyebrow {
  display: inline-flex;
  border: 1px solid rgba(232, 65, 46, 0.28);
  border-radius: 999px;
  padding: 0.22rem 0.64rem;
  color: #c8351f;
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0;
}

.eyebrow.small {
  font-size: 0.68rem;
}

.page-head h1 {
  margin: 0.5rem 0 0.35rem;
  font-size: 1.85rem;
  font-weight: 950;
  letter-spacing: 0;
}

.page-head p {
  max-width: 54rem;
  color: #665854;
  line-height: 1.75;
}

.head-actions,
.sample-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.head-actions a,
.sample-actions a,
.sample-actions button,
.text-action,
.search-box button {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.62rem 0.85rem;
  background: #fffdfb;
  color: #493a35;
  font-weight: 850;
}

.head-actions a.primary,
.search-box button,
.sample-actions button {
  border-color: #e8412e;
  background: #e8412e;
  color: #fff;
}

.sample-actions .secondary {
  border-color: #241a16;
  background: #241a16;
  color: #fff;
}

.metric-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.metric-strip article,
.toolbar-panel,
.rank-panel,
.sample-panel {
  border: 1px solid #eaded8;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 18px 50px rgba(54, 32, 24, 0.08);
}

.metric-strip article {
  padding: 0.85rem;
  box-shadow: none;
}

.metric-strip span {
  color: #9f7a6d;
  font-size: 0.78rem;
  font-weight: 850;
}

.metric-strip strong {
  display: block;
  margin-top: 0.25rem;
  font-weight: 950;
}

.metric-strip p {
  margin-top: 0.2rem;
  color: #665854;
  line-height: 1.55;
}

.toolbar-panel {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(320px, 0.45fr);
  gap: 1rem;
  padding: 0.9rem;
  margin-bottom: 1rem;
}

.channel-tabs {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.55rem;
}

.channel-tabs button {
  min-height: 4rem;
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.65rem;
  background: #fffdfb;
  color: #493a35;
  text-align: left;
}

.channel-tabs button.active {
  border-color: #241a16;
  background: #241a16;
  color: #fff;
}

.channel-tabs strong,
.channel-tabs span {
  display: block;
}

.channel-tabs span {
  margin-top: 0.22rem;
  color: #836f68;
  font-size: 0.76rem;
}

.channel-tabs button.active span {
  color: #f0c7bd;
}

.search-box label {
  display: block;
  margin-bottom: 0.35rem;
  color: #6e5a54;
  font-size: 0.8rem;
  font-weight: 850;
}

.search-box div {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.5rem;
}

.search-box input,
.copy-area {
  width: 100%;
  border: 1px solid #e6d6ce;
  border-radius: 8px;
  padding: 0.7rem 0.78rem;
  background: #fffdfb;
  color: #231917;
  outline: none;
}

.search-box input:focus,
.copy-area:focus {
  border-color: #e8412e;
  box-shadow: 0 0 0 3px rgba(232, 65, 46, 0.12);
}

.rank-workbench {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(320px, 0.65fr);
  gap: 1rem;
}

.rank-panel,
.sample-panel {
  padding: 1rem;
  align-self: start;
  position: sticky;
  top: 1rem;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: end;
  margin-bottom: 0.8rem;
}

.panel-head h2 {
  margin-top: 0.25rem;
  font-size: 1.15rem;
  font-weight: 950;
}

.rank-list {
  display: grid;
  gap: 0.45rem;
  max-height: 44rem;
  overflow: auto;
  padding-right: 0.25rem;
  font-family: "FanqieRankOfficial", "Microsoft YaHei", sans-serif;
}

.rank-row {
  width: 100%;
  display: grid;
  grid-template-columns: 2.6rem 3.1rem minmax(0, 1fr) auto;
  gap: 0.7rem;
  align-items: center;
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.7rem;
  background: #fffdfb;
  color: #493a35;
  text-align: left;
}

.rank-row.active {
  border-color: #e8412e;
  box-shadow: 0 0 0 3px rgba(232, 65, 46, 0.1);
}

.rank-num {
  color: #c8351f;
  font-weight: 950;
}

.book-cover-thumb {
  width: 3.1rem;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  border: 1px solid #eaded8;
  border-radius: 6px;
  background: #f8eee9;
}

.book-cover-thumb img,
.sample-cover {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
}

.book-cover-empty {
  width: 100%;
  height: 100%;
  display: grid;
  place-items: center;
  color: #c8351f;
  font-weight: 950;
}

.book-main {
  min-width: 0;
}

.book-main strong,
.book-main small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.book-main small {
  margin-top: 0.18rem;
  color: #8b7a74;
  font-size: 0.78rem;
}

.book-score {
  color: #1c755b;
  font-size: 0.8rem;
  font-weight: 850;
  white-space: nowrap;
}

.state-panel {
  min-height: 14rem;
  display: grid;
  place-items: center;
  border: 1px solid #f0e5df;
  border-radius: 8px;
  color: #8b7a74;
  text-align: center;
}

.state-panel.dashed {
  border-style: dashed;
}

.sample-card {
  border-radius: 8px;
  padding: 1rem;
  background: #241a16;
  color: #fff;
  font-family: "FanqieRankOfficial", "Microsoft YaHei", sans-serif;
}

.sample-card.has-cover {
  display: grid;
  grid-template-columns: 5rem minmax(0, 1fr);
  gap: 0.85rem;
  align-items: start;
}

.sample-cover {
  aspect-ratio: 3 / 4;
  border-radius: 7px;
  background: #3b2923;
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.24);
}

.sample-copy {
  min-width: 0;
}

.sample-card span {
  color: #f0c7bd;
  font-size: 0.78rem;
  font-weight: 900;
}

.sample-card h3 {
  margin-top: 0.35rem;
  font-size: 1.35rem;
  font-weight: 950;
  letter-spacing: 0;
}

.sample-card p {
  margin-top: 0.55rem;
  color: #f7ede4;
  line-height: 1.75;
}

.sample-card dl {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  margin-top: 0.85rem;
}

.sample-card dt {
  color: #f0c7bd;
  font-size: 0.75rem;
}

.sample-card dd {
  margin-top: 0.16rem;
  font-weight: 850;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin: 0.85rem 0;
}

.tag-list span {
  border-radius: 999px;
  padding: 0.24rem 0.56rem;
  background: #fff7f3;
  color: #c8351f;
  font-size: 0.78rem;
  font-weight: 850;
}

.copy-area {
  min-height: 11rem;
  resize: vertical;
  line-height: 1.7;
  font-family: "FanqieRankOfficial", "Microsoft YaHei", sans-serif;
}

.sample-actions {
  margin-top: 0.75rem;
}

.bottom-actions {
  justify-content: flex-end;
}

.consent-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 0.5rem;
  align-items: start;
  margin-top: 0.75rem;
  border: 1px solid #f0e5df;
  border-radius: 8px;
  padding: 0.65rem;
  background: #fffdfb;
  color: #6e5a54;
  font-size: 0.82rem;
  line-height: 1.55;
}

.consent-row input {
  margin-top: 0.22rem;
  accent-color: #e8412e;
}

.sample-actions button:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.copy-msg,
.error-text {
  margin-top: 0.7rem;
  color: #1c755b;
  font-weight: 800;
}

.error-text {
  color: #c8351f;
}

.action-result,
.analysis-result {
  margin-top: 0.8rem;
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.8rem;
  background: #fffdfb;
  font-family: "FanqieRankOfficial", "Microsoft YaHei", sans-serif;
}

.action-result span,
.analysis-result span {
  color: #c8351f;
  font-size: 0.76rem;
  font-weight: 950;
}

.action-result strong,
.analysis-result h3 {
  display: block;
  margin-top: 0.22rem;
  color: #221a18;
  font-weight: 950;
}

.action-result p,
.analysis-result p {
  margin-top: 0.35rem;
  color: #665854;
  line-height: 1.65;
}

.action-result ul {
  margin-top: 0.5rem;
  padding-left: 1rem;
  color: #665854;
  line-height: 1.6;
}

.result-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.65rem;
}

.result-actions button {
  border: 1px solid #241a16;
  border-radius: 8px;
  padding: 0.55rem 0.8rem;
  background: #241a16;
  color: #fff;
  font-weight: 850;
}

.mini-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  align-items: center;
  margin-top: 0.55rem;
}

.mini-list strong {
  color: #9f7a6d;
  font-size: 0.78rem;
}

.mini-list b {
  border-radius: 999px;
  padding: 0.22rem 0.52rem;
  background: #fff7f3;
  color: #c8351f;
  font-size: 0.78rem;
}

.dark .fanqie-page,
.dark .page-head h1,
.dark .panel-head h2 {
  color: #f7ede4;
}

.dark .page-head p,
.dark .metric-strip p,
.dark .state-panel,
.dark .search-box label {
  color: #cdbdb5;
}

.dark .metric-strip article,
.dark .toolbar-panel,
.dark .rank-panel,
.dark .sample-panel,
.dark .rank-row,
.dark .consent-row,
.dark .action-result,
.dark .analysis-result {
  border-color: #312720;
  background: #171311;
}

.dark .action-result strong,
.dark .analysis-result h3 {
  color: #f7ede4;
}

.dark .search-box input,
.dark .copy-area,
.dark .channel-tabs button {
  border-color: #3a2d26;
  background: #211916;
  color: #f7ede4;
}

@media (max-width: 1080px) {
  .page-head,
  .metric-strip,
  .toolbar-panel,
  .rank-workbench {
    grid-template-columns: 1fr;
  }

  .page-head {
    align-items: start;
  }
}

@media (max-width: 760px) {
  .channel-tabs,
  .sample-card dl,
  .rank-row {
    grid-template-columns: 1fr;
  }

  .book-score {
    white-space: normal;
  }
}
</style>
