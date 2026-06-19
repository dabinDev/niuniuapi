<template>
  <AppLayout>
    <div class="creative-page studio-wide-shell mx-auto max-w-none">
      <header class="creative-head">
        <div>
          <span class="eyebrow">Writing Bench</span>
          <h1>创作生成</h1>
          <p>围绕同一份素材生成大纲、正文续写、改写增强和改编脚本。每次结果都会进入作品归档，方便回看和继续加工。</p>
        </div>
        <router-link class="head-link studio-action-link" to="/studio/works">查看我的作品</router-link>
      </header>

      <div class="model-strip model-strip-compact" :class="{ ready: configured }" data-test="model-auto-config-strip">
        <span>{{ configured ? '文案模型已就绪' : '自动配置流程' }}</span>
        <strong>{{ configured ? textModel : '创建第一把密钥后，系统会优先自动选择第一把密钥的最新文案模型' }}</strong>
        <p class="model-strip-copy">{{ configured ? '如需覆盖默认值，可在 API 密钥页手动测试后保存。' : '自动选择最新文案模型；手动测试后保存会记住你的选择，不手动修改时创作台继续使用自动配置。' }}</p>
        <router-link to="/keys">{{ configured ? '模型设置' : '检查密钥' }}</router-link>
      </div>

      <div v-if="bridgeNotice" class="bridge-notice" data-test="creative-bridge-notice">
        {{ bridgeNotice }}
      </div>

      <section class="creative-grid creative-grid-wide">
        <form class="writer-panel" @submit.prevent="submit">
          <div class="template-strip" aria-label="创作模板">
            <button
              v-for="tpl in creativeTemplates"
              :key="tpl.key"
              :data-test="`creative-template-${tpl.key}`"
              type="button"
              @click="applyCreativeTemplate(tpl.key)"
            >
              <strong>{{ tpl.title }}</strong>
              <span>{{ tpl.desc }}</span>
            </button>
          </div>

          <div class="mode-grid mode-grid-wide" aria-label="创作模式">
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

          <div class="output-contract" data-test="creative-output-contract" aria-label="输出承诺">
            <div>
              <span>输出承诺</span>
              <strong>{{ outputContract.title }}</strong>
            </div>
            <ul>
              <li v-for="item in outputContract.points" :key="item">{{ item }}</li>
            </ul>
            <p class="model-strip-copy">生成后检查：人物动机、设定代价、章尾钩子和下一步归档动作。</p>
          </div>

          <div class="field-grid">
            <label>
              <span>题材</span>
              <input data-test="creative-genre" v-model="genre" type="text" placeholder="玄幻 / 都市 / 女频 / 悬疑" />
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
              rows="10"
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

            <div class="result-actions" aria-label="生成结果操作">
              <button type="button" @click="copyResult">复制结果</button>
              <router-link to="/studio/works">查看归档</router-link>
            </div>
            <p v-if="copyStatus" class="copy-status">{{ copyStatus }}</p>

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
            <div class="empty-kicker">
              <div class="draft-mark">GEN</div>
              <span>先定生成模式</span>
            </div>
            <h2>把诊断结论变成下一版稿子</h2>
            <p>选择生成类型，粘贴素材，这里会输出能复制、能归档、能继续加工的内容。适合承接热榜样本、拆书报告和爆款对标动作。</p>
            <button data-test="creative-empty-sample" type="button" class="empty-action" @click="applyCreativeTemplate('rewrite')">
              先填入改写示例
            </button>
            <div class="empty-guide" aria-label="创作生成使用路径">
              <strong>生成路径</strong>
              <ol>
                <li>选择大纲、续写、改写、短剧、分镜或长剧本。</li>
                <li>粘贴正文、拆书结论或爆款对标动作。</li>
                <li>复制结果或进入“我的作品”继续加工。</li>
              </ol>
            </div>
            <div class="empty-preview" aria-label="创作生成结果预览">
              <article>
                <span>正文</span>
                <strong>分段产出</strong>
                <small>按大纲、续写、改写或剧本结构生成</small>
              </article>
              <article>
                <span>检查</span>
                <strong>一致性提示</strong>
                <small>人物、设定、节奏和伏笔的风险点</small>
              </article>
              <article>
                <span>归档</span>
                <strong>继续加工</strong>
                <small>结果会进入我的作品，方便后续复制和改稿</small>
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
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  generateCreative,
  getModelConfig,
  type CreativeMode,
  type CreativeResult,
} from '@/api/studio'
import { consumeStudioBridgePayload } from '@/utils/studioBridge'

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
const copyStatus = ref('')
const bridgeNotice = ref('')

