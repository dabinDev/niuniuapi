<template>
  <AppLayout>
    <div class="fanqie-page studio-wide-shell mx-auto max-w-none">
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

      <section class="metric-strip metric-strip-wide" aria-label="榜单概览">
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

      <section class="rank-workbench rank-workbench-wide">
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
          <div v-else-if="currentBooks.length === 0 && errorMsg" class="empty-action-card" data-test="fanqie-rank-empty-action">
            <div class="empty-orbit" aria-hidden="true">
              <span class="empty-icon">!</span>
            </div>
            <div>
              <span class="empty-kicker">离线也能继续</span>
              <strong>榜单暂时没拉到</strong>
              <p>{{ errorMsg }}</p>
              <div class="empty-steps">
                <b><span>01</span>刷新榜单</b>
                <b><span>02</span>搜索指定小说</b>
                <b><span>03</span>粘贴作品页链接导入完整 TXT</b>
              </div>
            </div>
          </div>
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
                <span v-else class="book-cover-empty" data-test="fanqie-cover-placeholder">
                  <b>{{ book.title.slice(0, 1) }}</b>
                  <small>榜样</small>
                </span>
              </span>
              <span class="book-main">
                <strong>{{ book.title }}</strong>
                <small>{{ book.author }} · {{ book.category }} · {{ book.status }}</small>
              </span>
              <span class="book-score">{{ book.score }}</span>
            </button>
          </div>
        </main>

        <aside class="sample-panel sample-panel-scroll" data-test="fanqie-sample-panel">
          <div class="panel-head">
            <div>
              <span class="eyebrow small">Benchmark Sample</span>
              <h2>样本卡</h2>
            </div>
          </div>

          <div v-if="!selectedBook" class="sample-empty-guide" data-test="fanqie-sample-empty-guide">
            <div class="guide-cover-stack" aria-hidden="true">
              <span></span>
              <span></span>
              <strong>样</strong>
            </div>
            <div>
              <span class="empty-kicker">Benchmark starts here</span>
              <strong>先选一本样本，或从搜索结果导入</strong>
              <p>右侧会生成带封面、题材、字数、简介和授权备份入口的样本卡；再一键送去拆书、爆款对标或创作生成。</p>
              <div class="guide-flow">
                <b>热榜样本</b>
                <i></i>
                <b>对标卡</b>
                <i></i>
                <b>下游加工</b>
              </div>
            </div>
          </div>
          <template v-else>
            <nav class="sample-fast-lane" data-test="fanqie-sample-fast-lane" aria-label="样本快捷操作">
              <button data-test="fanqie-fast-analyze" type="button" :disabled="analyzeLoading" @click="analyzeSelectedBook">
                {{ analyzeLoading ? '分析中...' : '分析前10章' }}
              </button>
              <button data-test="fanqie-fast-hotspot" type="button" @click="sendSelectedBookTo('hotspot')">送去对标</button>
              <a href="#fanqie-backup">授权下载</a>
            </nav>

            <div class="sample-card sample-card-compact" :class="{ 'has-cover': selectedBook.cover_url }" data-test="fanqie-sample-card">
              <div class="sample-poster">
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
                  <span v-else class="sample-cover-empty" data-test="fanqie-selected-cover-placeholder">
                    <small>NO COVER</small>
                    <b>{{ coverPlaceholderText(selectedBook.title) }}</b>
                    <em>Rank {{ selectedBook.rank }}</em>
                  </span>
                </div>
                <div class="poster-caption">
                  <span>{{ activeChannelLabel }} #{{ selectedBook.rank }}</span>
                  <strong>{{ selectedBook.score || '热度样本' }}</strong>
                </div>
              </div>
              <div class="sample-copy">
                <div class="sample-meta-line">
                  <span>{{ selectedBook.category }}</span>
                  <span>{{ selectedBook.status }}</span>
                  <span>{{ selectedBook.word_count }}</span>
                </div>
                <h3>{{ selectedBook.title }}</h3>
                <p class="sample-author">作者：{{ selectedBook.author }}</p>
                <div class="decision-strip" data-test="fanqie-decision-strip" aria-label="样本阅读决策">
                  <article>
                    <span>先看</span>
                    <strong>题材钩子</strong>
                  </article>
                  <article>
                    <span>再拆</span>
                    <strong>前十章</strong>
                  </article>
                  <article>
                    <span>最后</span>
                    <strong>授权导入</strong>
                  </article>
                </div>
                <dl class="sample-profile">
                  <div class="sample-profile-topic" data-test="fanqie-sample-topic">
                    <dt>题材</dt>
                    <dd>{{ selectedBook.category }}</dd>
                  </div>
                  <div data-test="fanqie-sample-word-count">
                    <dt>字数</dt>
                    <dd>{{ selectedBook.word_count }}</dd>
                  </div>
                  <div data-test="fanqie-sample-status">
                    <dt>状态</dt>
                    <dd>{{ selectedBook.status }}</dd>
                  </div>
                </dl>
                <section class="sample-desc">
                  <span>简介</span>
                  <p>{{ selectedBook.description || '暂无简介，可先复制样本卡或打开番茄搜索查看详情。' }}</p>
                </section>
              </div>
            </div>

            <div class="tag-list">
              <span v-for="tag in selectedBook.tags" :key="tag">{{ tag }}</span>
            </div>

            <textarea class="copy-area" readonly :value="benchmarkText"></textarea>

            <div class="sample-ops sample-ops-board sample-ops-compact" data-test="fanqie-sample-ops">
              <section data-test="fanqie-sample-action-rail">
                <span>前十章轻拆</span>
                <p>先看卖点、标签和前十章钩子，不需要为了分析先拉完整本。</p>
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
              </section>

              <section data-test="fanqie-sample-downstream-card">
                <span>下游加工</span>
                <p>把热榜样本带到拆书、对标和生成页，自动预填标题、题材和素材。</p>
                <div class="sample-actions">
                  <button data-test="fanqie-send-teardown" type="button" class="secondary" @click="sendSelectedBookTo('teardown')">
                    送去拆书
                  </button>
                  <button data-test="fanqie-send-hotspot" type="button" class="secondary" @click="sendSelectedBookTo('hotspot')">
                    送去对标
                  </button>
                  <button data-test="fanqie-send-generate" type="button" class="secondary" @click="sendSelectedBookTo('generate')">
                    生成方向
                  </button>
                </div>
              </section>

              <section id="fanqie-backup" class="backup-section backup-section-stack" data-test="fanqie-backup-card">
                <span>授权全本备份</span>
                <p>确认授权后才下载/导入完整小说 TXT；未授权时仍可先用样本卡做拆书和对标。</p>
                <label class="consent-row">
                  <input data-test="fanqie-consent" v-model="consent" type="checkbox" />
                  <span>我确认该作品为本人作品或已获得授权，仅用于个人备份/学习分析。</span>
                </label>
                <div v-if="downloadLoading" class="download-progress-card" data-test="fanqie-download-progress">
                  <strong>完整小说采集中</strong>
                  <p>全本下载会逐章抓取正文，热门长篇可能需要数分钟。请保持页面打开，完成后会自动生成 TXT 导入包。</p>
                </div>
                <div class="sample-actions bottom-actions">
                  <button
                    data-test="fanqie-download"
                    type="button"
                    :disabled="downloadLoading || !consent"
                    @click="downloadSelectedBook"
                  >
                    {{ downloadLoading ? '导入中...' : '下载/导入完整小说 TXT' }}
                  </button>
                </div>
              </section>
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

      <p v-if="errorMsg && currentBooks.length > 0" class="error-text">{{ errorMsg }}</p>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
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
import { buildFanqieBridgePayload, saveStudioBridgePayload, type StudioBridgeTarget } from '@/utils/studioBridge'

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
const router = useRouter()

