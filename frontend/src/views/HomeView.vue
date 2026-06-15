<template>
  <div class="lanfanqie-home">
    <div class="grain" aria-hidden="true"></div>

    <header class="site-header">
      <nav class="nav-inner">
        <router-link to="/home" class="brand" aria-label="烂番茄首页">
          <span class="brand-seal">🍅</span>
          <span class="brand-text">
            <span class="brand-name">烂番茄</span>
            <span class="brand-sub">LANFANQIE · 毒舌创作质检台</span>
          </span>
        </router-link>

        <div class="nav-actions">
          <a href="#report" class="nav-link">烂度报告</a>
          <a href="#features" class="nav-link">创作台</a>
          <router-link to="/purchase" class="nav-link">Token 套餐</router-link>
          <LocaleSwitcher class="hidden sm:block" />
          <button class="icon-btn" :title="isDark ? '浅色' : '深色'" @click="toggleTheme">
            <Icon v-if="isDark" name="sun" size="sm" />
            <Icon v-else name="moon" size="sm" />
          </button>
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="btn-login">
            {{ isAuthenticated ? '进入控制台' : '登录' }}
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <!-- HERO -->
      <section class="hero">
        <div class="hero-glow" aria-hidden="true"></div>
        <div class="hero-inner">
          <div class="hero-copy fade-up">
            <p class="kicker">专治三烂 · 烂梗 / 烂节奏 / 烂套路</p>
            <h1 class="hero-title">
              你的稿子<span class="hl">烂</span>在哪，<br />
              烂番茄一眼挑出来。
            </h1>
            <p class="hero-sub">
              别等读者划走才发现。把章节丢进来，先做一次毒舌烂度体检——套路、注水、纸片人、假反转全给你标红，再告诉你怎么改成能卖的。
            </p>
            <div class="hero-actions">
              <router-link :to="studioPath" class="cta">开始拆书质检 <span aria-hidden="true">→</span></router-link>
              <router-link to="/studio/cover" class="cta-ghost">生成小说封面</router-link>
            </div>
            <ul class="rot-tags">
              <li>🩸 烂梗雷达</li>
              <li>🦠 节奏尸检</li>
              <li>🔥 爆款复盘</li>
            </ul>
          </div>

          <!-- 烂番茄主视觉 -->
          <div class="rot-stage fade-up delay-1">
            <div class="rot-orbit orbit-a" aria-hidden="true"></div>
            <div class="rot-orbit orbit-b" aria-hidden="true"></div>
            <div class="rot-splat" aria-hidden="true"></div>
            <div class="tomato floaty">
              <span class="t-leaf" aria-hidden="true"></span>
              <span class="t-shine" aria-hidden="true"></span>
              <span class="t-spot s1" aria-hidden="true"></span>
              <span class="t-spot s2" aria-hidden="true"></span>
              <span class="t-spot s3" aria-hidden="true"></span>
              <span class="t-mold" aria-hidden="true"></span>
              <span class="t-crack" aria-hidden="true"></span>
              <span class="t-drip" aria-hidden="true"></span>
            </div>
            <div class="rot-stamp">烂<small>ROTTEN</small></div>
            <span class="rot-fly f1" aria-hidden="true">🪰</span>
            <span class="rot-fly f2" aria-hidden="true">🪰</span>
          </div>
        </div>
      </section>

      <!-- 烂度报告 showcase -->
      <section id="report" class="report-band">
        <div class="band-inner report-grid">
          <div class="report-copy fade-up">
            <p class="eyebrow">烂度报告 · ROTTEN REPORT</p>
            <h2 class="band-title">不哄你写得好，<br />专挑你写得烂。</h2>
            <p class="band-desc">每次拆书都吐一份毒舌诊断：综合分、维度评分、烂点清单和能直接动刀的改写建议。烂得明明白白，改得清清楚楚。</p>
            <router-link :to="studioPath" class="cta small">丢一章试试 <span aria-hidden="true">→</span></router-link>
          </div>

          <div class="report-card fade-up delay-1">
            <div class="rc-head">
              <div>
                <span class="rc-eyebrow">爆款潜力分</span>
                <strong class="rc-score">{{ rottenReport.score }}<small>/100</small></strong>
              </div>
              <span class="rc-stamp">烂</span>
            </div>
            <p class="rc-verdict">“{{ rottenReport.verdict }}”</p>
            <div class="rc-bars">
              <div v-for="b in rottenReport.bars" :key="b.label" class="rc-bar">
                <span class="rc-bar-label">{{ b.label }}</span>
                <span class="rc-bar-track"><span class="rc-bar-fill" :style="{ width: b.value + '%' }"></span></span>
                <span class="rc-bar-val">{{ b.value }}</span>
              </div>
            </div>
            <div class="rc-rotten">
              <span class="rc-rotten-title">🍅 烂点</span>
              <ul><li v-for="(r, i) in rottenReport.rotten" :key="i">{{ r }}</li></ul>
            </div>
          </div>
        </div>
      </section>

      <!-- 创作台入口 -->
      <section id="features" class="section">
        <div class="band-inner">
          <div class="section-head fade-up">
            <p class="eyebrow">创作台入口</p>
            <h2 class="band-title">不是陪你温柔写稿，<br />是把能卖的部分挑出来。</h2>
          </div>
          <div class="feature-grid">
            <router-link
              v-for="(f, i) in features"
              :key="f.title"
              :to="f.to"
              class="feature-card fade-up"
              :style="{ animationDelay: 0.05 * i + 's' }"
            >
              <span class="f-index">{{ f.index }}</span>
              <span class="f-tag">{{ f.tag }}</span>
              <h3>{{ f.title }}</h3>
              <p>{{ f.desc }}</p>
              <span class="f-go" aria-hidden="true">→</span>
            </router-link>
          </div>
        </div>
      </section>

      <!-- 流程 -->
      <section class="section alt">
        <div class="band-inner flow-grid">
          <div class="section-head sticky fade-up">
            <p class="eyebrow">烂稿化验流程</p>
            <h2 class="band-title">丢进去的是草稿，<br />吐出来的是可执行清单。</h2>
          </div>
          <div class="flow-stack">
            <article v-for="(s, i) in workflow" :key="s.title" class="flow-card fade-up" :style="{ animationDelay: 0.06 * i + 's' }">
              <span class="flow-step">{{ s.step }}</span>
              <div>
                <h3>{{ s.title }}</h3>
                <p>{{ s.desc }}</p>
              </div>
            </article>
          </div>
        </div>
      </section>

      <!-- Token -->
      <section class="section">
        <div class="band-inner">
          <div class="conversion fade-up">
            <div>
              <p class="eyebrow">Token 套餐</p>
              <h2 class="band-title">拆书、封面、剧本、质检，<br />一套额度跑完整条生产线。</h2>
              <p class="band-desc">底层支付和额度沿用现有系统，前台只把它包装成作者能理解的创作燃料。</p>
            </div>
            <div class="conversion-actions">
              <router-link to="/purchase" class="cta">查看 Token 套餐 <span aria-hidden="true">→</span></router-link>
              <router-link :to="isAuthenticated ? studioPath : '/register'" class="cta-ghost">
                {{ isAuthenticated ? '进入创作台' : '免费注册' }}
              </router-link>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="site-footer">
      <div class="band-inner footer-inner">
        <p>© {{ currentYear }} 烂番茄 · 毒舌 AI 创作质检台</p>
        <div class="footer-links">
          <router-link to="/studio/teardown">拆书质检</router-link>
          <router-link to="/studio/cover">封面生成</router-link>
          <router-link to="/purchase">Token 套餐</router-link>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">文档</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const authStore = useAuthStore()
