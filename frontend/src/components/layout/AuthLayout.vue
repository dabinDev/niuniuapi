<template>
  <div class="auth-shell relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <!-- Background -->
    <div class="auth-backdrop absolute inset-0"></div>

    <!-- Decorative Elements -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="auth-orb auth-orb-one"></div>
      <div class="auth-orb auth-orb-two"></div>
      <div class="auth-noise"></div>
      <div class="auth-ruling"></div>
    </div>

    <!-- Content Container -->
    <div class="auth-stage relative z-10 grid w-full max-w-6xl overflow-hidden rounded-[2rem] border border-white/70 bg-white/78 shadow-[0_30px_100px_rgba(69,31,20,0.18)] backdrop-blur-xl dark:border-white/10 dark:bg-dark-950/78">
      <section class="auth-brand-panel">
        <template v-if="settingsLoaded">
          <div class="auth-seal">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <p class="auth-kicker">LANFANQIE STUDIO</p>
          <h1>{{ siteName }}</h1>
          <p class="auth-subtitle">{{ siteSubtitle }}</p>
        </template>
        <div class="auth-proof-grid">
          <article>
            <strong>热榜雷达</strong>
            <span>快速抓取可对标题材样本</span>
          </article>
          <article>
            <strong>封面工坊</strong>
            <span>让书名、人物和卖点一起出图</span>
          </article>
          <article>
            <strong>毒舌质检</strong>
            <span>拆节奏、看钩子、补爽点</span>
          </article>
        </div>
      </section>

      <!-- Card Container -->
      <section class="auth-form-panel">
        <div class="auth-card">
          <slot />
        </div>

        <!-- Footer Links -->
        <div class="mt-6 text-center text-sm">
          <slot name="footer" />
        </div>

        <!-- Copyright -->
        <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || '烂番茄')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.text-gradient {
  @apply bg-gradient-to-r from-primary-600 to-primary-500 bg-clip-text text-transparent;
}

.auth-shell {
  font-family: "Noto Serif SC", "Source Han Serif SC", "Microsoft YaHei", sans-serif;
}

.auth-backdrop {
  background:
    radial-gradient(circle at 18% 18%, rgba(255, 196, 120, 0.32), transparent 27rem),
    radial-gradient(circle at 86% 16%, rgba(232, 65, 46, 0.22), transparent 24rem),
    linear-gradient(135deg, #fff7ec 0%, #fffdf8 43%, #f7e8d6 100%);
}

.dark .auth-backdrop {
  background:
    radial-gradient(circle at 18% 18%, rgba(232, 65, 46, 0.18), transparent 26rem),
    radial-gradient(circle at 86% 20%, rgba(255, 196, 120, 0.11), transparent 24rem),
    linear-gradient(135deg, #130f0c 0%, #1c1411 52%, #0d0a09 100%);
}

.auth-orb {
  position: absolute;
  border-radius: 999px;
  filter: blur(36px);
}

.auth-orb-one {
  width: 24rem;
  height: 24rem;
  right: -7rem;
  top: -7rem;
  background: rgba(232, 65, 46, 0.24);
}

.auth-orb-two {
  width: 20rem;
  height: 20rem;
  left: -6rem;
  bottom: -6rem;
  background: rgba(121, 71, 37, 0.18);
}

.auth-noise {
  position: absolute;
  inset: 0;
  opacity: 0.26;
  background-image: radial-gradient(rgba(83, 42, 26, 0.18) 1px, transparent 1px);
  background-size: 16px 16px;
}

.auth-ruling {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(232, 65, 46, 0.08) 1px, transparent 1px),
    linear-gradient(rgba(232, 65, 46, 0.06) 1px, transparent 1px);
  background-size: 76px 76px;
  mask-image: linear-gradient(120deg, transparent 0%, black 18%, black 72%, transparent 100%);
}

.auth-stage {
  grid-template-columns: minmax(0, 1fr) minmax(380px, 0.86fr);
}

.auth-brand-panel {
  position: relative;
  min-height: 42rem;
  padding: 3.2rem;
  color: #fff8ef;
  background:
    linear-gradient(145deg, rgba(41, 22, 16, 0.94), rgba(99, 36, 21, 0.9)),
    radial-gradient(circle at 18% 20%, rgba(255, 199, 136, 0.32), transparent 18rem);
}

.auth-brand-panel::after {
  content: "烂番茄";
  position: absolute;
  right: -1rem;
  bottom: 0.5rem;
  color: rgba(255, 244, 230, 0.055);
  font-size: clamp(5rem, 12vw, 11rem);
  font-weight: 950;
  line-height: 1;
  writing-mode: vertical-rl;
}

.auth-seal {
  display: inline-flex;
  width: 4.75rem;
  height: 4.75rem;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.28);
  border-radius: 1.45rem;
  background: rgba(255, 255, 255, 0.12);
  box-shadow: 0 20px 42px rgba(0, 0, 0, 0.24);
}

.auth-kicker {
  margin-top: 2.8rem;
  color: #ffc8a5;
  font-size: 0.78rem;
  font-weight: 950;
  letter-spacing: 0.18em;
}

.auth-brand-panel h1 {
  margin-top: 0.8rem;
  max-width: 22rem;
  font-size: clamp(2.8rem, 6vw, 5.4rem);
  font-weight: 950;
  letter-spacing: -0.07em;
  line-height: 0.92;
}

.auth-subtitle {
  margin-top: 1.2rem;
  max-width: 25rem;
  color: #f8d9c3;
  font-size: 1rem;
  line-height: 1.9;
}

.auth-proof-grid {
  position: relative;
  z-index: 1;
  display: grid;
  gap: 0.75rem;
  margin-top: 3.2rem;
  max-width: 25rem;
}

.auth-proof-grid article {
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 1rem;
  padding: 0.85rem 1rem;
  background: rgba(255, 255, 255, 0.08);
}

.auth-proof-grid strong,
.auth-proof-grid span {
  display: block;
}

.auth-proof-grid strong {
  font-weight: 950;
}

.auth-proof-grid span {
  margin-top: 0.2rem;
  color: #f5d4bf;
  font-size: 0.82rem;
}

.auth-form-panel {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 3rem;
}

.auth-card {
  border: 1px solid rgba(232, 65, 46, 0.14);
  border-radius: 1.4rem;
  background: rgba(255, 255, 255, 0.78);
  padding: 2rem;
  box-shadow: 0 24px 60px rgba(69, 31, 20, 0.1);
}

.dark .auth-card {
  border-color: rgba(255, 255, 255, 0.09);
  background: rgba(23, 19, 17, 0.82);
}

@media (max-width: 920px) {
  .auth-stage {
    grid-template-columns: 1fr;
    max-width: 32rem;
  }

  .auth-brand-panel {
    min-height: auto;
    padding: 2rem;
  }

  .auth-proof-grid {
    display: none;
  }

  .auth-form-panel {
    padding: 1rem;
  }
}
</style>
