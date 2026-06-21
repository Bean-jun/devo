import { useCommandStore } from '@/stores/command'
import type { Command } from '@/stores/command'

export function useCommand() {
  const commandStore = useCommandStore()

  const builtinCommands: Command[] = [
    {
      id: 'new',
      name: '/new',
      description: '创建新会话',
      placeholder: '[名称]',
    },
    {
      id: 'switch',
      name: '/switch',
      description: '切换会话',
    },
    {
      id: 'rename',
      name: '/rename',
      description: '重命名当前会话',
      placeholder: '<新名称>',
    },
    {
      id: 'export',
      name: '/export',
      description: '导出当前会话记录',
    },
    {
      id: 'rollback',
      name: '/rollback',
      description: '回滚消息',
    },
    {
      id: 'pause',
      name: '/pause',
      description: '暂停当前会话',
    },
    {
      id: 'resume',
      name: '/resume',
      description: '恢复当前会话',
    },
    {
      id: 'cancel',
      name: '/cancel',
      description: '取消当前操作',
    },
    {
      id: 'help',
      name: '/help',
      description: '显示帮助',
    },
  ]

  function openPalette(onSelect?: (cmd: Command) => void): void {
    commandStore.open(builtinCommands, onSelect)
  }

  return { openPalette, builtinCommands }
}