const appStore = useAppStore()

const isDark = ref(document.documentElement.classList.contains('dark'))

const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const studioPath = computed(() => (isAuthenticated.value ? '/studio/teardown' : '/login?redirect=/studio/teardown'))
const currentYear = computed(() => new Date().getFullYear())

const rottenReport = {
  score: 18,
  verdict: '套路堆叠的签到流，爽点像 PPT，配角全是纸片人。',
  bars: [
    { label: '套路新鲜度', value: 6 },
    { label: '人物', value: 9 },
    { label: '节奏', value: 22 },
    { label: '爽点密度', value: 41 },
  ],
  rotten: ['签到系统毫无代价，张力直接归零', '配角是工具纸片人，没有动机', '中段大量注水，三段重复同一桥段', '关键反转靠巧合，读者不买账'],
}

const features = [
  { index: '01', tag: 'COVER', title: '小说封面生成', desc: '按书名、题材和卖点生成封面方向，第一眼先替你抢点击。', to: '/studio/cover' },
  { index: '02', tag: 'TEARDOWN', title: '拆书 & 爆款分析', desc: '拆人物、爽点、伏笔和章节结构，把一坨稿子切成可复用模块。', to: '/studio/teardown' },
  { index: '03', tag: 'BACKUP', title: '番茄小说下载器', desc: '面向个人作品备份与素材整理，下载后可继续喂给拆书与剧本。', to: '/studio/downloader' },
  { index: '04', tag: 'WORKS', title: '我的作品', desc: '封面、报告、剧本一处归档，每次生成都能随时回看。', to: '/studio/works' },
  { index: '05', tag: 'SCRIPT', title: '剧本生成', desc: '把小说片段转成短剧脚本、分镜和口播素材，一稿多吃。', to: '/studio/script' },
  { index: '06', tag: 'TOKEN', title: 'Token 套餐', desc: '把底层额度包装成作者的创作燃料，按需购买、按量消耗。', to: '/purchase' },
]

