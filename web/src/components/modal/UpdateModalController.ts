import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'
import { useUpdateCheck } from '@/composables/useUpdateCheck'
import { renderMarkdown } from '@/utils/markdown'

export function useUpdateModal() {
  const uiStore = useUiStore()
  const { updateInfo } = useUpdateCheck()

  const isOpen = computed(() => uiStore.activeModal === 'update')

  const releaseBodyHtml = computed(() => {
    if (!updateInfo.value?.release_body) return ''
    return renderMarkdown(updateInfo.value.release_body)
  })

  const publishedDate = computed(() => {
    if (!updateInfo.value?.published_at) return ''
    try {
      return new Date(updateInfo.value.published_at).toLocaleString('zh-CN')
    } catch {
      return updateInfo.value.published_at
    }
  })

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.stopPropagation()
      uiStore.setActiveModal(null)
    }
  }

  function openReleaseUrl() {
    if (updateInfo.value?.release_url) {
      window.open(updateInfo.value.release_url, '_blank')
    }
  }

  return {
    uiStore,
    isOpen,
    updateInfo,
    releaseBodyHtml,
    publishedDate,
    handleKeydown,
    openReleaseUrl,
  }
}