<script setup lang="ts">
import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'
import AppIcon from '@/components/common/AppIcon.vue'

const uiStore = useUiStore()

const isOpen = computed(() => uiStore.activeModal === 'help')
const appVersion = import.meta.env.VITE_APP_VERSION || 'dev'

const shortcuts = [
  { key: 'Enter', desc: '发送消息' },
  { key: 'Shift + Enter', desc: '换行' },
  { key: 'Shift + ↑/↓', desc: '切换输入历史' },
  { key: 'Escape', desc: '暂停/取消当前操作（智能判断）' },
  { key: 'Ctrl + K', desc: '打开命令面板' },
  { key: 'F2', desc: '重命名当前会话' },
  { key: 'Alt + Y', desc: '切换 YOLO 模式' },
  { key: 'Y', desc: '批准操作（审批弹窗中）' },
  { key: 'N', desc: '拒绝操作（审批弹窗中）' },
]

const commands = [
  { name: '/new', desc: '创建新会话', args: '[名称]' },
  { name: '/switch', desc: '切换会话', args: '' },
  { name: '/rename', desc: '重命名当前会话', args: '<新名称>' },
  { name: '/export', desc: '导出当前会话记录', args: '' },
  { name: '/rollback', desc: '回滚消息', args: '' },
  { name: '/pause', desc: '暂停当前会话', args: '' },
  { name: '/resume', desc: '恢复当前会话', args: '' },
  { name: '/cancel', desc: '取消当前操作', args: '' },
  { name: '/help', desc: '显示帮助', args: '' },
]

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    uiStore.setActiveModal(null)
  }
}
</script>

<template>
  <div v-if="isOpen" class="modal-overlay" @click="uiStore.setActiveModal(null)" @keydown="handleKeydown">
    <div class="help-panel" @click.stop>
      <div class="panel-header">
        <h3>帮助</h3>
        <button class="btn-close" @click="uiStore.setActiveModal(null)"><AppIcon name="x" :size="16" /></button>
      </div>

      <div class="panel-body">
        <section class="help-section">
          <h4>键盘快捷键</h4>
          <div class="shortcut-list">
            <div v-for="s in shortcuts" :key="s.key" class="shortcut-item">
              <kbd>{{ s.key }}</kbd>
              <span>{{ s.desc }}</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h4>/ 命令</h4>
          <div class="command-list">
            <div v-for="c in commands" :key="c.name" class="command-item">
              <code class="cmd-name">{{ c.name }}</code>
              <code v-if="c.args" class="cmd-args">{{ c.args }}</code>
              <span class="cmd-desc">{{ c.desc }}</span>
            </div>
          </div>
        </section>

        <section class="help-section">
          <h4>关于</h4>
          <p>Devo Web v{{ appVersion }} — AI 编码助手</p>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 6000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
}

.help-panel {
  width: 520px;
  max-height: 75vh;
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-modal);
  animation: modalIn var(--transition-base) ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-lg);
  border-bottom: 1px solid var(--color-border-light);
}

.panel-header h3 {
  font-size: var(--font-size-lg);
  font-weight: 600;
}

.btn-close {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  font-size: var(--font-size-base);
}

.btn-close:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-lg);
}

.help-section {
  margin-bottom: var(--space-xl);
}

.help-section h4 {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: var(--space-md);
}

.help-section p {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.shortcut-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.shortcut-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-xs) 0;
  font-size: var(--font-size-sm);
}

.shortcut-item kbd {
  display: inline-block;
  padding: 2px 8px;
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-primary);
}

.shortcut-item span {
  color: var(--color-text-secondary);
}

.command-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.command-item {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-xs) 0;
  font-size: var(--font-size-sm);
}

.cmd-name {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-accent);
  background: var(--color-accent-light);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
}

.cmd-args {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.cmd-desc {
  color: var(--color-text-secondary);
  margin-left: auto;
}
</style>