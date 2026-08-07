<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import type { Message } from '@/types/message'
import { useMessageBubble } from './MessageBubbleController'

const props = defineProps<{
  message: Message
}>()

const {
  copied,
  renderedContent,
  displayTime,
  copyContent,
  contentRef,
  lightboxImage,
  openLightbox,
  closeLightbox,
  handleCodeBlockCopy,
} = useMessageBubble(props)
</script>

<template>
  <div
    class="message-bubble"
    :class="[`role-${message.role}`]"
    :data-test="message.role === 'assistant' ? 'message-bubble assistant' : 'message-bubble'"
  >
    <div class="bubble-inner">
      <div class="bubble-header">
        <span class="bubble-role">
          {{ message.role === 'user' ? '你' : message.role === 'system' ? '系统' : 'Devo' }}
        </span>
        <div class="bubble-header-right">
          <button
            v-if="message.role === 'assistant'"
            class="copy-btn"
            :class="{ copied }"
            :title="copied ? '已复制' : '复制内容'"
            @click.stop="copyContent"
          >
            <template v-if="copied">
              <AppIcon name="check" :size="14" />
              已复制
            </template>
            <template v-if="!copied">
              <AppIcon name="copy" :size="14" />
              复制
            </template>
          </button>
          <span class="bubble-time">{{ displayTime }}</span>
        </div>
      </div>

      <details v-if="message.role === 'assistant' && message.reasoning" class="reasoning-collapse" data-test="reasoning-collapse">
        <summary class="reasoning-summary">
          <AppIcon name="brain" :size="14" class="reasoning-summary-icon" />
          思考过程
        </summary>
        <pre class="reasoning-text">{{ message.reasoning }}</pre>
      </details>

      <div
        v-if="message.role === 'assistant'"
        ref="contentRef"
        class="bubble-content markdown-body"
        v-html="renderedContent"
        @click="handleCodeBlockCopy"
      />
      <div v-else class="bubble-content">
        <div v-if="message.images && message.images.length > 0" class="message-images">
          <img
            v-for="(img, idx) in message.images"
            :key="idx"
            :src="img"
            alt="uploaded image"
            class="message-image"
            @click="openLightbox(img)"
          />
        </div>
        {{ message.content }}
      </div>
    </div>

    <Teleport to="body">
      <div v-if="lightboxImage" class="image-lightbox" @click="closeLightbox">
        <img :src="lightboxImage" alt="enlarged" class="lightbox-image" @click.stop />
        <button class="lightbox-close" @click="closeLightbox" aria-label="关闭">&times;</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped src="./MessageBubble.css">
</style>