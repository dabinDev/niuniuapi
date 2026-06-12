<template>
  <div class="lingxi-home min-h-screen overflow-hidden bg-[#f7f3ea] text-[#211f1a] dark:bg-[#151715] dark:text-[#f6f0e4]">
    <header class="relative z-30 border-b border-[#272016]/10 bg-[#f7f3ea]/86 backdrop-blur-xl dark:border-white/10 dark:bg-[#151715]/86">
      <nav class="mx-auto flex max-w-7xl items-center justify-between px-5 py-4 sm:px-7 lg:px-10">
        <router-link to="/home" class="brand-mark group flex items-center gap-3" aria-label="灵犀文创首页">
          <span class="brand-seal">犀</span>
          <span>
            <span class="block text-base font-semibold tracking-[0.18em] text-[#211f1a] dark:text-[#f6f0e4]">灵犀文创</span>
            <span class="block text-[11px] tracking-[0.24em] text-[#716957] dark:text-[#b7ad9a]">LINGXI CREATIVE</span>
          </span>
        </router-link>

        <div class="flex items-center gap-2 sm:gap-3">
          <router-link to="/purchase" class="hidden px-3 py-2 text-sm font-medium text-[#554d3f] transition hover:text-[#0d766f] dark:text-[#d7cdbb] dark:hover:text-[#70e0d4] sm:inline-flex">
            套餐
          </router-link>
          <LocaleSwitcher class="hidden sm:block" />
          <button
            class="grid h-9 w-9 place-items-center border border-[#2a2418]/12 bg-white/55 text-[#5c5546] transition hover:border-[#0d766f]/40 hover:text-[#0d766f] dark:border-white/12 dark:bg-white/6 dark:text-[#d8cebd] dark:hover:text-[#70e0d4]"
            :title="isDark ? '切换到浅色' : '切换到深色'"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="sm" />
            <Icon v-else name="moon" size="sm" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="nav-primary"
          >
            {{ isAuthenticated ? '进入控制台' : '登录' }}
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section ref="heroSection" class="relative border-b border-[#272016]/10 dark:border-white/10">
        <div class="paper-grid absolute inset-0"></div>
        <div class="mx-auto grid min-h-[calc(100vh-73px)] max-w-7xl items-center gap-10 px-5 py-12 sm:px-7 lg:grid-cols-[0.9fr_1.1fr] lg:px-10 lg:py-16">
          <div class="relative z-10 max-w-2xl">
            <div class="hero-kicker mb-5 inline-flex items-center gap-2 border border-[#0d766f]/20 bg-white/55 px-3 py-2 text-xs font-semibold tracking-[0.18em] text-[#0d766f] shadow-sm dark:border-[#70e0d4]/25 dark:bg-white/6 dark:text-[#70e0d4]">
              AI 写作 · AIGC 视频 · 创作 Skill
            </div>
            <h1 class="hero-title text-balance text-[clamp(3rem,8vw,7.6rem)] font-black leading-[0.88] tracking-[-0.02em]">
              灵犀文创
            </h1>
            <p class="hero-subtitle mt-6 max-w-xl text-balance text-xl font-medium leading-8 text-[#4f493c] dark:text-[#d9cfbc] sm:text-2xl sm:leading-9">
              为小说作者与内容创作者打造的 AI 创作平台。
            </p>
            <p class="hero-copy mt-5 max-w-2xl text-base leading-8 text-[#746c5c] dark:text-[#aaa18f]">
              从灵感、角色、世界观到章节续写和视频分镜，把零散想法整理成可以持续生长的作品宇宙。
            </p>

            <div class="hero-actions mt-8 flex flex-col gap-3 sm:flex-row">
              <router-link :to="isAuthenticated ? dashboardPath : '/register'" class="cta-main">
                {{ isAuthenticated ? '进入创作台' : '开始创作' }}
                <span aria-hidden="true">→</span>
              </router-link>
              <router-link to="/purchase" class="cta-secondary">
                查看创作套餐
              </router-link>
            </div>

            <div class="hero-metrics mt-10 grid max-w-xl grid-cols-3 border-y border-[#2a2418]/10 py-4 dark:border-white/10">
              <div v-for="metric in metrics" :key="metric.label" class="metric-item">
                <strong>{{ metric.value }}</strong>
                <span>{{ metric.label }}</span>
              </div>
            </div>
          </div>

          <div ref="deskScene" class="creator-desk relative z-10">
            <div class="desk-shell">
              <div class="desk-toolbar">
                <span>长篇项目</span>
                <div class="flex gap-1.5">
                  <i></i><i></i><i></i>
                </div>
              </div>
              <div class="desk-layout">
                <aside class="desk-sidebar">
                  <div class="sidebar-title">灵感库</div>
                  <button v-for="item in sidebarItems" :key="item" class="sidebar-pill">{{ item }}</button>
                </aside>
                <div class="desk-main">
                  <div class="chapter-card floating-card">
                    <div class="card-eyebrow">Chapter 07</div>
                    <h3>雨夜，主角第一次听见城市的心跳</h3>
                    <p>系统正在根据角色动机、上一章情绪曲线与伏笔清单生成下一段转折...</p>
                    <div class="writing-lines">
                      <span></span><span></span><span></span>
                    </div>
                  </div>
                  <div class="desk-card character-card floating-card">
                    <span>角色卡</span>
                    <strong>林见微</strong>
                    <p>外冷内热，记忆缺口，擅长解构梦境。</p>
                  </div>
                  <div class="desk-card world-card floating-card">
                    <span>世界观</span>
                    <strong>北境书塔</strong>
                    <p>每本书都会生成一座可进入的城市。</p>
                  </div>
                  <div class="skill-node node-a">爽点增强</div>
                  <div class="skill-node node-b">伏笔检查</div>
                  <div class="skill-node node-c">视频分镜</div>
                  <svg class="skill-lines" viewBox="0 0 420 280" aria-hidden="true">
                    <path class="flow-path" d="M70 214 C130 142 180 120 250 154 S338 150 374 82" />
                    <path class="flow-path delay" d="M92 74 C150 112 190 168 250 154 S320 182 366 220" />
                  </svg>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="section-band">
        <div class="mx-auto max-w-7xl px-5 py-16 sm:px-7 lg:px-10">
          <div class="section-heading">
            <p>创作流程</p>
            <h2>让故事从一句灵感，走到完整作品。</h2>
          </div>
          <div class="mt-10 grid gap-4 md:grid-cols-4">
            <article v-for="(step, index) in workflow" :key="step.title" class="workflow-card reveal-card">
              <span>{{ String(index + 1).padStart(2, '0') }}</span>
              <h3>{{ step.title }}</h3>
              <p>{{ step.desc }}</p>
            </article>
          </div>
        </div>
      </section>

      <section class="section-band alt">
        <div class="mx-auto grid max-w-7xl gap-10 px-5 py-16 sm:px-7 lg:grid-cols-[0.8fr_1.2fr] lg:px-10">
          <div class="section-heading sticky-copy">
            <p>Skill 平台</p>
            <h2>把作者的经验，沉淀成可复用的创作技能。</h2>
            <router-link to="/purchase" class="mt-6 inline-flex w-fit border border-[#0d766f]/30 px-4 py-2 text-sm font-semibold text-[#0d766f] transition hover:bg-[#0d766f] hover:text-white dark:text-[#70e0d4] dark:hover:text-[#151715]">
              解锁更多 Skill
            </router-link>
          </div>
          <div class="skill-grid">
            <article v-for="skill in skills" :key="skill.title" class="skill-card reveal-card">
              <div class="skill-symbol">{{ skill.symbol }}</div>
              <h3>{{ skill.title }}</h3>
              <p>{{ skill.desc }}</p>
            </article>
          </div>
        </div>
      </section>

      <section class="mx-auto max-w-7xl px-5 py-16 sm:px-7 lg:px-10">
        <div class="conversion-panel reveal-card">
          <div>
            <p class="text-sm font-semibold tracking-[0.2em] text-[#0d766f] dark:text-[#70e0d4]">创作额度</p>
            <h2 class="mt-3 max-w-2xl text-3xl font-black leading-tight sm:text-5xl">
              购买创作额度，开启更长的故事线。
            </h2>
            <p class="mt-4 max-w-2xl text-base leading-8 text-[#746c5c] dark:text-[#b5ac9b]">
              套餐用于 AI 写作、角色设定、章节生成、AIGC 视频脚本与后续插件调用。支付和额度仍由底层系统稳定管理。
            </p>
          </div>
          <div class="flex flex-col gap-3 sm:flex-row lg:flex-col">
            <router-link to="/purchase" class="cta-main">查看套餐 <span aria-hidden="true">→</span></router-link>
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="cta-secondary">
              {{ isAuthenticated ? '进入控制台' : '登录账户' }}
            </router-link>
          </div>
        </div>
      </section>
    </main>

    <footer class="border-t border-[#272016]/10 bg-[#eee7d9] px-5 py-8 dark:border-white/10 dark:bg-[#101210] sm:px-7 lg:px-10">
      <div class="mx-auto flex max-w-7xl flex-col gap-4 text-sm text-[#746c5c] dark:text-[#aaa18f] md:flex-row md:items-center md:justify-between">
        <p>© {{ currentYear }} 灵犀文创 · AI 写作与内容创作平台</p>
        <div class="flex flex-wrap gap-4">
          <router-link to="/home" class="hover:text-[#0d766f] dark:hover:text-[#70e0d4]">首页</router-link>
          <router-link to="/purchase" class="hover:text-[#0d766f] dark:hover:text-[#70e0d4]">套餐</router-link>
          <router-link :to="dashboardPath" class="hover:text-[#0d766f] dark:hover:text-[#70e0d4]">控制台</router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="hover:text-[#0d766f] dark:hover:text-[#70e0d4]">文档</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { gsap } from 'gsap'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const authStore = useAuthStore()
