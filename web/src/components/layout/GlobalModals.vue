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
  UpdateModal,
  ModelPicker,
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
    <UpdateModal />
    <ModelPicker />
  </Teleport>
</template>