const workflow = [
  { step: 'A', title: '丢稿', desc: '输入章节、书名、简介或榜单样本，先不急着美化。' },
  { step: 'B', title: '验烂', desc: '烂梗、拖沓、反转疲劳、设定断裂会被直接标红。' },
  { step: 'C', title: '改爆', desc: '输出封面方向、开篇钩子、爽点补强和短剧化建议。' },
]

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
</script>

<style scoped>
.lanfanqie-home {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  background:
    radial-gradient(900px 500px at 78% -8%, rgba(232, 65, 46, 0.22), transparent 60%),
    radial-gradient(700px 500px at 0% 20%, rgba(90, 107, 52, 0.12), transparent 55%),
    #140f0d;
  color: #fff5e8;
  font-family: "Noto Sans SC", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif;
}

.grain {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  opacity: 0.5;
  background-image: radial-gradient(rgba(255, 255, 255, 0.025) 1px, transparent 1.4px);
  background-size: 4px 4px;
}

.band-inner {
  margin: 0 auto;
  max-width: 1180px;
  padding-left: 1.25rem;
  padding-right: 1.25rem;
}

/* ---------- header ---------- */
.site-header {
  position: sticky;
  top: 0;
  z-index: 40;
  border-bottom: 1px solid rgba(255, 245, 232, 0.08);
  background: rgba(20, 15, 13, 0.78);
  backdrop-filter: blur(14px);
}

.nav-inner {
  margin: 0 auto;
  max-width: 1180px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.85rem 1.25rem;
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.7rem;
}

.brand-seal {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  font-size: 22px;
  border-radius: 12px;
  background: rgba(232, 65, 46, 0.18);
  border: 1px solid rgba(232, 65, 46, 0.4);
  box-shadow: 4px 4px 0 rgba(232, 65, 46, 0.16);
}

.brand-name {
  display: block;
  font-size: 17px;
  font-weight: 900;
  letter-spacing: 0.04em;
}

.brand-sub {
  display: block;
  font-size: 10px;
  letter-spacing: 0.22em;
  color: #cf9f86;
}

.nav-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.nav-link {
  display: none;
  padding: 0.5rem 0.7rem;
  font-size: 14px;
  font-weight: 600;
  color: #d8b7a2;
  transition: color 0.15s ease;
}

.nav-link:hover {
  color: #ff7a5e;
}

@media (min-width: 860px) {
  .nav-link {
    display: inline-flex;
  }
}

.icon-btn {
  display: grid;
  height: 38px;
  width: 38px;
  place-items: center;
  border-radius: 10px;
  border: 1px solid rgba(255, 245, 232, 0.12);
  background: rgba(255, 245, 232, 0.04);
  color: #e9c9b6;
  transition: all 0.15s ease;
}

.icon-btn:hover {
  border-color: rgba(232, 65, 46, 0.45);
  color: #ff7a5e;
}

