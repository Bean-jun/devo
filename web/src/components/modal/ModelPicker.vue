<template>
  <div
    v-if="isOpen"
    class="model-picker-overlay"
    data-test="model-picker-overlay"
    @keydown="handleKeydown"
    @click.self="close"
  >
    <div class="model-picker" data-test="model-picker">
      <div class="model-picker-header">
        <h2>选择模型</h2>
        <button class="model-picker-close" @click="close" data-test="model-picker-close">&times;</button>
      </div>
      <div class="model-picker-body">
        <div v-if="isLoading" class="model-picker-loading">加载中...</div>
        <div v-else-if="models.length === 0" class="model-picker-empty">
          <p>暂无模型配置</p>
          <p class="hint">请在下方添加模型，或前往设置页面管理</p>
          <button v-if="!showAddForm" class="model-picker-add-btn" @click="showAddForm = true">+ 添加模型</button>
          <AddModelForm
            v-else
            mode="inline"
            @submit="showAddForm = false"
            @cancel="showAddForm = false"
          />
        </div>
        <div v-else class="model-list">
          <div
            v-for="m in models"
            :key="m.id"
            class="model-item"
            :class="{ active: m.id === activeModelId }"
            :data-test="`model-item-${m.id}`"
            @click="selectModel(m.id)"
          >
            <div class="model-item-info">
              <span class="model-item-name">{{ m.name }}</span>
              <span class="model-item-model">{{ m.model }}</span>
            </div>
            <div v-if="m.id === activeModelId" class="model-item-badge">当前</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useUiStore } from '@/stores/ui'
import { useModelStore } from '@/stores/model'
import AddModelForm from '@/components/common/AddModelForm.vue'

const uiStore = useUiStore()
const modelStore = useModelStore()

const isOpen = computed(() => uiStore.activeModal === 'model-picker')
const models = computed(() => modelStore.models)
const activeModelId = computed(() => modelStore.activeModelId)
const isLoading = computed(() => modelStore.isLoading)
const switching = ref<string | null>(null)

const showAddForm = ref(false)

onMounted(() => {
  modelStore.fetchModels()
})

async function selectModel(id: string) {
  if (id === activeModelId.value || switching.value) return
  switching.value = id
  try {
    await modelStore.activateModel(id)
    uiStore.showToast('success', `已切换到 ${modelStore.models.find(m => m.id === id)?.name ?? id}`)
    close()
  } catch (e: any) {
    uiStore.showToast('error', e.message || '切换失败')
  } finally {
    switching.value = null
  }
}

function close() {
  uiStore.setActiveModal(null)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    close()
  }
}
</script>

<style scoped>
.model-picker-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.5);
}

.model-picker {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  width: 400px;
  max-width: 90vw;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.model-picker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border);
}

.model-picker-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.model-picker-close {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--color-text-secondary);
  padding: 4px;
  line-height: 1;
}

.model-picker-close:hover {
  color: var(--color-text-primary);
}

.model-picker-body {
  padding: 12px;
  overflow-y: auto;
  flex: 1;
}

.model-picker-loading,
.model-picker-empty {
  text-align: center;
  padding: 32px 16px;
  color: var(--color-text-secondary);
}

.model-picker-empty .hint {
  font-size: 13px;
  margin-top: 8px;
  opacity: 0.7;
}

.model-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.model-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
}

.model-item:hover {
  background: var(--color-bg-hover);
}

.model-item.active {
  background: var(--color-bg-selected);
  border: 1px solid var(--color-accent);
}

.model-item-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.model-item-name {
  font-size: 14px;
  font-weight: 500;
}

.model-item-model {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.model-item-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--color-accent);
  color: white;
}

.model-picker-add-btn {
  margin-top: 12px;
  padding: 8px 20px;
  border: 1px solid var(--color-accent);
  border-radius: 6px;
  background: transparent;
  color: var(--color-accent);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}

.model-picker-add-btn:hover {
  background: var(--color-accent);
  color: white;
}
</style>