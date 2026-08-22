import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

export function useHelpPanel() {
  const uiStore = useUiStore()

  const isOpen = computed(() => uiStore.activeModal === 'help')
  const appVersion = import.meta.env.VITE_APP_VERSION || 'dev'

  const shortcuts = [
    { key: 'Enter', desc: '发送消息' },
    { key: 'Shift + Enter', desc: '换行' },
    { key: 'Shift + ↑/↓', desc: '切换输入历史' },
    { key: 'Escape', desc: '暂停/取消当前操作（智能判断）' },
    { key: 'Ctrl + K', desc: '打开命令面板' },
    { key: 'F2', desc: '重命名当前会话' },
    { key: 'Alt + Y', desc: '切换 YOLO 模式' },
    { key: 'Alt + E', desc: '切换 Team Mode' },
    { key: 'Y', desc: '批准操作（审批弹窗中）' },
    { key: 'N', desc: '拒绝操作（审批弹窗中）' },
  ]

  const commands = [
    { name: '/model', desc: '切换模型', args: '' },
    { name: '/new', desc: '创建新会话', args: '[名称]' },
    { name: '/switch', desc: '切换会话', args: '' },
    { name: '/rename', desc: '重命名当前会话', args: '<新名称>' },
    { name: '/export', desc: '导出当前会话记录', args: '' },
    { name: '/rollback', desc: '回滚消息', args: '' },
    { name: '/pause', desc: '暂停当前会话', args: '' },
    { name: '/resume', desc: '恢复当前会话', args: '' },
    { name: '/cancel', desc: '取消当前操作', args: '' },
    { name: '/team', desc: '切换 Team Mode', args: '' },
    { name: '/help', desc: '显示帮助', args: '' },
  ]

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.stopPropagation()
      uiStore.setActiveModal(null)
    }
  }

  return { uiStore, isOpen, appVersion, shortcuts, commands, handleKeydown }
}