const activeChannelLabel = computed(() => channels.find((item) => item.value === activeChannel.value)?.label || '热榜')
const currentBooks = computed(() => showingSearch.value ? searchResults.value : rankBooks.value)
const updatedAtLabel = computed(() => {
  if (!updatedAt.value) return '待刷新'
  const d = new Date(updatedAt.value)
  return Number.isNaN(d.getTime()) ? updatedAt.value : d.toLocaleString()
})
const rankSourceLabel = computed(() => {
  if (rankSource.value === 'fanqie-official-cache') return '后端官方缓存'
  if (rankSource.value === 'fanqie-rank-cache') return '后端榜单缓存'
  return rankSource.value || '后端聚合接口'
})
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
  const error = err as { message?: string; response?: { data?: { message?: string } }; status?: number }
  return error?.response?.data?.message || error?.message || ''
}

function formatDownloadErrorMessage(err: unknown) {
  const error = err as { code?: string; message?: string }
  const msg = extractApiErrorMessage(err)
  if (error?.code === 'ECONNABORTED' || /timeout/i.test(error?.message || msg)) {
    return '完整小说下载耗时较长，本次请求已超时。请稍后重试，或先使用“分析前10章”完成拆书；长篇全本建议保持页面打开等待导入包生成。'
  }
  return msg || '下载/导入失败，请稍后重试。'
}