.btn-login {
  display: inline-flex;
  align-items: center;
  border-radius: 10px;
  background: #e8412e;
  padding: 0.55rem 1rem;
  font-size: 14px;
  font-weight: 800;
  color: #fff6f1;
  box-shadow: 4px 4px 0 rgba(232, 65, 46, 0.22);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.btn-login:hover {
  transform: translate(-2px, -2px);
  box-shadow: 7px 7px 0 rgba(232, 65, 46, 0.26);
}

/* ---------- hero ---------- */
.hero {
  position: relative;
  z-index: 1;
}

.hero-glow {
  position: absolute;
  top: -120px;
  right: -80px;
  width: 520px;
  height: 520px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(232, 65, 46, 0.4), transparent 65%);
  filter: blur(20px);
  pointer-events: none;
}

.hero-inner {
  position: relative;
  z-index: 1;
  margin: 0 auto;
  max-width: 1180px;
  display: grid;
  gap: 2.5rem;
  align-items: center;
  padding: 3.5rem 1.25rem 4rem;
}

@media (min-width: 980px) {
  .hero-inner {
    grid-template-columns: 1.05fr 0.95fr;
    padding: 5rem 1.25rem 5.5rem;
  }
}

.kicker {
  display: inline-flex;
  border: 1px solid rgba(232, 65, 46, 0.4);
  border-radius: 999px;
  background: rgba(232, 65, 46, 0.12);
  padding: 6px 14px;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: #ff9d8c;
}

.hero-title {
  margin-top: 1.2rem;
  font-size: clamp(2.4rem, 6vw, 4.4rem);
  font-weight: 950;
  line-height: 1.02;
  letter-spacing: -0.01em;
}

.hero-title .hl {
  position: relative;
  color: #ff5436;
  text-shadow: 0 4px 24px rgba(232, 65, 46, 0.4);
}

.hero-title .hl::after {
  content: "";
  position: absolute;
  left: -2px;
  right: -2px;
  bottom: 6px;
  height: 8px;
  background: rgba(232, 65, 46, 0.35);
  border-radius: 2px;
  transform: rotate(-1.5deg);
}

.hero-sub {
  margin-top: 1.3rem;
  max-width: 32rem;
  font-size: 15.5px;
  line-height: 1.85;
  color: #d8b7a2;
}

.hero-actions {
  margin-top: 1.8rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.cta,
.cta-ghost {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: 12px;
  padding: 0.85rem 1.4rem;
  font-size: 15px;
  font-weight: 800;
  transition: transform 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
}

.cta {
  background: #e8412e;
  color: #fff6f1;
  box-shadow: 5px 5px 0 rgba(232, 65, 46, 0.22);
}

.cta:hover {
  transform: translate(-2px, -2px);
  box-shadow: 8px 8px 0 rgba(232, 65, 46, 0.28);
}

.cta.small {
  padding: 0.65rem 1.1rem;
  font-size: 14px;
}

.cta-ghost {
  border: 1px solid rgba(255, 245, 232, 0.18);
  color: #f3dccd;
}

.cta-ghost:hover {
  border-color: rgba(232, 65, 46, 0.5);
  color: #ff7a5e;
}

.rot-tags {
  margin-top: 2rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem 1.4rem;
  font-size: 13px;
  font-weight: 700;
  color: #c6a48f;
}

/* ---------- rotten tomato ---------- */
.rot-stage {
  position: relative;
  display: grid;
  place-items: center;
  min-height: 380px;
}

.rot-orbit {
  position: absolute;
  border: 1px dashed rgba(232, 65, 46, 0.3);
  border-radius: 50%;
  animation: spin 30s linear infinite;
}

.orbit-a {
  width: 430px;
  height: 430px;
}

.orbit-b {
  width: 320px;
  height: 320px;
  border-color: rgba(138, 154, 82, 0.28);
  animation-direction: reverse;
  animation-duration: 22s;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.rot-splat {
  position: absolute;
  bottom: 4%;
  left: 50%;
  width: 260px;
  height: 54px;
  transform: translateX(-50%);
  border-radius: 50%;
  background: radial-gradient(ellipse at center, rgba(110, 28, 14, 0.7), rgba(110, 28, 14, 0.2) 60%, transparent 76%);
  filter: blur(2px);
}

.tomato {
  position: relative;
  z-index: 2;
  width: 250px;
  height: 230px;
  border-radius: 52% 48% 45% 55% / 57% 55% 45% 43%;
  background:
    radial-gradient(circle at 70% 64%, rgba(96, 104, 46, 0.5), transparent 32%),
    radial-gradient(circle at 28% 70%, rgba(56, 38, 14, 0.7), transparent 40%),
    radial-gradient(circle at 36% 28%, #c2502f 0%, #93301c 44%, #5a1a0e 100%);
  box-shadow:
    inset -26px -32px 56px rgba(28, 6, 2, 0.72),
    inset 16px 18px 30px rgba(214, 150, 120, 0.18),
    inset 0 0 46px rgba(22, 6, 3, 0.5),
    0 30px 60px rgba(40, 10, 6, 0.55);
}

.floaty {
  animation: floaty 4.5s ease-in-out infinite;
}

@keyframes floaty {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-12px); }
}

.t-leaf {
  position: absolute;
  top: -16px;
  left: 50%;
  width: 70px;
  height: 48px;
  transform: translateX(-50%) rotate(8deg);
  background: linear-gradient(160deg, #5b6b34, #2f3c1b);
  clip-path: polygon(50% 0, 64% 34%, 100% 38%, 72% 56%, 84% 100%, 50% 70%, 16% 100%, 28% 56%, 0 38%, 36% 34%);
  filter: saturate(0.75);
}

.t-shine {
  position: absolute;
  top: 16%;
  left: 22%;
  width: 64px;
  height: 38px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 224, 206, 0.4), transparent 70%);
  transform: rotate(-18deg);
}

