import { useCommandStore } from '@/stores/command'
import { useSessionStore } from '@/stores/session'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import type { Command } from '@/stores/command'

export function useCommand() {
  const commandStore = useCommandStore()
  const sessionStore = useSessionStore()
  const chatStore = useChatStore()
  const uiStore = useUiStore()

  const builtinCommands: Command[] = [
    {
      id: 'new',
      name: '/new',
      description: '创建新会话',
      placeholder: '会话名称（可选）',
      action: (args?: string) => {
        uiStore.setActiveModal(null)
        sessionStore.createSession({ title: args || undefined, workingDirectory: sessionStore.workingDirectory })
      },
    },
    {
      id: 'sessions',
      name: '/sessions',
      description: '查看会话列表',
      action: () => {
        uiStore.setActiveModal('session-picker')
      },
    },
    {
      id: 'switch',
      name: '/switch',
      description: '切换会话',
      placeholder: '会话ID',
      action: async (args?: string) => {
        if (args) {
          await sessionStore.switchSessionById(args)
        }
      },
    },
    {
      id: 'rename',
      name: '/rename',
      description: '重命名当前会话',
      placeholder: '新名称',
      action: async (args?: string) => {
        if (args && sessionStore.currentSession) {
          await sessionStore.renameSession(sessionStore.currentSession.id, args)
        }
      },
    },
    {
      id: 'archive',
      name: '/archive',
      description: '归档当前会话',
      action: () => {
        if (sessionStore.currentSession) {
          if (confirm('确定要归档当前会话吗？')) {
            sessionStore.archiveSession(sessionStore.currentSession.id)
          }
        }
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
    {
      id: 'clear',
      name: '/clear',
      description: '清屏',
      action: () => {
        chatStore.clearMessages()
      },
    },
  ]

  function openPalette(): void {
    commandStore.open(builtinCommands)
  }

  return { openPalette, builtinCommands }
}