function formatRankErrorMessage(err: unknown) {
  const msg = extractApiErrorMessage(err)
  const error = err as { message?: string; response?: { status?: number; data?: { message?: string } } }
  const isServerFailure = (error?.response?.status || 0) >= 500
  const isNetworkFailure = /network|failed|timeout/i.test(error?.message || '')
  if (!msg || isServerFailure || isNetworkFailure) {
    return '番茄榜单接口暂时不可用。本地后端未启动、接口异常或番茄源站临时拦截时会出现这个提示；请稍后刷新，或先使用搜索/作品页链接导入。'
  }
  return msg
}

function sendSelectedBookTo(target: StudioBridgeTarget) {
  if (!selectedBook.value) return
  saveStudioBridgePayload(buildFanqieBridgePayload(selectedBook.value, benchmarkText.value, target))
  const label = target === 'teardown' ? '拆书诊断' : target === 'hotspot' ? '爆款对标' : '创作生成'
  actionMsg.value = `已把《${selectedBook.value.title}》样本送到${label}，打开对应页面即可继续。`
  void router.push(`/studio/${target === 'generate' ? 'generate' : target}`)
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

function coverPlaceholderText(title: string) {
  const firstWord = title.trim().split(/\s+/)[0]
  return firstWord || title.slice(0, 2) || '样本'
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
    errorMsg.value = formatRankErrorMessage(err)
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
    actionMsg.value = formatDownloadErrorMessage(err)
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
.fanqie-page {
  width: min(100%, 118rem);
  max-width: calc(100vw - 1.25rem);
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
  font-family: "Noto Sans SC", "Microsoft YaHei", "PingFang SC", sans-serif;
}

.studio-wide-shell {
  width: min(100%, 118rem);
  max-width: calc(100vw - 1.25rem);
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

.metric-strip-wide {
  grid-template-columns: repeat(4, minmax(12rem, 1fr));
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
  grid-template-columns: minmax(0, 1.28fr) minmax(360px, 0.72fr);
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
  grid-template-columns: minmax(520px, 1.12fr) minmax(430px, 0.88fr);
  gap: 1rem;
}

.rank-workbench-wide {
  grid-template-columns: minmax(0, 1.34fr) minmax(26rem, 0.86fr);
}

.rank-panel,
.sample-panel {
  padding: 1rem;
  align-self: start;
}

.sample-panel {
  position: sticky;
  top: 1rem;
}

.sample-panel-scroll {
  max-height: calc(100vh - 2rem);
  overflow: auto;
  scrollbar-gutter: stable;
  overscroll-behavior: contain;
}

.sample-panel-scroll::-webkit-scrollbar {
  width: 0.45rem;
}

.sample-panel-scroll::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(232, 65, 46, 0.28);
}

.sample-fast-lane {
  position: sticky;
  top: -1rem;
  z-index: 2;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.45rem;
  margin: -0.15rem 0 0.75rem;
  border: 1px solid #eaded8;
  border-radius: 14px;
  padding: 0.48rem;
  background:
    linear-gradient(180deg, rgba(255, 253, 251, 0.96), rgba(255, 247, 243, 0.92));
  backdrop-filter: blur(10px);
  box-shadow: 0 12px 30px rgba(54, 32, 24, 0.08);
}

.sample-fast-lane button,
.sample-fast-lane a {
  display: grid;
  min-height: 2.35rem;
  place-items: center;
  border: 1px solid #eaded8;
  border-radius: 10px;
  padding: 0.45rem 0.4rem;
  background: #fffdfb;
  color: #493a35;
  font-size: 0.78rem;
  font-weight: 950;
  text-align: center;
}

.sample-fast-lane button:first-child {
  border-color: #241a16;
  background: #241a16;
  color: #fff7ed;
}

.sample-fast-lane a {
  border-color: rgba(28, 117, 91, 0.28);
  background: #f2fbf6;
  color: #1c755b;
}

.sample-fast-lane button:disabled {
  cursor: not-allowed;
  opacity: 0.52;
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
  font-family: "Noto Sans SC", "Microsoft YaHei", "PingFang SC", sans-serif;
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
  align-content: center;
  gap: 0.05rem;
  color: #c8351f;
  font-weight: 950;
}

.book-cover-empty b,
.book-cover-empty small {
  display: block;
  line-height: 1;
}

.book-cover-empty b {
  font-size: 0.98rem;
}

.book-cover-empty small {
  color: #8b7a74;
  font-size: 0.48rem;
  letter-spacing: 0.08em;
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

.empty-action-card {
  min-height: 14rem;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 1rem;
  align-items: start;
  border: 1px solid rgba(232, 65, 46, 0.18);
  border-radius: 12px;
  padding: 1rem;
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.16), transparent 12rem),
    linear-gradient(135deg, #fffaf6, #fff);
  color: #493a35;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
}

.empty-orbit {
  display: grid;
  width: 3rem;
  height: 3rem;
  place-items: center;
  border-radius: 999px;
  background: #241a16;
  box-shadow: 0 12px 28px rgba(36, 26, 22, 0.16);
}

.empty-icon {
  display: grid;
  width: 1.85rem;
  height: 1.85rem;
  place-items: center;
  border-radius: 999px;
  background: #e8412e;
  color: #fff7ed;
  font-weight: 950;
}

.empty-kicker {
  display: inline-flex;
  margin-bottom: 0.35rem;
  border: 1px solid rgba(232, 65, 46, 0.2);
  border-radius: 999px;
  padding: 0.18rem 0.55rem;
  color: #c8351f;
  font-size: 0.68rem;
  font-weight: 950;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.empty-action-card strong,
.sample-empty-guide strong {
  display: block;
  color: #201714;
  font-size: 1.05rem;
  font-weight: 950;
}

.empty-action-card p,
.sample-empty-guide p {
  margin-top: 0.45rem;
  color: #6c5a52;
  line-height: 1.72;
}

.empty-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.5rem;
  margin-top: 0.85rem;
}

.empty-steps b {
  display: flex;
  min-height: 3rem;
  align-items: center;
  gap: 0.45rem;
  border: 1px solid #f0e5df;
  border-radius: 10px;
  padding: 0.55rem;
  background: rgba(255, 255, 255, 0.74);
  color: #3b2923;
  font-size: 0.8rem;
  font-weight: 950;
}

.empty-steps span {
  display: grid;
  width: 1.35rem;
  height: 1.35rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 999px;
  background: #241a16;
  color: #fff7ed;
  font-size: 0.62rem;
}

.sample-empty-guide {
  min-height: 18rem;
  display: grid;
  place-items: center;
  border: 1px solid #f0e5df;
  border-radius: 12px;
  padding: 1rem;
  background:
    linear-gradient(135deg, rgba(255, 247, 243, 0.72), rgba(255, 255, 255, 0.92)),
    repeating-linear-gradient(135deg, rgba(232, 65, 46, 0.05) 0 1px, transparent 1px 18px);
  text-align: left;
}

.sample-empty-guide > div:last-child {
  max-width: 23rem;
}

.guide-cover-stack {
  position: relative;
  width: 5.2rem;
  height: 6.8rem;
  margin-bottom: 1rem;
}

.guide-cover-stack span,
.guide-cover-stack strong {
  position: absolute;
  inset: 0;
  border-radius: 12px;
}

.guide-cover-stack span:first-child {
  transform: translate(-0.65rem, 0.5rem) rotate(-8deg);
  background: #f5d8cc;
}

.guide-cover-stack span:nth-child(2) {
  transform: translate(0.62rem, 0.35rem) rotate(7deg);
  background: #efb49f;
}

.guide-cover-stack strong {
  display: grid;
  place-items: center;
  background:
    radial-gradient(circle at 35% 25%, rgba(255, 226, 181, 0.9), transparent 1.3rem),
    linear-gradient(145deg, #241a16, #e8412e);
  color: #fff7ed;
  font-size: 2.5rem;
  font-weight: 950;
  box-shadow: 0 18px 40px rgba(54, 32, 24, 0.22);
}

.guide-flow {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem;
  margin-top: 0.9rem;
}

.guide-flow b {
  border-radius: 999px;
  padding: 0.28rem 0.6rem;
  background: #fff;
  color: #3b2923;
  font-size: 0.75rem;
  font-weight: 950;
}

.guide-flow i {
  width: 1rem;
  height: 1px;
  background: #d9c5bb;
}

.sample-card {
  display: grid;
  grid-template-columns: minmax(5.6rem, 0.28fr) minmax(0, 1fr);
  gap: 0.85rem;
  align-items: start;
  border-radius: 8px;
  padding: 1rem;
  background:
    radial-gradient(circle at 100% 0%, rgba(232, 65, 46, 0.32), transparent 12rem),
    radial-gradient(circle at 0% 100%, rgba(255, 200, 165, 0.16), transparent 12rem),
    linear-gradient(145deg, #211613, #3a2118);
  color: #fff;
  font-family: "Noto Sans SC", "Microsoft YaHei", "PingFang SC", sans-serif;
}

.sample-card.has-cover {
  grid-template-columns: minmax(5.6rem, 0.28fr) minmax(0, 1fr);
}

.sample-card-compact,
.sample-card-compact.has-cover {
  grid-template-columns: minmax(6.2rem, 0.3fr) minmax(0, 1fr);
  gap: 0.9rem;
  padding: 1rem;
  border-radius: 12px;
}

.sample-poster {
  min-width: 0;
}

.sample-cover-frame {
  aspect-ratio: 3 / 4;
  max-height: 15.5rem;
  overflow: hidden;
  border-radius: 7px;
  background: #3b2923;
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.24);
}

.sample-card-compact .sample-cover-frame {
  max-height: 12.8rem;
  border-radius: 9px;
}

.sample-cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.sample-cover-empty {
  display: grid;
  width: 100%;
  height: 100%;
  place-items: center;
  align-content: center;
  gap: 0.35rem;
  color: #ffc8a5;
  font-weight: 950;
}

.sample-cover-empty b,
.sample-cover-empty small,
.sample-cover-empty em {
  display: block;
  line-height: 1;
}

.sample-cover-empty b {
  color: #ffe3cf;
  font-size: clamp(2.4rem, 6vw, 3.4rem);
}

.sample-cover-empty small,
.sample-cover-empty em {
  color: rgba(255, 200, 165, 0.76);
  font-size: 0.7rem;
  font-style: normal;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.poster-caption {
  margin-top: 0.5rem;
  border: 1px solid rgba(255, 200, 165, 0.2);
  border-radius: 10px;
  padding: 0.46rem 0.56rem;
  background: rgba(255, 255, 255, 0.08);
}

.poster-caption span,
.poster-caption strong {
  display: block;
}

.poster-caption span {
  color: #f0c7bd;
  font-size: 0.76rem;
  font-weight: 850;
}

.poster-caption strong {
  margin-top: 0.12rem;
  color: #fff8f1;
  font-size: 0.95rem;
  font-weight: 950;
}

.sample-copy {
  min-width: 0;
}

.sample-meta-line {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.sample-meta-line span {
  border: 1px solid rgba(255, 200, 165, 0.24);
  border-radius: 999px;
  padding: 0.16rem 0.48rem;
  background: rgba(255, 255, 255, 0.08);
  color: #f0c7bd;
  font-size: 0.78rem;
  font-weight: 900;
}

.sample-card h3 {
  margin-top: 0.42rem;
  font-size: clamp(1.18rem, 2.4vw, 1.58rem);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1.2;
}

.sample-card-compact h3 {
  margin-top: 0.34rem;
  font-size: clamp(1.18rem, 2.1vw, 1.48rem);
}

.sample-author {
  margin-top: 0.35rem;
  color: #ffd8bd;
  font-size: 0.9rem;
  font-weight: 850;
}

.decision-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.42rem;
  margin-top: 0.68rem;
}

.sample-card-compact .decision-strip {
  gap: 0.42rem;
  margin-top: 0.62rem;
}

.decision-strip article {
  border: 1px solid rgba(255, 200, 165, 0.18);
  border-radius: 12px;
  padding: 0.46rem 0.5rem;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.1), rgba(255, 255, 255, 0.04));
}

.decision-strip span,
.decision-strip strong {
  display: block;
}

.decision-strip span {
  color: #f0c7bd;
  font-size: 0.7rem;
  font-weight: 900;
}

.decision-strip strong {
  margin-top: 0.12rem;
  color: #fff8f1;
  font-size: 0.84rem;
  font-weight: 950;
}

.sample-card-compact .decision-strip article {
  padding: 0.46rem 0.5rem;
}

.sample-card-compact .decision-strip strong {
  font-size: 0.82rem;
}

.sample-desc {
  margin-top: 0.68rem;
  border-top: 1px solid rgba(255, 255, 255, 0.12);
  padding-top: 0.62rem;
}

.sample-desc span {
  color: #f0c7bd;
  font-size: 0.76rem;
  font-weight: 950;
}

.sample-desc p {
  margin-top: 0.38rem;
  max-height: 5.6rem;
  overflow: auto;
  color: #f7ede4;
  line-height: 1.75;
}

.sample-profile {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
  margin-top: 0.68rem;
}

.sample-card-compact .sample-profile {
  gap: 0.5rem;
  margin-top: 0.62rem;
}

.sample-profile-topic {
  grid-column: 1 / -1;
}

.sample-profile div {
  border: 1px solid rgba(255, 200, 165, 0.16);
  border-radius: 10px;
  padding: 0.45rem 0.5rem;
  background: rgba(255, 255, 255, 0.06);
}

.sample-card-compact .sample-profile div {
  padding: 0.46rem 0.5rem;
}

.sample-profile dt {
  color: #f0c7bd;
  font-size: 0.75rem;
}

.sample-profile dd {
  margin-top: 0.16rem;
  font-weight: 850;
  line-height: 1.45;
  word-break: normal;
  overflow-wrap: break-word;
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
  min-height: 7.5rem;
  resize: vertical;
  line-height: 1.7;
  font-family: "Noto Sans SC", "Microsoft YaHei", "PingFang SC", sans-serif;
}

.sample-panel .copy-area {
  min-height: 4.7rem;
}

.sample-ops-board {
  display: grid;
  gap: 0.72rem;
  margin-top: 0.85rem;
}

.sample-ops-compact {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: stretch;
  gap: 0.75rem;
}

.sample-ops-board > section {
  border: 1px solid #eaded8;
  border-radius: 14px;
  padding: 0.82rem;
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.08), transparent 9rem),
    #fffdfb;
}

