<template>
  <Transition name="cover-viewer" appear>
    <div v-if="activeImage" class="cover-viewer" data-test="cover-viewer" @click.self="emitClose">
      <div class="cover-viewer-panel">
        <div class="cover-viewer-topbar">
          <div class="viewer-meta">
            <span data-test="cover-viewer-counter">{{ activeIndex + 1 }} / {{ images.length }}</span>
            <span>{{ activeImage.id || title || 'cover' }}</span>
          </div>
          <button type="button" class="viewer-icon-btn" data-test="cover-close" title="关闭" aria-label="关闭大图查看" @click="emitClose">
            <Icon name="x" size="md" :stroke-width="2" />
          </button>
        </div>

        <button
          v-if="hasMultipleImages"
          type="button"
          class="viewer-nav-btn viewer-nav-prev"
          data-test="cover-prev"
          title="上一张"
          aria-label="上一张封面"
          @click="showImage(-1)"
        >
          <Icon name="chevronLeft" size="lg" :stroke-width="2.2" />
        </button>
        <button
          v-if="hasMultipleImages"
          type="button"
          class="viewer-nav-btn viewer-nav-next"
          data-test="cover-next"
          title="下一张"
          aria-label="下一张封面"
          @click="showImage(1)"
        >
          <Icon name="chevronRight" size="lg" :stroke-width="2.2" />
        </button>

        <div
          class="cover-viewer-stage"
          data-test="cover-viewer-stage"
          @mousedown="startDrag"
          @wheel="zoomWithWheel"
        >
          <img
            :key="activeImage.id || activeImage.url"
            data-test="cover-viewer-image"
            :src="activeImage.url"
            :alt="title || 'cover preview'"
            class="cover-viewer-image"
            :class="{ dragging }"
            :style="imageStyle"
            draggable="false"
          />
        </div>

        <div class="cover-viewer-toolbar">
          <a
            class="viewer-tool-btn viewer-download"
            data-test="cover-download"
            :href="activeImage.url"
            :download="downloadName"
            title="下载"
            aria-label="下载当前封面"
          >
            <Icon name="download" size="sm" :stroke-width="2" />
            <span>下载</span>
          </a>
          <button type="button" class="viewer-tool-btn" data-test="cover-zoom-out" title="缩小" aria-label="缩小" @click="zoomImage(-0.25)">
            <Icon name="minus" size="sm" :stroke-width="2" />
          </button>
          <button type="button" class="viewer-tool-btn" data-test="cover-zoom-reset" title="重置" aria-label="重置缩放和位置" @click="resetTransform">
            <Icon name="refresh" size="sm" :stroke-width="2" />
          </button>
          <button type="button" class="viewer-tool-btn" data-test="cover-zoom-in" title="放大" aria-label="放大" @click="zoomImage(0.25)">
            <Icon name="plus" size="sm" :stroke-width="2" />
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'

interface ViewerImage {
  id?: string
  url: string
  prompt?: string
}

const props = withDefaults(defineProps<{
  images: ViewerImage[]
  initialIndex?: number
  title?: string
  downloadBaseName?: string
}>(), {
  initialIndex: 0,
  title: '',
  downloadBaseName: '',
})

const emit = defineEmits<{
  close: []
}>()

const activeIndex = ref(0)
const zoom = ref(1)
const offsetX = ref(0)
const offsetY = ref(0)
const dragging = ref(false)
let dragStartX = 0
let dragStartY = 0
let dragOriginX = 0
let dragOriginY = 0
let previousBodyOverflow = ''

