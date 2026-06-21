import { useCommandStore } from '@/stores/command'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { API_BASE } from '@/utils/constants'
import type { Command } from '@/stores/command'

export function useCommand() {
  const commandStore = useCommandStore()
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()

  const builtinCommands: Command[] = [
    {
      id: 'new',
      name: '/new',
      description: '创建新会话',
      action: () => {
        uiStore.openInputPrompt({
          title: '创建新会话',
          placeholder: '会话名称（可选，留空则自动生成）',
          confirmLabel: '创建',
          onConfirm: (value) => {
            sessionStore.createSession({ title: value || undefined, workingDirectory: sessionStore.workingDirectory })
          },
        })
      },
    },
    {
      id: 'switch',
      name: '/switch',
      description: '切换会话',
      action: () => {
        uiStore.setActiveModal('session-picker')
      },
    },
    {
      id: 'rename',
      name: '/rename',
      description: '重命名当前会话',
      action: () => {
        if (!sessionStore.currentSession) return
        uiStore.openInputPrompt({
          title: '重命名当前会话',
          placeholder: '输入新名称',
          defaultValue: sessionStore.currentSession.title || '',
          confirmLabel: '重命名',
          onConfirm: (value) => {
            sessionStore.renameSession(sessionStore.currentSession!.id, value)
          },
        })
      },
    },
    {
      id: 'export',
      name: '/export',
      description: '导出当前会话记录',
      action: async () => {
        if (!sessionStore.currentSession) return
        const sid = sessionStore.currentSession.id
        await fetch(`${API_BASE}/sessions/${sid}/sync-archive`, { method: 'POST' })
        const url = `${API_BASE}/sessions/${sid}/archive`
        const filename = `${sid}.md`
        const a = document.createElement('a')
        a.href = url
        a.download = filename
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
      },
    },
    {
      id: 'rollback',
      name: '/rollback',
      description: '回滚消息',
      action: () => {
        uiStore.setActiveModal('rollback-picker')
      },
    },
    {
      id: 'pause',
      name: '/pause',
      description: '暂停当前会话',
      action: () => {},
    },
    {
      id: 'resume',
      name: '/resume',
      description: '恢复当前会话',
      action: () => {},
    },
    {
      id: 'cancel',
      name: '/cancel',
      description: '取消当前操作',
      action: () => {},
    },
    {
      id: 'help',
      name: '/help',
      description: '显示帮助',
      action: () => {
        uiStore.setActiveModal('help')
      },
    },
  ]

  function openPalette(): void {
    commandStore.open(builtinCommands)
  }

  return { openPalette, builtinCommands }
}