.sample-ops-board > section > span {
  display: inline-flex;
  border-radius: 999px;
  padding: 0.18rem 0.55rem;
  background: #fff7f3;
  color: #c8351f;
  font-size: 0.72rem;
  font-weight: 950;
}

.sample-ops-board > section > p {
  margin-top: 0.38rem;
  color: #6e5a54;
  font-size: 0.84rem;
  line-height: 1.65;
}

.sample-ops-board .backup-section {
  grid-column: 1 / -1;
  border-color: rgba(28, 117, 91, 0.22);
  background:
    radial-gradient(circle at 0% 0%, rgba(28, 117, 91, 0.12), transparent 10rem),
    #f7fffb;
}

.sample-ops-compact .backup-section:not(.backup-section-stack) {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(13.5rem, 0.78fr);
  gap: 0.65rem 0.9rem;
  align-items: center;
}

.sample-ops-compact .backup-section-stack {
  display: block;
}

.sample-ops-compact .backup-section:not(.backup-section-stack) > span,
.sample-ops-compact .backup-section:not(.backup-section-stack) > p,
.sample-ops-compact .backup-section:not(.backup-section-stack) .download-progress-card {
  grid-column: 1;
}

.sample-ops-compact .backup-section:not(.backup-section-stack) .consent-row,
.sample-ops-compact .backup-section:not(.backup-section-stack) .bottom-actions {
  grid-column: 2;
}