.t-spot {
  position: absolute;
  border-radius: 48% 52% 50% 50% / 52% 48% 52% 48%;
  background: radial-gradient(circle at 42% 40%, rgba(12, 4, 1, 0.92), rgba(46, 18, 8, 0.5) 54%, transparent 76%);
  box-shadow: inset 3px 4px 7px rgba(0, 0, 0, 0.6);
}

.s1 { width: 84px; height: 70px; top: 44%; left: 14%; }
.s2 { width: 56px; height: 50px; top: 20%; right: 16%; transform: rotate(18deg); }
.s3 { width: 50px; height: 44px; top: 62%; right: 28%; transform: rotate(-12deg); }

.t-mold {
  position: absolute;
  bottom: 24%;
  right: 20%;
  width: 60px;
  height: 54px;
  border-radius: 50%;
  background:
    radial-gradient(circle at 30% 30%, rgba(176, 186, 138, 0.92), transparent 56%),
    radial-gradient(circle at 66% 62%, rgba(132, 148, 100, 0.85), transparent 60%),
    radial-gradient(circle, rgba(96, 108, 70, 0.7), transparent 72%);
}

.t-mold::after {
  content: "";
  position: absolute;
  inset: 0;
  border-radius: 50%;
  background-image: radial-gradient(rgba(40, 50, 26, 0.7) 1.2px, transparent 1.6px);
  background-size: 7px 7px;
  opacity: 0.6;
}

.t-crack {
  position: absolute;
  top: 28%;
  left: 52%;
  width: 4px;
  height: 90px;
  transform: rotate(15deg);
  border-radius: 3px;
  background: linear-gradient(180deg, rgba(36, 10, 4, 0) 0%, rgba(36, 10, 4, 0.85) 32%, rgba(36, 10, 4, 0.5) 100%);
  box-shadow: -3px 12px 0 -1px rgba(36, 10, 4, 0.4);
}

.t-drip {
  position: absolute;
  bottom: -10px;
  left: 39%;
  width: 16px;
  height: 30px;
  background: #7a1f12;
  border-radius: 0 0 50% 50%;
}

.t-drip::after {
  content: "";
  position: absolute;
  top: -7px;
  left: 2px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #7a1f12;
}

.rot-stamp {
  position: absolute;
  top: 8%;
  left: 4%;
  z-index: 3;
  display: grid;
  place-items: center;
  width: 84px;
  height: 84px;
  border-radius: 50%;
  border: 3px solid rgba(255, 84, 54, 0.85);
  color: #ff5436;
  font-size: 38px;
  font-weight: 950;
  transform: rotate(-14deg);
  background: rgba(20, 15, 13, 0.6);
  box-shadow: 0 0 24px rgba(232, 65, 46, 0.3);
}