const appStore = useAppStore()

const heroSection = ref<HTMLElement | null>(null)
const deskScene = ref<HTMLElement | null>(null)
const isDark = ref(document.documentElement.classList.contains('dark'))
let animationContext: gsap.Context | null = null
let removeParallaxListener: (() => void) | null = null

const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const currentYear = computed(() => new Date().getFullYear())

const metrics = [
  { value: '6+', label: '创作 Skill' },
  { value: '24h', label: '灵感归档' },
  { value: 'AI', label: '持续协作' },
]

const sidebarItems = ['故事种子', '角色关系', '世界观', '章节草稿']

const workflow = [
  { title: '捕捉灵感', desc: '把一句想法、一个画面或一段设定保存成可扩展的故事种子。' },
  { title: '构建设定', desc: '沉淀角色、地点、组织、时间线，让作品世界保持一致。' },
  { title: '生成章节', desc: '按大纲、文风和伏笔续写章节，减少卡文时的空白感。' },
  { title: '扩展内容', desc: '把小说内容延展为短视频脚本、分镜和宣发素材。' },
]

const skills = [
  { symbol: '人', title: '角色小传', desc: '生成动机、弱点、口癖、人物弧光与关系冲突。' },
  { symbol: '纲', title: '章节大纲', desc: '把灵感拆成起承转合，保留节奏和悬念。' },
  { symbol: '燃', title: '爽点增强', desc: '检查情绪峰值、反转力度和读者期待兑现。' },
  { symbol: '伏', title: '伏笔检查', desc: '追踪线索、暗示和回收点，减少剧情断裂。' },
  { symbol: '润', title: '文风润色', desc: '统一叙述口吻，让章节更贴近目标风格。' },
  { symbol: '映', title: '视频分镜', desc: '把章节场景转成镜头、旁白和画面提示。' },
]

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