.sample-ops-board .backup-section > span {
  background: #eaf8f2;
  color: #1c755b;
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

.backup-section > p {
  margin-top: 0.35rem;
  color: #6e5a54;
  font-size: 0.82rem;
  line-height: 1.65;
}

.consent-row input {
  margin-top: 0.22rem;
  accent-color: #e8412e;
}

.download-progress-card {
  margin-top: 0.75rem;
  border: 1px solid rgba(232, 65, 46, 0.2);
  border-radius: 10px;
  padding: 0.72rem 0.78rem;
  background:
    linear-gradient(90deg, rgba(232, 65, 46, 0.1), transparent),
    #fffaf6;
  color: #493a35;
}

.download-progress-card strong {
  display: block;
  color: #c8351f;
  font-weight: 950;
}

.download-progress-card p {
  margin-top: 0.28rem;
  color: #6e5a54;
  font-size: 0.82rem;
  line-height: 1.6;
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
  font-family: "Noto Sans SC", "Microsoft YaHei", "PingFang SC", sans-serif;
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
.dark .empty-action-card,
.dark .sample-empty-guide,
.dark .rank-row,
.dark .consent-row,
.dark .action-result,
.dark .analysis-result {
  border-color: #312720;
  background: #171311;
}

.dark .empty-action-card,
.dark .sample-empty-guide {
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.12), transparent 12rem),
    #171311;
}