.rot-stamp small {
  position: absolute;
  bottom: 8px;
  font-size: 9px;
  letter-spacing: 0.18em;
}

.rot-fly {
  position: absolute;
  z-index: 4;
  font-size: 18px;
}

.f1 {
  top: 22%;
  right: 18%;
  animation: buzz 6s ease-in-out infinite;
}

.f2 {
  bottom: 28%;
  left: 16%;
  font-size: 14px;
  animation: buzz2 7s ease-in-out infinite;
}

@keyframes buzz {
  0%, 100% { transform: translate(0, 0) rotate(0); }
  30% { transform: translate(-26px, -16px) rotate(12deg); }
  60% { transform: translate(14px, -30px) rotate(-8deg); }
}

@keyframes buzz2 {
  0%, 100% { transform: translate(0, 0); }
  40% { transform: translate(22px, -14px) rotate(-10deg); }
  70% { transform: translate(-14px, 12px) rotate(8deg); }
}

/* ---------- bands ---------- */
.section {
  position: relative;
  z-index: 1;
  border-top: 1px solid rgba(255, 245, 232, 0.07);
  padding: 4.5rem 0;
}

.section.alt {
  background: rgba(0, 0, 0, 0.22);
}

.report-band {
  position: relative;
  z-index: 1;
  border-top: 1px solid rgba(255, 245, 232, 0.07);
  background: rgba(0, 0, 0, 0.25);
  padding: 4.5rem 0;
}

.eyebrow {
  font-size: 12px;
  font-weight: 900;
  letter-spacing: 0.2em;
  color: #ff7a5e;
}

.band-title {
  margin-top: 0.7rem;
  font-size: clamp(1.7rem, 4vw, 3rem);
  font-weight: 950;
  line-height: 1.1;
}

.band-desc {
  margin-top: 1rem;
  max-width: 34rem;
  font-size: 15px;
  line-height: 1.85;
  color: #c6a48f;
}

.report-grid,
.flow-grid {
  display: grid;
  gap: 2.5rem;
}

@media (min-width: 980px) {
  .report-grid {
    grid-template-columns: 0.9fr 1.1fr;
    align-items: center;
  }
  .flow-grid {
    grid-template-columns: 0.85fr 1.15fr;
  }
}

.report-copy .cta {
  margin-top: 1.6rem;
}

/* report card */
.report-card {
  border: 1px solid rgba(255, 245, 232, 0.12);
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(34, 22, 18, 0.9), rgba(20, 15, 13, 0.9));
  padding: 1.5rem;
  box-shadow: 0 30px 70px rgba(0, 0, 0, 0.4);
}

.rc-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.rc-eyebrow {
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
  color: #c6a48f;
}

.rc-score {
  display: block;
  margin-top: 0.2rem;
  font-size: 56px;
  font-weight: 950;
  line-height: 1;
  color: #ff5436;
}

.rc-score small {
  font-size: 18px;
  color: #8a6b5f;
}

.rc-stamp {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  border: 2px solid rgba(255, 84, 54, 0.8);
  color: #ff5436;
  font-size: 24px;
  font-weight: 950;
  transform: rotate(-12deg);
}

.rc-verdict {
  margin-top: 0.9rem;
  font-size: 15px;
  font-weight: 700;
  color: #f3dccd;
}

.rc-bars {
  margin-top: 1.1rem;
  display: grid;
  gap: 0.55rem;
}

.rc-bar {
  display: grid;
  grid-template-columns: 76px 1fr 28px;
  align-items: center;
  gap: 0.6rem;
  font-size: 13px;
  color: #c6a48f;
}

.rc-bar-track {
  height: 8px;
  border-radius: 999px;
  background: rgba(255, 245, 232, 0.08);
  overflow: hidden;
}

.rc-bar-fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, #e8412e, #ff7a5e);
}

.rc-bar-val {
  text-align: right;
  font-weight: 800;
  color: #f3dccd;
}

.rc-rotten {
  margin-top: 1.2rem;
  border-top: 1px solid rgba(255, 245, 232, 0.08);
  padding-top: 0.9rem;
}

.rc-rotten-title {
  font-size: 13px;
  font-weight: 900;
  color: #ff9d8c;
}