function initAnimations() {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (reduceMotion || !heroSection.value) return

  animationContext = gsap.context(() => {
    const tl = gsap.timeline({ defaults: { ease: 'power3.out', duration: 0.8 } })
    tl.from('.hero-kicker', { y: 18, autoAlpha: 0 })
      .from('.hero-title', { y: 34, autoAlpha: 0, duration: 1 }, '<0.08')
      .from('.hero-subtitle, .hero-copy', { y: 24, autoAlpha: 0, stagger: 0.1 }, '<0.2')
      .from('.hero-actions > *', { y: 18, autoAlpha: 0, stagger: 0.08 }, '<0.2')
      .from('.metric-item', { y: 14, autoAlpha: 0, stagger: 0.08 }, '<0.1')
      .from('.desk-shell', { x: 42, y: 24, rotation: 1.2, autoAlpha: 0, duration: 1 }, 0.18)
      .from('.floating-card, .skill-node', { y: 20, autoAlpha: 0, stagger: 0.07 }, '<0.35')

    gsap.to('.floating-card', {
      y: (index) => (index % 2 === 0 ? -8 : 7),
      duration: 3.6,
      ease: 'sine.inOut',
      repeat: -1,
      yoyo: true,
      stagger: 0.2,
    })

    gsap.to('.skill-node', {
      scale: 1.035,
      duration: 2.5,
      ease: 'sine.inOut',
      repeat: -1,
      yoyo: true,
      stagger: 0.18,
    })

    gsap.from('.reveal-card', {
      y: 28,
      autoAlpha: 0,
      duration: 0.75,
      ease: 'power2.out',
      stagger: 0.08,
    })
  }, heroSection.value)
}

