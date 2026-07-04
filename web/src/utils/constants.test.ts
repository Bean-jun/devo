import { describe, it, expect } from 'vitest'
import {
  API_BASE,
  MAX_MESSAGE_LENGTH,
  TOAST_DURATION,
  SSE_RECONNECT_BASE_MS,
  SSE_RECONNECT_MAX_MS,
  AUTO_SCROLL_THRESHOLD,
  STATUS_LABELS,
  STATUS_COLORS,
  RISK_LABELS,
  RISK_COLORS,
} from '@/utils/constants'

describe('constants', () => {
  it('should have correct API_BASE', () => {
    expect(API_BASE).toBe('/api/v1')
  })

  it('should have reasonable MAX_MESSAGE_LENGTH', () => {
    expect(MAX_MESSAGE_LENGTH).toBeGreaterThan(0)
  })

  it('should have TOAST_DURATION', () => {
    expect(TOAST_DURATION).toBe(3000)
  })

  it('should have SSE reconnect constants', () => {
    expect(SSE_RECONNECT_BASE_MS).toBe(1000)
    expect(SSE_RECONNECT_MAX_MS).toBe(30000)
  })

  it('should have AUTO_SCROLL_THRESHOLD', () => {
    expect(AUTO_SCROLL_THRESHOLD).toBe(100)
  })

  describe('STATUS_LABELS', () => {
    it('should have labels for all states', () => {
      expect(STATUS_LABELS.idle).toBe('空闲')
      expect(STATUS_LABELS.processing).toBe('处理中')
      expect(STATUS_LABELS.awaiting_approval).toBe('等待审批')
      expect(STATUS_LABELS.paused).toBe('已暂停')
      expect(STATUS_LABELS.cancelled).toBe('已取消')
      expect(STATUS_LABELS.completed).toBe('已完成')
      expect(STATUS_LABELS.archived).toBe('已归档')
    })
  })

  describe('STATUS_COLORS', () => {
    it('should have colors for all states', () => {
      expect(STATUS_COLORS.idle).toBeTruthy()
      expect(STATUS_COLORS.processing).toBeTruthy()
      expect(STATUS_COLORS.awaiting_approval).toBeTruthy()
      expect(STATUS_COLORS.cancelled).toBe('#ff3b30')
    })
  })

  describe('RISK_LABELS', () => {
    it('should have labels for all risk levels', () => {
      expect(RISK_LABELS.low).toBe('低风险')
      expect(RISK_LABELS.medium).toBe('中风险')
      expect(RISK_LABELS.high).toBe('高风险')
    })
  })

  describe('RISK_COLORS', () => {
    it('should have colors for all risk levels', () => {
      expect(RISK_COLORS.low).toBeTruthy()
      expect(RISK_COLORS.medium).toBeTruthy()
      expect(RISK_COLORS.high).toBeTruthy()
    })
  })
})