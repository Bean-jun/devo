export const API_BASE = '/api/v1'

export const MAX_MESSAGE_LENGTH = 10000
export const TOAST_DURATION = 3000
export const SSE_RECONNECT_BASE_MS = 1000
export const SSE_RECONNECT_MAX_MS = 30000
export const AUTO_SCROLL_THRESHOLD = 100

export const STATUS_LABELS: Record<string, string> = {
  idle: '空闲',
  processing: '处理中',
  awaiting_approval: '等待审批',
  paused: '已暂停',
  completed: '已完成',
  archived: '已归档',
}

export const STATUS_COLORS: Record<string, string> = {
  idle: '#34c759',
  processing: '#0071e3',
  awaiting_approval: '#ff9500',
  paused: '#aeaeb2',
  completed: '#34c759',
  archived: '#86868b',
}

export const RISK_LABELS: Record<string, string> = {
  low: '低风险',
  medium: '中风险',
  high: '高风险',
}

export const RISK_COLORS: Record<string, string> = {
  low: '#34c759',
  medium: '#ff9500',
  high: '#ff3b30',
}