function initParallax() {
  if (!deskScene.value || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  const xTo = gsap.quickTo(deskScene.value, 'x', { duration: 0.5, ease: 'power3.out' })
  const yTo = gsap.quickTo(deskScene.value, 'y', { duration: 0.5, ease: 'power3.out' })
  const onMove = (event: MouseEvent) => {
    const rect = deskScene.value?.getBoundingClientRect()
    if (!rect) return
    xTo(((event.clientX - rect.left) / rect.width - 0.5) * 12)
    yTo(((event.clientY - rect.top) / rect.height - 0.5) * 10)
  }
  deskScene.value.addEventListener('mousemove', onMove)
  removeParallaxListener = () => deskScene.value?.removeEventListener('mousemove', onMove)
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    void appStore.fetchPublicSettings()
  }
  initAnimations()
  initParallax()
})

onBeforeUnmount(() => {
  removeParallaxListener?.()
  animationContext?.revert()
})
</script>

<style scoped>
.lingxi-home {
  font-family: "Noto Serif SC", "Songti SC", "Microsoft YaHei", serif;
}

.paper-grid {
  background:
    linear-gradient(rgba(31, 28, 20, 0.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(31, 28, 20, 0.04) 1px, transparent 1px);
  background-size: 44px 44px;
  mask-image: linear-gradient(to bottom, black 0%, transparent 88%);
}

.dark .paper-grid {
  background:
    linear-gradient(rgba(246, 240, 228, 0.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(246, 240, 228, 0.035) 1px, transparent 1px);
}

.brand-seal {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border: 1px solid rgba(13, 118, 111, 0.34);
  background: #0d766f;
  color: #fff9ec;
  font-weight: 900;
  box-shadow: 6px 6px 0 rgba(13, 118, 111, 0.12);
}

.nav-primary,
.cta-main,
.cta-secondary {
  display: inline-flex;
  min-height: 42px;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border: 1px solid transparent;
  padding: 10px 16px;
  font-weight: 800;
  transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease, color 0.2s ease;
}

.nav-primary,
.cta-main {
  background: #0d766f;
  color: #fffaf0;
  box-shadow: 6px 6px 0 rgba(13, 118, 111, 0.16);
}

.nav-primary:hover,
.cta-main:hover {
  transform: translate(-2px, -2px);
  box-shadow: 9px 9px 0 rgba(13, 118, 111, 0.18);
}

.cta-secondary {
  border-color: rgba(42, 36, 24, 0.14);
  background: rgba(255, 255, 255, 0.6);
  color: #2b281f;
}

.dark .cta-secondary {
  border-color: rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.06);
  color: #f6f0e4;
}

.cta-secondary:hover {
  border-color: rgba(13, 118, 111, 0.36);
  color: #0d766f;
}

.hero-title {
  font-family: "STKaiti", "KaiTi", "Noto Serif SC", serif;
  text-shadow: 0 12px 30px rgba(52, 45, 31, 0.08);
}

.metric-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-item strong {
  font-size: clamp(1.2rem, 3vw, 1.7rem);
  line-height: 1;
}

.metric-item span {
  font-size: 12px;
  color: #746c5c;
}

.dark .metric-item span {
  color: #aaa18f;
}

.creator-desk {
  min-height: 520px;
}

.desk-shell {
  overflow: hidden;
  border: 1px solid rgba(42, 36, 24, 0.12);
  background: rgba(255, 252, 244, 0.78);
  box-shadow: 0 28px 70px rgba(52, 45, 31, 0.14), 12px 12px 0 rgba(13, 118, 111, 0.08);
  backdrop-filter: blur(18px);
}

.dark .desk-shell {
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(32, 34, 31, 0.78);
  box-shadow: 0 28px 70px rgba(0, 0, 0, 0.36), 12px 12px 0 rgba(112, 224, 212, 0.08);
}

.desk-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid rgba(42, 36, 24, 0.1);
  padding: 14px 16px;
  color: #746c5c;
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.16em;
}

