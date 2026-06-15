<template>
  <AppLayout>
    <div class="studio-wrap">
      <div class="studio-card">
        <div class="studio-emoji" aria-hidden="true">{{ emoji }}</div>
        <span class="studio-badge" :class="priority === 'P0' ? 'badge-p0' : 'badge-p1'">{{ priority }} · 开发中</span>
        <h1 class="studio-title">{{ title }}</h1>
        <p class="studio-subtitle">{{ subtitle }}</p>
        <ul v-if="points.length" class="studio-points">
          <li v-for="point in points" :key="point">{{ point }}</li>
        </ul>
        <div class="studio-actions">
          <router-link to="/purchase" class="studio-cta">查看 Token 套餐</router-link>
          <router-link to="/dashboard" class="studio-cta-ghost">返回控制台</router-link>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'

defineProps<{
  emoji: string
  title: string
  subtitle: string
  priority: 'P0' | 'P1'
  points: string[]
}>()
</script>

<style scoped>
.studio-wrap {
  display: flex;
  min-height: 60vh;
  align-items: center;
  justify-content: center;
  padding: 2rem 1rem;
}

.studio-card {
  position: relative;
  width: 100%;
  max-width: 560px;
  border: 1px solid rgba(232, 65, 46, 0.18);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.7);
  padding: 2.5rem 2rem;
  text-align: center;
  box-shadow: 0 18px 44px rgba(120, 40, 24, 0.1);
}

.dark .studio-card {
  border-color: rgba(255, 138, 120, 0.22);
  background: rgba(255, 255, 255, 0.04);
  box-shadow: none;
}

.studio-emoji {
  font-size: 60px;
  line-height: 1;
}

.studio-badge {
  display: inline-block;
  margin-top: 1rem;
  border-radius: 999px;
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.badge-p0 {
  background: #ffe7e0;
  color: #c8351f;
}

.badge-p1 {
  background: #fff1d9;
  color: #a8631a;
}

.dark .badge-p0 {
  background: rgba(232, 65, 46, 0.18);
  color: #ff9d8c;
}

.dark .badge-p1 {
  background: rgba(200, 140, 30, 0.2);
  color: #f0c073;
}

.studio-title {
  margin-top: 1rem;
  font-size: clamp(1.6rem, 4vw, 2.4rem);
  font-weight: 900;
  color: #241a16;
}

.dark .studio-title {
  color: #f7ede4;
}

.studio-subtitle {
  margin-top: 0.75rem;
  font-size: 15px;
  line-height: 1.8;
  color: #83685c;
}

.dark .studio-subtitle {
  color: #b6a294;
}

.studio-points {
  margin: 1.25rem auto 0;
  max-width: 420px;
  display: grid;
  gap: 0.5rem;
  text-align: left;
}

.studio-points li {
  position: relative;
  padding-left: 1.4rem;
  font-size: 14px;
  line-height: 1.7;
  color: #5c463c;
}

.dark .studio-points li {
  color: #cdbcae;
}

.studio-points li::before {
  content: '🍅';
  position: absolute;
  left: 0;
  font-size: 12px;
}

.studio-actions {
  margin-top: 1.75rem;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.75rem;
}

.studio-cta,
.studio-cta-ghost {
  display: inline-flex;
  align-items: center;
  border-radius: 10px;
  padding: 10px 18px;
  font-size: 14px;
  font-weight: 800;
  transition: transform 0.18s ease, background 0.18s ease, color 0.18s ease;
}

.studio-cta {
  background: #e8412e;
  color: #fff6f1;
}

.studio-cta:hover {
  transform: translateY(-1px);
  background: #d3361f;
}

.studio-cta-ghost {
  border: 1px solid rgba(58, 28, 18, 0.16);
  color: #6b4d42;
}

.dark .studio-cta-ghost {
  border-color: rgba(255, 255, 255, 0.16);
  color: #d8c4b8;
}

.studio-cta-ghost:hover {
  border-color: rgba(232, 65, 46, 0.4);
  color: #e8412e;
}
</style>