.dark .empty-action-card strong,
.dark .sample-empty-guide strong,
.dark .empty-steps b,
.dark .guide-flow b {
  color: #f7ede4;
}

.dark .empty-action-card p,
.dark .sample-empty-guide p {
  color: #cdbdb5;
}

.dark .empty-steps b,
.dark .guide-flow b {
  border-color: #312720;
  background: #211916;
}

.dark .download-progress-card {
  border-color: rgba(232, 65, 46, 0.24);
  background:
    linear-gradient(90deg, rgba(232, 65, 46, 0.12), transparent),
    #211916;
  color: #f7ede4;
}

.dark .download-progress-card p {
  color: #cdbdb5;
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

.dark .sample-ops-board > section {
  border-color: #312720;
  background:
    radial-gradient(circle at 0% 0%, rgba(232, 65, 46, 0.1), transparent 9rem),
    #171311;
}

.dark .sample-ops-board > section > p {
  color: #cdbdb5;
}

.dark .sample-ops-board .backup-section {
  border-color: rgba(28, 117, 91, 0.28);
  background:
    radial-gradient(circle at 0% 0%, rgba(28, 117, 91, 0.14), transparent 10rem),
    #121a16;
}

.dark .sample-fast-lane {
  border-color: #312720;
  background:
    linear-gradient(180deg, rgba(23, 19, 17, 0.96), rgba(33, 25, 22, 0.92));
}

.dark .sample-fast-lane button,
.dark .sample-fast-lane a {
  border-color: #312720;
  background: #211916;
  color: #f7ede4;
}

.dark .sample-fast-lane button:first-child {
  border-color: #f0c7bd;
  background: #f0c7bd;
  color: #241a16;
}

.dark .sample-fast-lane a {
  border-color: rgba(28, 117, 91, 0.36);
  background: rgba(28, 117, 91, 0.16);
  color: #86d7c0;
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
  .sample-profile,
  .decision-strip,
  .empty-action-card,
  .empty-steps,
  .rank-row,
  .sample-card,
  .sample-card.has-cover {
    grid-template-columns: 1fr;
  }

  .empty-orbit {
    width: 2.5rem;
    height: 2.5rem;
  }

  .book-score {
    white-space: normal;
  }

  .sample-card.sample-card-compact,
  .sample-card.sample-card-compact.has-cover {
    grid-template-columns: minmax(4.4rem, 5rem) minmax(0, 1fr);
  }

  .sample-card-compact .sample-cover-frame {
    max-height: 7.2rem;
  }

  .sample-card-compact .decision-strip,
  .sample-card-compact .sample-profile {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .sample-ops-compact,
  .sample-fast-lane,
  .sample-ops-compact .backup-section {
    grid-template-columns: 1fr;
  }

  .sample-ops-compact .backup-section > span,
  .sample-ops-compact .backup-section > p,
  .sample-ops-compact .download-progress-card,
  .sample-ops-compact .backup-section .consent-row,
  .sample-ops-compact .backup-section .bottom-actions {
    grid-column: 1;
  }
}
</style>
