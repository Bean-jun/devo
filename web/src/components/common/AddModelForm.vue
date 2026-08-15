<template>
  <div v-if="mode === 'modal'" class="add-model-overlay" @click.self="$emit('cancel')">
    <div class="add-model-dialog">
      <h3 class="add-model-title">添加模型</h3>
      <div v-if="error" class="add-model-error">{{ error }}</div>
      <div class="add-model-field">
        <label>名称 *</label>
        <input v-model="form.name" type="text" placeholder="如: GPT-4o" />
      </div>
      <div class="add-model-field">
        <label>API Key *</label>
        <input v-model="form.apiKey" type="password" placeholder="sk-..." />
      </div>
      <div class="add-model-field">
        <label>Base URL</label>
        <input v-model="form.baseUrl" type="text" placeholder="默认: https://api.openai.com/v1" />
      </div>
      <div class="add-model-field">
        <label>模型名称 *</label>
        <input v-model="form.model" type="text" placeholder="如: gpt-4o" />
      </div>
      <div class="add-model-field">
        <label>最大输出 Tokens</label>
        <input v-model.number="form.maxTokens" type="number" placeholder="（服务端默认值）" />
      </div>
      <div class="add-model-actions">
        <button class="add-model-submit" :disabled="submitting" @click="handleSubmit">
          {{ submitting ? '添加中...' : '添加' }}
        </button>
        <button class="add-model-cancel" @click="$emit('cancel')">取消</button>
      </div>
    </div>
  </div>

  <div v-else class="add-model-inline">
    <div v-if="error" class="add-model-error">{{ error }}</div>
    <div class="add-model-field">
      <label>名称 *</label>
      <input v-model="form.name" type="text" placeholder="如: GPT-4o" />
    </div>
    <div class="add-model-field">
      <label>API Key *</label>
      <input v-model="form.apiKey" type="password" placeholder="sk-..." />
    </div>
    <div class="add-model-field">
      <label>Base URL</label>
      <input v-model="form.baseUrl" type="text" placeholder="默认: https://api.openai.com/v1" />
    </div>
    <div class="add-model-field">
      <label>模型名称 *</label>
      <input v-model="form.model" type="text" placeholder="如: gpt-4o" />
    </div>
    <div class="add-model-field">
      <label>最大输出 Tokens</label>
      <input v-model.number="form.maxTokens" type="number" placeholder="（服务端默认值）" />
    </div>
    <div class="add-model-actions">
      <button class="add-model-submit" :disabled="submitting" @click="handleSubmit">
        {{ submitting ? '添加中...' : '添加' }}
      </button>
      <button class="add-model-cancel" @click="$emit('cancel')">取消</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useModelStore } from '@/stores/model'
import { useUiStore } from '@/stores/ui'

defineProps<{
  mode?: 'modal' | 'inline'
}>()

const emit = defineEmits<{
  submit: [data: { name: string; api_key: string; base_url?: string; model: string; max_tokens?: number }]
  cancel: []
}>()

const modelStore = useModelStore()
const uiStore = useUiStore()

const submitting = ref(false)
const error = ref('')
const form = reactive({
  name: '',
  apiKey: '',
  baseUrl: '',
  model: '',
  maxTokens: 128000,
})

async function handleSubmit() {
  if (!form.name || !form.apiKey || !form.model) {
    error.value = '名称、API Key 和模型名称为必填项'
    return
  }
  error.value = ''
  submitting.value = true
  try {
    await modelStore.addModel({
      name: form.name,
      api_key: form.apiKey,
      base_url: form.baseUrl || undefined,
      model: form.model,
      max_tokens: form.maxTokens || undefined,
    })
    uiStore.showToast('success', '模型已添加')
    emit('submit', {
      name: form.name,
      api_key: form.apiKey,
      base_url: form.baseUrl || undefined,
      model: form.model,
      max_tokens: form.maxTokens || undefined,
    })
  } catch (e: any) {
    error.value = e.message || '添加失败'
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.add-model-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.5);
}

.add-model-dialog {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 20px;
  width: 420px;
  max-width: 90vw;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.add-model-inline {
  width: 100%;
}

.add-model-title {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
}

.add-model-error {
  padding: 8px 12px;
  margin-bottom: 12px;
  border-radius: 6px;
  background: rgba(239, 68, 68, 0.12);
  color: #ef4444;
  font-size: 12px;
}

.add-model-field {
  margin-bottom: 12px;
}

.add-model-field label {
  display: block;
  margin-bottom: 4px;
  font-size: 12px;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.add-model-field input {
  width: 100%;
  padding: 7px 10px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 13px;
  box-sizing: border-box;
}

.add-model-field input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.add-model-actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}

.add-model-submit {
  padding: 6px 16px;
  border: 1px solid var(--color-accent);
  border-radius: 4px;
  background: var(--color-accent);
  color: white;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  box-sizing: border-box;
  transition: opacity 0.15s;
}

.add-model-submit:hover {
  opacity: 0.9;
}

.add-model-submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.add-model-cancel {
  padding: 6px 16px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  box-sizing: border-box;
  transition: all 0.15s;
}

.add-model-cancel:hover {
  border-color: var(--color-text-secondary);
  color: var(--color-text-primary);
}
</style>