.dark .desk-toolbar {
  border-bottom-color: rgba(255, 255, 255, 0.1);
  color: #aaa18f;
}

.desk-toolbar i {
  display: block;
  width: 8px;
  height: 8px;
  background: #0d766f;
}

.desk-layout {
  display: grid;
  min-height: 470px;
  grid-template-columns: 138px 1fr;
}

.desk-sidebar {
  border-right: 1px solid rgba(42, 36, 24, 0.1);
  padding: 18px 14px;
}

.dark .desk-sidebar {
  border-right-color: rgba(255, 255, 255, 0.1);
}

.sidebar-title {
  margin-bottom: 14px;
  color: #0d766f;
  font-size: 12px;
  font-weight: 900;
  letter-spacing: 0.18em;
}

.sidebar-pill {
  margin-bottom: 10px;
  width: 100%;
  border: 1px solid rgba(42, 36, 24, 0.08);
  background: rgba(255, 255, 255, 0.54);
  padding: 9px 10px;
  text-align: left;
  font-size: 13px;
  color: #5d5548;
}

.dark .sidebar-pill {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.06);
  color: #d8cebd;
}

.desk-main {
  position: relative;
  min-height: 470px;
  padding: 20px;
}

.chapter-card,
.desk-card {
  position: absolute;
  border: 1px solid rgba(42, 36, 24, 0.12);
  background: #fffaf0;
  box-shadow: 0 18px 40px rgba(52, 45, 31, 0.12);
}

.dark .chapter-card,
.dark .desk-card {
  border-color: rgba(255, 255, 255, 0.12);
  background: #20221f;
  box-shadow: 0 18px 40px rgba(0, 0, 0, 0.26);
}

.chapter-card {
  left: 26px;
  top: 24px;
  width: min(76%, 390px);
  padding: 22px;
}

.card-eyebrow,
.desk-card span {
  color: #0d766f;
  font-size: 12px;
  font-weight: 900;
  letter-spacing: 0.18em;
}

.chapter-card h3 {
  margin-top: 10px;
  max-width: 300px;
  font-size: 22px;
  font-weight: 900;
  line-height: 1.24;
}

.chapter-card p,
.desk-card p {
  margin-top: 10px;
  color: #746c5c;
  font-size: 13px;
  line-height: 1.8;
}

.dark .chapter-card p,
.dark .desk-card p {
  color: #aaa18f;
}

.writing-lines {
  margin-top: 18px;
  display: grid;
  gap: 8px;
}

.writing-lines span {
  display: block;
  height: 7px;
  background: linear-gradient(90deg, rgba(13, 118, 111, 0.3), rgba(13, 118, 111, 0.05));
}

.writing-lines span:nth-child(2) {
  width: 78%;
}

.writing-lines span:nth-child(3) {
  width: 58%;
}

.desk-card {
  width: 180px;
  padding: 16px;
}

.desk-card strong {
  margin-top: 8px;
  display: block;
  font-size: 18px;
}

.character-card {
  right: 22px;
  top: 170px;
}