.rc-rotten ul {
  margin-top: 0.5rem;
  display: grid;
  gap: 0.35rem;
  padding-left: 1.1rem;
  list-style: disc;
  font-size: 13px;
  line-height: 1.7;
  color: #d8b7a2;
}

/* features */
.section-head.sticky {
  align-self: start;
}

.feature-grid {
  margin-top: 2.2rem;
  display: grid;
  gap: 0.85rem;
  grid-template-columns: 1fr;
}

@media (min-width: 680px) {
  .feature-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 980px) {
  .feature-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

.feature-card {
  position: relative;
  display: block;
  border: 1px solid rgba(255, 245, 232, 0.1);
  border-radius: 16px;
  background: rgba(255, 245, 232, 0.03);
  padding: 1.4rem;
  min-height: 184px;
  transition: transform 0.18s ease, border-color 0.18s ease, background 0.18s ease;
}

.feature-card:hover {
  transform: translateY(-4px);
  border-color: rgba(232, 65, 46, 0.5);
  background: rgba(232, 65, 46, 0.07);
}

.f-index {
  font-size: 13px;
  font-weight: 900;
  color: rgba(232, 65, 46, 0.6);
}

.f-tag {
  position: absolute;
  top: 1.4rem;
  right: 1.4rem;
  font-size: 10px;
  font-weight: 900;
  letter-spacing: 0.14em;
  color: #8a6b5f;
}

.feature-card h3 {
  margin-top: 1rem;
  font-size: 19px;
  font-weight: 900;
  color: #fff5e8;
}

.feature-card p {
  margin-top: 0.6rem;
  font-size: 13.5px;
  line-height: 1.75;
  color: #c6a48f;
}

.f-go {
  position: absolute;
  bottom: 1.2rem;
  right: 1.4rem;
  font-size: 18px;
  color: #ff7a5e;
  opacity: 0;
  transform: translateX(-6px);
  transition: all 0.18s ease;
}

.feature-card:hover .f-go {
  opacity: 1;
  transform: translateX(0);
}

/* flow */
.flow-stack {
  display: grid;
  gap: 0.85rem;
}

.flow-card {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  border: 1px solid rgba(255, 245, 232, 0.1);
  border-radius: 16px;
  background: rgba(255, 245, 232, 0.03);
  padding: 1.3rem;
}

.flow-step {
  display: grid;
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  place-items: center;
  border-radius: 12px;
  background: #e8412e;
  color: #fff6f1;
  font-weight: 950;
}

.flow-card h3 {
  font-size: 17px;
  font-weight: 900;
}

.flow-card p {
  margin-top: 0.35rem;
  font-size: 13.5px;
  line-height: 1.75;
  color: #c6a48f;
}

/* conversion */
.conversion {
  display: grid;
  gap: 1.6rem;
  border: 1px solid rgba(232, 65, 46, 0.3);
  border-radius: 20px;
  background: linear-gradient(135deg, rgba(232, 65, 46, 0.14), rgba(20, 15, 13, 0.5));
  padding: clamp(1.6rem, 4vw, 3rem);
}

@media (min-width: 980px) {
  .conversion {
    grid-template-columns: 1.4fr 0.6fr;
    align-items: center;
  }
}

.conversion-actions {
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
}

/* footer */
.site-footer {
  position: relative;
  z-index: 1;
  border-top: 1px solid rgba(255, 245, 232, 0.08);
  padding: 2.2rem 0;
}

.footer-inner {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  font-size: 13.5px;
  color: #a98b78;
}

@media (min-width: 680px) {
  .footer-inner {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
}

.footer-links {
  display: flex;
  flex-wrap: wrap;
  gap: 1.2rem;
}

.footer-links a {
  color: #c6a48f;
  transition: color 0.15s ease;
}

.footer-links a:hover {
  color: #ff7a5e;
}

/* ---------- entrance ---------- */
.fade-up {
  animation: fadeUp 0.7s cubic-bezier(0.2, 0.7, 0.2, 1) both;
}

.fade-up.delay-1 {
  animation-delay: 0.12s;
}

@keyframes fadeUp {
  from {
    opacity: 0;
    transform: translateY(22px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .fade-up,
  .floaty,
  .rot-orbit,
  .rot-fly {
    animation: none;
  }
}
</style>
