<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useMobileInputBar } from './MobileInputBarController'

const props = defineProps<{
  isDisabled: boolean
  isProcessing: boolean
}>()

const emit = defineEmits<{
  send: [text: string, images?: string[]]
  stop: []
  openCommand: []
}>()

const {
  inputText,
  textareaRef,
  fileInputRef,
  sessionStore,
  uiStore,
  pastedFullText,
  pasteLabel,
  uploadedImages,
  lightboxImage,
  charCount,
  tokenEstimate,
  canSend,
  contextUsage,
  sessionTokens,
  focusTextarea,
  handleKeydown,
  send,
  handlePaste,
  autoResize,
  triggerFileInput,
  handleFileInputChange,
  openLightbox,
  closeLightbox,
  openCommand,
  removeImage,
  MAX_MESSAGE_LENGTH,
  formatTokenCount,
} = useMobileInputBar(props, emit as any)
</script>

<template>
  <div class="mobile-input-bar">
    <div v-if="uploadedImages.length > 0" class="mobile-image-previews">
      <div v-for="(img, idx) in uploadedImages" :key="idx" class="mobile-image-preview-item">
        <img :src="img" alt="preview" class="mobile-image-preview-thumb" @click="openLightbox(img)" />
        <button class="mobile-image-remove-btn" @click="removeImage(idx)" aria-label="移除">&times;</button>
      </div>
    </div>

    <div class="mobile-input-row">
      <input
        ref="fileInputRef"
        type="file"
        accept="image/*"
        multiple
        class="file-input-hidden"
        @change="handleFileInputChange"
      />

      <button
        class="command-btn"
        :class="{ pressed: false }"
        aria-label="命令面板"
        data-test="mobile-command-btn"
        @click="openCommand"
        @touchstart.prevent
        @touchend.prevent="openCommand"
      >
        <span class="command-btn-text">/</span>
      </button>

      <div class="mobile-input-wrapper">
        <textarea
          ref="textareaRef"
          v-model="inputText"
          class="mobile-input-field"
          placeholder="输入消息..."
          :disabled="isDisabled"
          :maxlength="MAX_MESSAGE_LENGTH"
          rows="1"
          data-test="mobile-input-textarea"
          @keydown="handleKeydown"
          @paste="handlePaste"
        />
        <button
          class="mobile-input-image-btn"
          aria-label="上传图片"
          @click="triggerFileInput"
          @touchstart.prevent
          @touchend.prevent="triggerFileInput"
        >
          <AppIcon name="image" :size="18" />
        </button>
      </div>

      <button
        v-if="isProcessing"
        class="mobile-btn-stop"
        aria-label="停止"
        data-test="mobile-stop-btn"
        @click="emit('stop')"
      >
        <AppIcon name="stop" :size="18" />
      </button>
      <button
        v-else
        class="mobile-btn-send"
        :disabled="!canSend"
        aria-label="发送"
        @click="send"
      >
        <AppIcon name="caret-right" :size="18" color="white" />
      </button>
    </div>

    <div class="mobile-input-footer" data-test="mobile-input-footer">
      <div class="footer-row">
        <span class="footer-item">Context <span class="context-warn">{{ contextUsage }}</span></span>
        <span class="footer-sep">·</span>
        <span class="footer-item">Tokens {{ formatTokenCount(sessionTokens.total) }} (<AppIcon name="arrow-up" :size="10" />{{ formatTokenCount(sessionTokens.input) }} <AppIcon name="arrow-down" :size="10" />{{ formatTokenCount(sessionTokens.output) }})</span>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="lightboxImage" class="mobile-image-lightbox" @click="closeLightbox">
        <img :src="lightboxImage" alt="enlarged" class="mobile-lightbox-image" @click.stop />
        <button class="mobile-lightbox-close" @click="closeLightbox" aria-label="关闭">&times;</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped src="./MobileInputBar.css">
</style>