.world-card {
  left: 70px;
  bottom: 28px;
}

.skill-node {
  position: absolute;
  z-index: 3;
  border: 1px solid rgba(13, 118, 111, 0.32);
  background: #e6f4ef;
  padding: 8px 10px;
  color: #0d766f;
  font-size: 12px;
  font-weight: 900;
  box-shadow: 5px 5px 0 rgba(13, 118, 111, 0.08);
}

.dark .skill-node {
  background: rgba(112, 224, 212, 0.1);
  color: #70e0d4;
}

.node-a {
  right: 40px;
  top: 70px;
}

.node-b {
  left: 22px;
  top: 260px;
}

.node-c {
  right: 30px;
  bottom: 34px;
}

.skill-lines {
  position: absolute;
  inset: 60px 20px 20px;
  width: calc(100% - 40px);
  height: calc(100% - 80px);
  pointer-events: none;
}

.flow-path {
  fill: none;
  stroke: rgba(13, 118, 111, 0.34);
  stroke-width: 2;
  stroke-dasharray: 8 10;
  animation: flow 2.8s linear infinite;
}

.flow-path.delay {
  animation-delay: 0.9s;
  opacity: 0.6;
}

@keyframes flow {
  to {
    stroke-dashoffset: -36;
  }
}

.section-band {
  border-bottom: 1px solid rgba(39, 32, 22, 0.1);
  background: #fffaf0;
}

.section-band.alt {
  background: #efe8da;
}

.dark .section-band {
  border-bottom-color: rgba(255, 255, 255, 0.1);
  background: #191b18;
}

.dark .section-band.alt {
  background: #111310;
}

.section-heading p {
  color: #0d766f;
  font-size: 13px;
  font-weight: 900;
  letter-spacing: 0.22em;
}

.section-heading h2 {
  margin-top: 12px;
  max-width: 760px;
  font-size: clamp(2rem, 5vw, 4rem);
  font-weight: 950;
  line-height: 1.05;
  letter-spacing: -0.01em;
}

.workflow-card,
.skill-card,
.conversion-panel {
  border: 1px solid rgba(42, 36, 24, 0.12);
  background: rgba(255, 255, 255, 0.52);
  box-shadow: 0 16px 36px rgba(52, 45, 31, 0.08);
}

.dark .workflow-card,
.dark .skill-card,
.dark .conversion-panel {
  border-color: rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.055);
  box-shadow: none;
}

.workflow-card {
  min-height: 220px;
  padding: 22px;
}

.workflow-card span {
  color: rgba(13, 118, 111, 0.44);
  font-size: 34px;
  font-weight: 950;
}

.workflow-card h3,
.skill-card h3 {
  margin-top: 22px;
  font-size: 20px;
  font-weight: 900;
}

.workflow-card p,
.skill-card p {
  margin-top: 12px;
  color: #746c5c;
  font-size: 14px;
  line-height: 1.8;
}

.dark .workflow-card p,
.dark .skill-card p {
  color: #aaa18f;
}

.sticky-copy {
  align-self: start;
}

.skill-grid {
  display: grid;
  gap: 4px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.skill-card {
  min-height: 210px;
  padding: 22px;
}

.skill-symbol {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  background: #0d766f;
  color: white;
  font-weight: 950;
}

.conversion-panel {
  display: grid;
  gap: 28px;
  padding: clamp(24px, 5vw, 52px);
}

@media (min-width: 1024px) {
  .conversion-panel {
    grid-template-columns: 1fr 230px;
    align-items: end;
  }
}

@media (max-width: 760px) {
  .desk-layout {
    grid-template-columns: 1fr;
  }

  .desk-sidebar {
    display: none;
  }

  .creator-desk {
    min-height: 480px;
  }

  .desk-main {
    min-height: 430px;
  }

  .chapter-card {
    left: 14px;
    width: calc(100% - 28px);
  }

  .character-card {
    right: 14px;
    top: 230px;
  }

  .world-card {
    left: 14px;
    bottom: 24px;
  }

  .skill-node,
  .skill-lines {
    display: none;
  }

  .skill-grid {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .flow-path {
    animation: none;
  }
}
</style>