const activeImage = computed(() => props.images[activeIndex.value] || null)
const hasMultipleImages = computed(() => props.images.length > 1)
const imageStyle = computed(() => ({
  transform: `translate3d(${offsetX.value}px, ${offsetY.value}px, 0) scale(${zoom.value})`,
}))
const downloadName = computed(() => {
  const rawName = props.downloadBaseName || props.title || activeImage.value?.id || 'cover'
  const base = rawName.trim().replace(/[\\/:*?"<>|]+/g, '-')
  return `${base || 'cover'}-${activeIndex.value + 1}.png`
})

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function setActiveIndex(index: number) {
  activeIndex.value = clamp(index, 0, Math.max(props.images.length - 1, 0))
  resetTransform()
}

function resetTransform() {
  zoom.value = 1
  offsetX.value = 0
  offsetY.value = 0
}

function zoomImage(delta: number) {
  zoom.value = Number(clamp(zoom.value + delta, 0.5, 4).toFixed(2))
}

function zoomWithWheel(event: WheelEvent) {
  event.preventDefault()
  zoomImage(event.deltaY < 0 ? 0.25 : -0.25)
}

function showImage(direction: number) {
  if (!props.images.length) return
  activeIndex.value = (activeIndex.value + direction + props.images.length) % props.images.length
  resetTransform()
}

function startDrag(event: MouseEvent) {
  if (!activeImage.value) return
  dragging.value = true
  dragStartX = event.clientX
  dragStartY = event.clientY
  dragOriginX = offsetX.value
  dragOriginY = offsetY.value
  window.addEventListener('mousemove', moveDrag)
  window.addEventListener('mouseup', stopDrag)
}

function moveDrag(event: MouseEvent) {
  if (!dragging.value) return
  offsetX.value = dragOriginX + event.clientX - dragStartX
  offsetY.value = dragOriginY + event.clientY - dragStartY
}

function stopDrag() {
  if (dragging.value) {
    dragging.value = false
  }
  window.removeEventListener('mousemove', moveDrag)
  window.removeEventListener('mouseup', stopDrag)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emitClose()
  if (event.key === 'ArrowLeft') showImage(-1)
  if (event.key === 'ArrowRight') showImage(1)
}

function emitClose() {
  stopDrag()
  emit('close')
}

watch(() => props.initialIndex, (index) => setActiveIndex(index))
watch(() => props.images.length, () => setActiveIndex(activeIndex.value))

onMounted(() => {
  previousBodyOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  setActiveIndex(props.initialIndex)
  window.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  stopDrag()
  window.removeEventListener('keydown', handleKeydown)
  document.body.style.overflow = previousBodyOverflow
})
</script>

<style scoped>
.cover-viewer {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(6, 8, 12, 0.88);
  padding: 18px;
  backdrop-filter: blur(14px);
}

.cover-viewer-panel {
  position: relative;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  width: min(1120px, 100%);
  height: min(880px, calc(100vh - 36px));
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 18px;
  background: linear-gradient(180deg, rgba(16, 18, 24, 0.96), rgba(7, 9, 14, 0.98));
  box-shadow: 0 30px 90px rgba(0, 0, 0, 0.42);
}

.cover-viewer-topbar {
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  color: #f9fafb;
}

.viewer-meta {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  font-weight: 800;
}

.viewer-meta span:last-child {
  max-width: 46vw;
  overflow: hidden;
  color: rgba(249, 250, 251, 0.58);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.viewer-icon-btn,
.viewer-nav-btn,
.viewer-tool-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(255, 255, 255, 0.14);
  color: #f9fafb;
  background: rgba(255, 255, 255, 0.09);
  transition: transform 0.16s ease, background 0.16s ease, border-color 0.16s ease;
}

.viewer-icon-btn:hover,
.viewer-nav-btn:hover,
.viewer-tool-btn:hover {
  border-color: rgba(255, 255, 255, 0.28);
  background: rgba(255, 255, 255, 0.16);
  transform: translateY(-1px);
}

.viewer-icon-btn {
  width: 40px;
  height: 40px;
  border-radius: 999px;
}

.cover-viewer-stage {
  position: relative;
  display: flex;
  min-height: 0;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  cursor: grab;
  background:
    linear-gradient(90deg, rgba(255, 255, 255, 0.045) 1px, transparent 1px),
    linear-gradient(0deg, rgba(255, 255, 255, 0.045) 1px, transparent 1px);
  background-size: 34px 34px;
}

.cover-viewer-stage:active {
  cursor: grabbing;
}

.cover-viewer-image {
  max-width: min(78vw, 760px);
  max-height: calc(100vh - 210px);
  user-select: none;
  border-radius: 10px;
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.52);
  transform-origin: center center;
  transition: transform 0.16s ease;
}

.cover-viewer-image.dragging {
  transition: none;
}

.viewer-nav-btn {
  position: absolute;
  top: 50%;
  z-index: 3;
  width: 48px;
  height: 64px;
  border-radius: 999px;
}

.viewer-nav-prev {
  left: 18px;
}

.viewer-nav-next {
  right: 18px;
}

.cover-viewer-toolbar {
  z-index: 2;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 14px 16px 16px;
}

.viewer-tool-btn {
  min-width: 42px;
  height: 42px;
  gap: 8px;
  border-radius: 999px;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 800;
  text-decoration: none;
}

.viewer-download {
  background: rgb(220 56 31);
  border-color: rgba(255, 255, 255, 0.2);
}

.viewer-download:hover {
  background: rgb(190 45 25);
}

.cover-viewer-enter-active,
.cover-viewer-leave-active {
  transition: opacity 0.22s ease;
}

.cover-viewer-enter-active .cover-viewer-panel,
.cover-viewer-leave-active .cover-viewer-panel {
  transition: transform 0.22s ease, opacity 0.22s ease;
}

.cover-viewer-enter-from,
.cover-viewer-leave-to {
  opacity: 0;
}

.cover-viewer-enter-from .cover-viewer-panel,
.cover-viewer-leave-to .cover-viewer-panel {
  opacity: 0;
  transform: translateY(14px) scale(0.96);
}

@media (max-width: 640px) {
  .cover-viewer {
    padding: 10px;
  }

  .cover-viewer-panel {
    height: calc(100vh - 20px);
    border-radius: 14px;
  }

  .viewer-nav-btn {
    width: 42px;
    height: 52px;
  }

  .viewer-nav-prev {
    left: 10px;
  }

  .viewer-nav-next {
    right: 10px;
  }

  .cover-viewer-image {
    max-width: 86vw;
    max-height: calc(100vh - 220px);
  }
}
</style>
