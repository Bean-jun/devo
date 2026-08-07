import { ref, watch, nextTick } from 'vue'

export interface ConfirmDeleteDialogProps {
  visible: boolean
  serverName: string
  deleting: boolean
  entityType?: string
}

export function useConfirmDeleteDialog(props: ConfirmDeleteDialogProps, emit: (e: string, ...args: any[]) => void) {
  const typeLabel = props.entityType || 'MCP 服务器'

  const inputValue = ref('')
  const inputRef = ref<HTMLInputElement | null>(null)

watch(() => props.visible, (val) => {
  if (val) {
    inputValue.value = ''
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
})

const isMatch = () => inputValue.value.trim() === props.serverName

function handleConfirm() {
  if (!isMatch() || props.deleting) return
  emit('confirm')
}

  return {
    typeLabel,
    inputValue,
    inputRef,
    isMatch,
    handleConfirm,
  }
}