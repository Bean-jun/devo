<script setup lang="ts">
import ToastContainer from './ToastContainer.vue'
import CommandPalette from '@/components/command/CommandPalette.vue'
import ApprovalModal from '@/components/modal/ApprovalModal.vue'
import SessionPicker from '@/components/modal/SessionPicker.vue'
import RollbackPicker from '@/components/modal/RollbackPicker.vue'
import HelpPanel from '@/components/modal/HelpPanel.vue'
</script>

<template>
  <Teleport to="body">
    <ToastContainer />
    <CommandPalette />
    <ApprovalModal />
    <SessionPicker />
    <RollbackPicker />
    <HelpPanel />
  </Teleport>
</template>