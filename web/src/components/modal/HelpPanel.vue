<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useHelpPanel } from './HelpPanelController'

const { uiStore, isOpen, appVersion, shortcuts, commands, handleKeydown } = useHelpPanel()
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

<style scoped src="./HelpPanel.css">
</style>