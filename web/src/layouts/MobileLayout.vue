<script setup lang="ts">
import StatusBar from '@/components/layout/StatusBar.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import GlobalModals from '@/components/layout/GlobalModals.vue'
import MobileInputBar from '@/components/mobile/MobileInputBar.vue'
import MobileCommandSheet from '@/components/mobile/MobileCommandSheet.vue'
import MobilePanelDrawer from '@/components/mobile/MobilePanelDrawer.vue'
import MobileWorkspacePicker from '@/components/mobile/MobileWorkspacePicker.vue'
import MobileSessionPicker from '@/components/mobile/MobileSessionPicker.vue'
import { usePlatform } from '@/composables/usePlatform'
import { useMobileLayout } from './MobileLayoutController'

const { densityLevel } = usePlatform()

const {
  sessionStore,
  isProcessing,
  handleSend,
  handleStop,
  openCommandSheet,
  showCommandSheet,
  handleCommandSelect,
  closeCommandSheet,
  showPanelDrawer,
  closePanelDrawer,
  showWorkspacePicker,
  closeWorkspacePicker,
  handleWorkspaceSwitched,
  showSessionPicker,
  closeSessionPicker,
  handleNewSessionFromPicker,
  showNewSessionDialog,
  newSessionTitle,
  cancelNewSession,
  confirmNewSession,
  showInfoDialog,
  infoDialogTitle,
  infoDialogContent,
  closeInfoDialog,
} = useMobileLayout()
</script>

<template>
  <div class="mobile-layout" data-test="mobile-layout">
    <StatusBar v-if="sessionStore.currentSession" :density="densityLevel" />
    <div class="mobile-chat">
      <ChatPanel :hide-input="true" />
    </div>
    <MobileInputBar
      :is-disabled="false"
      :is-processing="isProcessing"
      @send="handleSend"
      @stop="handleStop"
      @open-command="openCommandSheet"
    />
    <GlobalModals />

    <MobileCommandSheet
      v-if="showCommandSheet"
      @select="handleCommandSelect"
      @close="closeCommandSheet"
    />

    <MobilePanelDrawer
      v-if="showPanelDrawer"
      @close="closePanelDrawer"
    />

    <MobileWorkspacePicker
      v-if="showWorkspacePicker"
      @close="closeWorkspacePicker"
      @switched="handleWorkspaceSwitched"
    />

    <MobileSessionPicker
      v-if="showSessionPicker"
      @close="closeSessionPicker"
      @new-session="handleNewSessionFromPicker"
    />

    <Teleport to="body">
      <div v-if="showNewSessionDialog" class="dialog-overlay" @click.self="cancelNewSession" data-test="new-session-dialog-overlay">
        <div class="dialog-sheet" @click.stop data-test="new-session-dialog">
          <div class="dialog-title">新建会话</div>
          <input
            v-model="newSessionTitle"
            class="new-session-mobile-input"
            type="text"
            placeholder="输入会话名称（可选）"
            @keydown.enter="confirmNewSession"
            @keydown.escape.stop="cancelNewSession"
          />
          <select
            v-if="sessionStore.agents.length > 0"
            v-model="sessionStore.selectedAgentId"
            class="new-session-mobile-input agent-select"
          >
            <option
              v-for="agent in sessionStore.agents"
              :key="agent.id"
              :value="agent.id"
            >{{ agent.name }}</option>
          </select>
          <div class="dialog-actions">
            <button class="dialog-btn-cancel" @click="cancelNewSession">取消</button>
            <button class="dialog-btn-confirm" @click="confirmNewSession">创建</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showInfoDialog" class="dialog-overlay" @click.self="closeInfoDialog" data-test="info-dialog-overlay">
        <div class="dialog-sheet" @click.stop data-test="info-dialog">
          <div class="dialog-title" data-test="info-dialog-title">{{ infoDialogTitle }}</div>
          <pre class="info-dialog-content" data-test="info-dialog-content">{{ infoDialogContent }}</pre>
          <div class="dialog-actions">
            <button class="dialog-btn-confirm" @click="closeInfoDialog" data-test="info-dialog-confirm">确认</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped src="./MobileLayout.css">
</style>