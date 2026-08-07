<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useInputArea } from './InputAreaController'

const props = defineProps<{
  isDisabled: boolean
  isProcessing: boolean
}>()

const emit = defineEmits<{
  send: [text: string, images?: string[]]
  stop: []
  clear: []
  openCommand: []
  executeCommand: [text: string]
}>()

const {
  editorRef,
  fileInputRef,
  lightboxImage,
  triggerFileInput,
  handleFileInputChange,
  openLightbox,
  closeLightbox,
  charCount,
  tokenEstimate,
  canSend,
  showPlaceholder,
  contextUsage,
  sessionTokens,
  workingDir,
  handleInput,
  handlePaste,
  handleKeydown,
  handleCompositionStart,
  handleCompositionEnd,
  handleImageDrop,
  handleDragOver,
  removeImage,
  send,
  isEmpty,
  isComposing,
  uploadedImages,
  autoResize,
  MAX_MESSAGE_LENGTH,
  formatTokenCount,
} = useInputArea(props, emit as any)
</script>

<template>
  <div class="input-area">
    <div class="input-row">
      <button
        class="command-btn"
        aria-label="命令面板"
        data-test="command-btn"
        @click="emit('openCommand')"
      >
        <span class="command-btn-text">/</span>
      </button>

      <input
        ref="fileInputRef"
        type="file"
        accept="image/*"
        multiple
        class="file-input-hidden"
        @change="handleFileInputChange"
      />

      <div class="input-wrapper">
        <div
          ref="editorRef"
          class="input-field"
          :class="{ 'is-empty': showPlaceholder, 'is-disabled': isDisabled }"
          contenteditable="true"
          data-test="message-input"
          role="textbox"
          aria-multiline="true"
          data-placeholder="输入消息，或按 / 使用命令..."
          @input="handleInput"
          @keydown="handleKeydown"
          @paste="handlePaste"
          @compositionstart="handleCompositionStart"
          @compositionend="handleCompositionEnd"
          @dragover="handleDragOver"
          @drop="handleImageDrop"
        />

        <div class="input-actions">
          <span class="input-info">
            <button
              class="image-upload-btn"
              aria-label="上传图片"
              title="上传图片"
              @click="triggerFileInput"
            >
              <AppIcon name="image" :size="14" />
            </button>
            <span v-if="charCount > 0" class="char-count" data-test="char-count">
              {{ charCount }} / {{ MAX_MESSAGE_LENGTH }}
              <span class="token-estimate">~{{ tokenEstimate }} tokens</span>
            </span>
          </span>

          <button
            v-if="isProcessing"
            class="btn-stop"
            data-test="stop-button"
            aria-label="停止"
            @click="emit('stop')"
          >
            <AppIcon name="stop" :size="14" class="stop-icon" />
            停止
          </button>

          <button
            v-else
            class="btn-send"
            :disabled="!canSend"
            aria-label="发送"
            @click="send"
          >
            发送
          </button>
        </div>

        <div v-if="uploadedImages.length > 0" class="image-previews">
          <div v-for="(img, idx) in uploadedImages" :key="idx" class="image-preview-item">
            <img :src="img" alt="preview" class="image-preview-thumb" @click="openLightbox(img)" />
            <button class="image-remove-btn" @click="removeImage(idx)" aria-label="移除图片">&times;</button>
          </div>
        </div>
      </div>
    </div>
    <div class="input-footer">
      <span class="footer-item">Context </span><span class="footer-item context-warn">{{ contextUsage }}</span>
      <span class="footer-sep">·</span>
      <span class="footer-item">Tokens {{ formatTokenCount(sessionTokens.total) }} (<AppIcon name="arrow-up" :size="10" />{{ formatTokenCount(sessionTokens.input) }} <AppIcon name="arrow-down" :size="10" />{{ formatTokenCount(sessionTokens.output) }})</span>
      <span v-if="workingDir" class="footer-item footer-dir">{{ workingDir }}</span>
    </div>

    <Teleport to="body">
      <div v-if="lightboxImage" class="image-lightbox" @click="closeLightbox">
        <img :src="lightboxImage" alt="enlarged" class="lightbox-image" @click.stop />
        <button class="lightbox-close" @click="closeLightbox" aria-label="关闭">&times;</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped src="./InputArea.css">
</style>