const modeOptions: { value: CreativeMode; label: string; hint: string }[] = [
  { value: 'outline', label: '大纲', hint: '卷纲、主线、章节钩子' },
  { value: 'draft', label: '正文续写', hint: '承接原文继续写' },
  { value: 'rewrite', label: '改写', hint: '增强冲突和爽点' },
  { value: 'short', label: '短剧', hint: '竖屏强冲突版本' },
  { value: 'storyboard', label: '分镜', hint: '镜头、旁白、时长' },
  { value: 'long', label: '长剧本', hint: '场景、动作、对白' },
]

const creativeTemplates = [
  { key: 'rewrite', title: '改写增强', desc: '补冲突、补爽点、补章尾钩子' },
  { key: 'outline', title: '热榜立项', desc: '题材卖点到卷纲章节' },
  { key: 'short', title: '短剧切片', desc: '小说桥段改成强冲突分集' },
]

const contentLength = computed(() => content.value.trim().length)
const canSubmit = computed(() => configured.value && contentLength.value >= MIN_LEN && !loading.value)
const outputContract = computed(() => {
  const contracts: Record<CreativeMode, { title: string; points: string[] }> = {
    outline: {
      title: '卷纲 + 主线 + 章节钩子',
      points: ['输出卷纲骨架', '拆出章节钩子', '标注每 3 章的小爽点'],
    },
    draft: {
      title: '承接原文的下一段正文',
      points: ['延续人物口吻', '推进当前冲突', '保留下一章悬念'],
    },
    rewrite: {
      title: '把弱段落改成可追读版本',
      points: ['补强冲突', '补首次爽点', '重写章尾钩子'],
    },
    short: {
      title: '强冲突分集短剧方案',
      points: ['输出强冲突分集', '每集一个反转点', '给出竖屏开场钩子'],
    },
    storyboard: {
      title: '镜头 + 旁白 + 节奏表',
      points: ['拆镜头动作', '补旁白承接', '标注单镜头时长'],
    },
    long: {
      title: '场景化长剧本',
      points: ['输出场景段落', '补动作和对白', '保留角色动机线'],
    },
  }
  return contracts[mode.value]
})
const resultText = computed(() => {
  if (!result.value) return ''
  const sections = result.value.sections.map((section) => `${section.heading}\n${section.content}`)
  return [result.value.title, result.value.summary, ...sections].filter(Boolean).join('\n\n')
})

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
  const payload = consumeStudioBridgePayload('generate')
  if (!payload) return
  mode.value = 'outline'
  genre.value = payload.genre
  brief.value = payload.brief
  content.value = payload.content
  bridgeNotice.value = payload.source === 'works'
    ? '已从我的作品带入素材，可直接生成下一版。'
    : '已从番茄热榜带入素材，可直接生成新书方向或改稿方案。'
}

function modeLabel(value: string) {
  return modeOptions.find((opt) => opt.value === value)?.label || value
}

function applyCreativeTemplate(key: string) {
  if (key === 'rewrite') {
    mode.value = 'rewrite'
    genre.value = '玄幻悬疑'
    style.value = '高压迫爽文'
    brief.value = '把第 2 章改成一次低成本胜利，结尾留更大的记忆代价钩子'
    targetWords.value = 1800
    episodes.value = undefined
    content.value = [
      '原章节：主角进入废弃书塔后，花了大量篇幅解释兑换规则、等级和历史。追债人直到章末才出现，主角还没有真正反击。',
      '问题：设定解释偏长，爽点兑现太慢，主角的主动性不足。',
      '目标：保留“记忆兑换禁忌知识”的设定，但让主角在本章完成一次聪明的小胜利，并用失去关键记忆作为章尾钩子。',
    ].join('\n\n')
    return
  }
  if (key === 'outline') {
    mode.value = 'outline'
    genre.value = '都市异能'
    style.value = '热榜爽文'
    brief.value = '围绕热榜卖点生成 30 章新书大纲'
    targetWords.value = undefined
    episodes.value = 30
    content.value = '题材卖点：底层主角被羞辱后获得有代价的异能，每 3 章一次小胜利，每 10 章一次身份升级。请生成卷一主线、人物关系和章末钩子。'
    return
  }
  mode.value = 'short'
  genre.value = '女频复仇'
  style.value = '短剧快节奏'
  brief.value = '拆成 6 集，每集 1 个强冲突和 1 个反转'
  targetWords.value = undefined
  episodes.value = 6
  content.value = '小说桥段：女主被未婚夫和妹妹联手背叛，继承母亲留下的公司暗线，准备在订婚宴公开反击。请改成竖屏短剧分集。'
}

