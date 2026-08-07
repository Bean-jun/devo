<script setup lang="ts">
import { useGlobalModals } from './GlobalModalsController'

const {
  ToastContainer,
  CommandPalette,
  ApprovalModal,
  SessionPicker,
  RollbackPicker,
  HelpPanel,
  ConfigWarningDialog,
} = useGlobalModals()
</script>

<template>
  <Teleport to="body">
    <ToastContainer />
    <CommandPalette />
    <ApprovalModal />
    <SessionPicker />
    <RollbackPicker />
    <HelpPanel />
    <ConfigWarningDialog />
  </Teleport>
</template>