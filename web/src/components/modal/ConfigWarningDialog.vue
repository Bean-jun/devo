<script setup lang="ts">
import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()

const isOpen = computed(() => uiStore.activeModal === 'config-warning')

function close() {
  uiStore.setActiveModal(null)
}
</script>

<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="close">
    <div class="config-warning-dialog" @click.stop>
      <div class="dialog-header">
        <div class="dialog-icon">!</div>
        <h3 class="dialog-title">LLM API Key 未配置</h3>
      </div>

      <div class="dialog-body">
        <p class="dialog-desc">
          Devo 需要 LLM API 密钥才能驱动 AI 编码。请在以下任一位置创建配置文件：
        </p>

        <div class="config-locations">
          <div class="location-item">
            <div class="location-label">项目配置</div>
            <code class="location-path">.devo/config.json</code>
            <span class="location-hint">仅当前项目生效</span>
          </div>
          <div class="location-item">
            <div class="location-label">全局配置</div>
            <code class="location-path">~/.devo/config.json</code>
            <span class="location-hint">所有项目生效</span>
          </div>
        </div>

        <p class="dialog-example-title">配置文件示例：</p>
        <pre class="config-example"><code>{
  "llm": {
    "api_key": "sk-your-key-here",
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4o"
  }
}</code></pre>

        <p class="dialog-env-hint">
          也可以通过环境变量 <code>DEVO_LLM_API_KEY</code> 设置。
        </p>
      </div>

      <div class="dialog-footer">
        <button class="btn-close" @click="close">我知道了</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.config-warning-dialog {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  box-shadow: var(--shadow-lg);
  width: 480px;
  max-width: 90vw;
  overflow: hidden;
}

.dialog-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 20px 0;
}

.dialog-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--color-warning, #f59e0b);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 16px;
  flex-shrink: 0;
}

.dialog-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.dialog-body {
  padding: 16px 20px;
}

.dialog-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin: 0 0 16px;
  line-height: 1.5;
}

.config-locations {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.location-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--color-bg-secondary);
  border-radius: 6px;
  border: 1px solid var(--color-border);
}

.location-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-primary);
  min-width: 56px;
}

.location-path {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-accent);
  flex: 1;
}

.location-hint {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.dialog-example-title {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin: 0 0 6px;
}

.config-example {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 12px;
  margin: 0 0 12px;
  overflow-x: auto;
}

.config-example code {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--color-text-primary);
  line-height: 1.6;
  white-space: pre;
}

.dialog-env-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin: 0;
}

.dialog-env-hint code {
  font-family: var(--font-mono);
  background: var(--color-bg-secondary);
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  color: var(--color-accent);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  padding: 12px 20px;
  border-top: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}

.btn-close {
  padding: 7px 20px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all var(--transition-fast);
}

.btn-close:hover {
  background: var(--color-bg-hover);
  border-color: var(--color-accent);
  color: var(--color-accent);
}
</style>