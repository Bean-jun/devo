<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import StatusBar from '@/components/layout/StatusBar.vue'
import RightPanel from '@/components/layout/RightPanel.vue'
import GlobalModals from '@/components/layout/GlobalModals.vue'

const sidebarWidth = ref(240)
const rightPanelWidth = ref(380)
const resizing = ref<'left' | 'right' | null>(null)
const leftWrapperRef = ref<HTMLElement | null>(null)
const rightWrapperRef = ref<HTMLElement | null>(null)
let rafId = 0

function startResize(type: 'left' | 'right') {
  resizing.value = type
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  if (type === 'left' && leftWrapperRef.value) {
    leftWrapperRef.value.style.willChange = 'width'
  } else if (type === 'right' && rightWrapperRef.value) {
    rightWrapperRef.value.style.willChange = 'width'
  }
}

function onMouseMove(e: MouseEvent) {
  if (!resizing.value) return
  cancelAnimationFrame(rafId)
  rafId = requestAnimationFrame(() => {
    if (resizing.value === 'left') {
      sidebarWidth.value = Math.max(180, Math.min(420, e.clientX))
    } else if (resizing.value === 'right') {
      rightPanelWidth.value = Math.max(220, Math.min(window.innerWidth * 0.75, window.innerWidth - e.clientX))
    }
  })
}

function onMouseUp() {
  if (resizing.value) {
    try { localStorage.setItem('devo-sidebar-width', String(sidebarWidth.value)) } catch {}
    try { localStorage.setItem('devo-rightpanel-width', String(rightPanelWidth.value)) } catch {}
  }
  if (leftWrapperRef.value) leftWrapperRef.value.style.willChange = ''
  if (rightWrapperRef.value) rightWrapperRef.value.style.willChange = ''
  resizing.value = null
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

onMounted(() => {
  try {
    const sl = localStorage.getItem('devo-sidebar-width')
    if (sl) sidebarWidth.value = parseInt(sl, 10)
    const sr = localStorage.getItem('devo-rightpanel-width')
    if (sr) rightPanelWidth.value = parseInt(sr, 10)
  } catch {}
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
})

onUnmounted(() => {
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
})
</script>

<template>
  <div class="browser-layout">
    <div ref="leftWrapperRef" class="sidebar-wrapper" :style="{ width: sidebarWidth + 'px' }">
      <AppSidebar />
    </div>
    <div class="resize-handle" @mousedown="startResize('left')">
      <div class="handle-hit"></div>
      <span class="handle-icon">║</span>
    </div>
    <div class="browser-main">
      <StatusBar />
      <div class="browser-content">
        <router-view />
      </div>
    </div>
    <div class="resize-handle" @mousedown="startResize('right')">
      <div class="handle-hit"></div>
      <span class="handle-icon">║</span>
    </div>
    <div ref="rightWrapperRef" class="right-wrapper" :style="{ width: rightPanelWidth + 'px' }">
      <RightPanel />
    </div>
    <GlobalModals />
  </div>
</template>

<style scoped>
.browser-layout {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background: var(--color-bg-primary);
}

.sidebar-wrapper,
.right-wrapper {
  flex-shrink: 0;
  height: 100vh;
  overflow: hidden;
}

.resize-handle {
  width: 6px;
  cursor: col-resize;
  background: var(--color-bg-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  position: relative;
}

.handle-hit {
  position: absolute;
  inset: 0 -4px;
  z-index: 0;
}

.handle-icon {
  font-size: 10px;
  color: var(--color-text-tertiary);
  line-height: 1;
  pointer-events: none;
  position: relative;
  z-index: 1;
}

.resize-handle:hover {
  background: var(--color-accent);
}

.resize-handle:hover .handle-icon {
  color: white;
}

.browser-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.browser-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  background: var(--color-bg-primary);
}
</style>