async function submit() {
  if (!canSubmit.value) return
  loading.value = true
  errorMsg.value = ''
  copyStatus.value = ''
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

async function copyResult() {
  if (!resultText.value) return
  try {
    await navigator.clipboard.writeText(resultText.value)
    copyStatus.value = '已复制生成结果'
  } catch {
    copyStatus.value = '复制失败，请手动选择结果内容'
  }
}
</script>

<style scoped>
.creative-page {
  width: min(100%, 118rem);
  max-width: calc(100vw - 1.25rem);
  padding: 0.5rem 0 2.5rem;
  color: #221a18;
}

.studio-wide-shell {
  width: min(100%, 118rem);
  max-width: calc(100vw - 1.25rem);
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
  font-size: 1.85rem;
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

.creative-grid {
  display: grid;
  grid-template-columns: minmax(340px, 0.9fr) minmax(0, 1.1fr);
  gap: 1rem;
}

.creative-grid-wide {
  grid-template-columns: minmax(30rem, 0.9fr) minmax(0, 1.1fr);
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

.template-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.55rem;
  margin-bottom: 0.9rem;
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
  line-height: 1.45;
}

.mode-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.55rem;
  margin-bottom: 0.9rem;
}

.mode-grid-wide {
  grid-template-columns: repeat(6, minmax(0, 1fr));
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

.output-contract {
  display: grid;
  grid-template-columns: 0.48fr 1fr;
  gap: 0.75rem;
  border: 1px solid rgba(232, 65, 46, 0.22);
  border-radius: 12px;
  padding: 0.85rem;
  margin-bottom: 0.9rem;
  background:
    radial-gradient(circle at 100% 0%, rgba(47, 93, 159, 0.1), transparent 10rem),
    #fffaf6;
}

.output-contract span {
  color: #c8351f;
  font-size: 0.76rem;
  font-weight: 950;
}

.output-contract strong {
  display: block;
  margin-top: 0.25rem;
  color: #241a16;
  line-height: 1.45;
}

.output-contract ul {
  display: grid;
  gap: 0.25rem;
  margin: 0;
  padding-left: 1rem;
  color: #5d4640;
  font-size: 0.84rem;
  line-height: 1.55;
}

.output-contract p {
  grid-column: 1 / -1;
  margin: 0;
  border-top: 1px dashed rgba(232, 65, 46, 0.22);
  padding-top: 0.62rem;
  color: #7b625a;
  font-size: 0.84rem;
  font-weight: 850;
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
  min-height: 34rem;
  padding: 1rem;
}

.empty-output {
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

.empty-output h2 {
  margin: 0;
  color: #221a18;
  font-size: 1.25rem;
  font-weight: 950;
  text-align: center;
}

.empty-output > p {
  max-width: 38rem;
  margin: 0 auto;
  text-align: center;
  line-height: 1.75;
}

.draft-mark {
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

.result-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.result-actions button,
.result-actions a {
  border: 1px solid #eaded8;
  border-radius: 8px;
  padding: 0.52rem 0.8rem;
  background: #fffdfb;
  color: #493a35;
  font-weight: 850;
}

.result-actions button {
  border-color: #e8412e;
  background: #e8412e;
  color: #fff;
}

.copy-status {
  margin-top: 0.45rem;
  color: #1c755b;
  text-align: right;
  font-size: 0.85rem;
  font-weight: 850;
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
.dark .empty-preview article,
.dark .scene,
.dark .post-grid section {
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
.dark .mode-grid button {
  border-color: #3a2d26;
  background: #211916;
  color: #f7ede4;
}

.dark .output-contract {
  border-color: rgba(232, 65, 46, 0.28);
  background:
    radial-gradient(circle at 100% 0%, rgba(47, 93, 159, 0.14), transparent 10rem),
    #171311;
}

.dark .output-contract strong {
  color: #f7ede4;
}

.dark .output-contract ul,
.dark .output-contract p {
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
  .model-strip-compact {
    grid-template-columns: 1fr;
  }

  .model-strip-compact .model-strip-copy,
  .model-strip-compact a {
    grid-column: 1;
    grid-row: auto;
  }

  .mode-grid,
  .template-strip,
  .output-contract,
  .field-grid,
  .number-grid,
  .empty-preview {
    grid-template-columns: 1fr;
  }
}
</style>
