import { useCommandStore } from '@/stores/command'
import type { Command } from '@/stores/command'

export function useCommand() {
  const commandStore = useCommandStore()

  const builtinCommands: Command[] = [
    {
      id: 'model',
      name: '/model',
      description: '切换模型',
      group: 'session',
      icon: 'robot',
      mobileLabel: '切换模型',
    },
    {
      id: 'new',
      name: '/new',
      description: '创建新会话',
      placeholder: '[名称]',
      group: 'session',
      icon: 'chat',
      mobileLabel: '创建新会话',
    },
    {
      id: 'switch',
      name: '/switch',
      description: '切换会话',
      group: 'session',
      icon: 'chat',
      mobileLabel: '切换会话',
    },
    {
      id: 'rename',
      name: '/rename',
      description: '重命名当前会话',
      placeholder: '<新名称>',
      group: 'session',
      icon: 'chat',
      mobileLabel: '重命名当前会话',
    },
    {
      id: 'export',
      name: '/export',
      description: '导出当前会话记录',
      group: 'session',
      icon: 'chat',
      mobileLabel: '导出会话记录',
    },
    {
      id: 'rollback',
      name: '/rollback',
      description: '回滚消息',
      group: 'session',
      icon: 'chat',
      mobileLabel: '回滚消息',
    },
    {
      id: 'pause',
      name: '/pause',
      description: '暂停当前会话',
      group: 'session',
      icon: 'chat',
      mobileLabel: '暂停当前会话',
    },
    {
      id: 'resume',
      name: '/resume',
      description: '恢复当前会话',
      group: 'session',
      icon: 'chat',
      mobileLabel: '恢复当前会话',
    },
    {
      id: 'cancel',
      name: '/cancel',
      description: '取消当前操作',
      group: 'session',
      icon: 'chat',
      mobileLabel: '取消当前操作',
    },
    {
      id: 'team',
      name: '/team',
      description: '切换 Team Mode',
      group: 'session',
      icon: 'users',
      mobileLabel: '切换 Team Mode',
    },
    {
      id: 'compact',
      name: '/compact',
      description: '压缩当前会话上下文',
      group: 'session',
      icon: 'chat',
      mobileLabel: '压缩会话上下文',
    },
    {
      id: 'help',
      name: '/help',
      description: '显示帮助',
      group: 'session',
      icon: 'chat',
      mobileLabel: '显示帮助',
    },
  ]

  const mobilePanelCommands: Command[] = [
    {
      id: 'files',
      name: '/files',
      description: '打开文件面板',
      group: 'panel',
      icon: 'folder',
      mobileLabel: '打开文件面板',
    },
    {
      id: 'skills',
      name: '/skills',
      description: '技能管理',
      group: 'panel',
      icon: 'puzzle-piece',
      mobileLabel: '技能管理',
    },
    {
      id: 'mcp',
      name: '/mcp',
      description: 'MCP 管理',
      group: 'panel',
      icon: 'plug',
      mobileLabel: 'MCP 管理',
    },
    {
      id: 'memory',
      name: '/memory',
      description: '记忆管理',
      group: 'panel',
      icon: 'brain',
      mobileLabel: '记忆管理',
    },
    {
      id: 'background',
      name: '/background',
      description: '后台任务',
      group: 'panel',
      icon: 'arrows-clockwise',
      mobileLabel: '后台任务',
    },
    {
      id: 'dashboard',
      name: '/dashboard',
      description: '仪表盘',
      group: 'panel',
      icon: 'chart-bar',
      mobileLabel: '仪表盘',
    },
    {
      id: 'settings',
      name: '/settings',
      description: '设置',
      group: 'panel',
      icon: 'gear',
      mobileLabel: '设置',
    },
  ]

  const mobileWorkspaceCommands: Command[] = [
    {
      id: 'workspace-switch',
      name: '/workspace-switch',
      description: '切换工作区',
      group: 'workspace',
      icon: 'folder',
      mobileLabel: '切换工作区',
    },
  ]

  const mobileAppCommands: Command[] = [
    {
      id: 'toggle-theme',
      name: '/toggle-theme',
      description: '切换主题',
      group: 'app',
      icon: 'circle-half',
      mobileLabel: '切换主题',
    },
    {
      id: 'status',
      name: '/status',
      description: '查看当前状态信息',
      group: 'app',
      icon: 'chart-bar',
      mobileLabel: '查看状态',
    },
    {
      id: 'version',
      name: '/version',
      description: '查看应用版本',
      group: 'app',
      icon: 'tag',
      mobileLabel: '查看版本',
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