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
      group: 'session',
      mobileLabel: '💬 创建新会话',
    },
    {
      id: 'switch',
      name: '/switch',
      description: '切换会话',
      group: 'session',
      mobileLabel: '💬 切换会话',
    },
    {
      id: 'rename',
      name: '/rename',
      description: '重命名当前会话',
      placeholder: '<新名称>',
      group: 'session',
      mobileLabel: '💬 重命名当前会话',
    },
    {
      id: 'export',
      name: '/export',
      description: '导出当前会话记录',
      group: 'session',
      mobileLabel: '💬 导出会话记录',
    },
    {
      id: 'rollback',
      name: '/rollback',
      description: '回滚消息',
      group: 'session',
      mobileLabel: '💬 回滚消息',
    },
    {
      id: 'pause',
      name: '/pause',
      description: '暂停当前会话',
      group: 'session',
      mobileLabel: '💬 暂停当前会话',
    },
    {
      id: 'resume',
      name: '/resume',
      description: '恢复当前会话',
      group: 'session',
      mobileLabel: '💬 恢复当前会话',
    },
    {
      id: 'cancel',
      name: '/cancel',
      description: '取消当前操作',
      group: 'session',
      mobileLabel: '💬 取消当前操作',
    },
    {
      id: 'help',
      name: '/help',
      description: '显示帮助',
      group: 'session',
      mobileLabel: '💬 显示帮助',
    },
  ]

  const mobilePanelCommands: Command[] = [
    {
      id: 'files',
      name: '/files',
      description: '打开文件面板',
      group: 'panel',
      mobileLabel: '📂 打开文件面板',
    },
    {
      id: 'skills',
      name: '/skills',
      description: '技能管理',
      group: 'panel',
      mobileLabel: '🧩 技能管理',
    },
    {
      id: 'mcp',
      name: '/mcp',
      description: 'MCP 管理',
      group: 'panel',
      mobileLabel: '🔌 MCP 管理',
    },
    {
      id: 'memory',
      name: '/memory',
      description: '记忆管理',
      group: 'panel',
      mobileLabel: '🧠 记忆管理',
    },
    {
      id: 'dashboard',
      name: '/dashboard',
      description: '仪表盘',
      group: 'panel',
      mobileLabel: '📊 仪表盘',
    },
    {
      id: 'settings',
      name: '/settings',
      description: '设置',
      group: 'panel',
      mobileLabel: '⚙️ 设置',
    },
  ]

  const mobileWorkspaceCommands: Command[] = [
    {
      id: 'workspace-switch',
      name: '/workspace-switch',
      description: '切换工作区',
      group: 'workspace',
      mobileLabel: '📁 切换工作区',
    },
  ]

  const mobileAppCommands: Command[] = [
    {
      id: 'toggle-theme',
      name: '/toggle-theme',
      description: '切换主题',
      group: 'app',
      mobileLabel: '🌙 切换主题',
    },
    {
      id: 'status',
      name: '/status',
      description: '查看当前状态信息',
      group: 'app',
      mobileLabel: '📊 查看状态',
    },
    {
      id: 'version',
      name: '/version',
      description: '查看应用版本',
      group: 'app',
      mobileLabel: '🏷️ 查看版本',
    },
  ]

  const allMobileCommands: Command[] = [
    ...builtinCommands,
    ...mobilePanelCommands,
    ...mobileWorkspaceCommands,
    ...mobileAppCommands,
  ]

  function openPalette(onSelect?: (cmd: Command) => void): void {
    commandStore.open(builtinCommands, onSelect)
  }

  function openMobilePalette(onSelect?: (cmd: Command) => void): void {
    commandStore.open(allMobileCommands, onSelect)
  }

  return {
    openPalette,
    openMobilePalette,
    builtinCommands,
    mobilePanelCommands,
    mobileWorkspaceCommands,
    mobileAppCommands,
    allMobileCommands,
  }
}