<script setup lang="ts">
import { useConfigWarningDialog } from './ConfigWarningDialogController'
import AddModelForm from '@/components/common/AddModelForm.vue'

const { isOpen, close, showAddForm } = useConfigWarningDialog()

function onModelAdded() {
  showAddForm.value = false
  close()
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
          Devo 需要 LLM API 密钥才能驱动 AI 编码。可通过下方表单快速添加，或手动创建配置文件：
        </p>

        <div v-if="!showAddForm" class="quick-add-section">
          <button class="quick-add-btn" @click="showAddForm = true">+ 快速添加模型</button>
        </div>

        <AddModelForm
          v-else
          mode="inline"
          @submit="onModelAdded"
          @cancel="showAddForm = false"
        />

        <template v-if="!showAddForm">
          <div class="divider">
            <span>或手动配置</span>
          </div>

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
        </template>
      </div>

      <div v-if="!showAddForm" class="dialog-footer">
        <button class="btn-close" @click="close">我知道了</button>
      </div>
    </div>
  </div>
</template>

<style scoped src="./ConfigWarningDialog.css">
</style>