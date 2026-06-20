import { describe, it, expect } from 'vitest'
import {
  formatBytes,
  formatDuration,
  formatTokenCount,
  formatRelativeTime,
  formatTime,
  truncateText,
  estimateTokens,
  generateId,
} from '@/utils/formatters'

describe('formatBytes', () => {
  it('should format 0 bytes as "0 B"', () => {
    expect(formatBytes(0)).toBe('0 B')
  })

  it('should format bytes', () => {
    expect(formatBytes(500)).toBe('500 B')
  })

  it('should format kilobytes', () => {
    expect(formatBytes(2048)).toBe('2.0 KB')
  })

  it('should format megabytes', () => {
    expect(formatBytes(1048576)).toBe('1.0 MB')
  })

  it('should format gigabytes', () => {
    expect(formatBytes(1073741824)).toBe('1.0 GB')
  })
})

describe('formatDuration', () => {
  it('should format milliseconds', () => {
    expect(formatDuration(500)).toBe('500ms')
  })

  it('should format seconds', () => {
    expect(formatDuration(1500)).toBe('1.5s')
  })

  it('should format minutes and seconds', () => {
    expect(formatDuration(125000)).toBe('2m 5s')
  })
})

describe('formatTokenCount', () => {
  it('should return string for small numbers', () => {
    expect(formatTokenCount(500)).toBe('500')
  })

  it('should format with K suffix', () => {
    expect(formatTokenCount(1500)).toBe('1.5k')
  })
})

describe('formatRelativeTime', () => {
  it('should return "刚刚" for recent time', () => {
    const now = new Date()
    expect(formatRelativeTime(now.toISOString())).toBe('刚刚')
  })

  it('should return minutes ago', () => {
    const date = new Date(Date.now() - 5 * 60 * 1000)
    expect(formatRelativeTime(date.toISOString())).toBe('5 分钟前')
  })

  it('should return hours ago', () => {
    const date = new Date(Date.now() - 3 * 60 * 60 * 1000)
    expect(formatRelativeTime(date.toISOString())).toBe('3 小时前')
  })

  it('should return days ago', () => {
    const date = new Date(Date.now() - 5 * 24 * 60 * 60 * 1000)
    expect(formatRelativeTime(date.toISOString())).toBe('5 天前')
  })

  it('should return date for older dates', () => {
    expect(formatRelativeTime('2025-01-01T00:00:00Z')).toContain('2025')
  })
})

describe('formatTime', () => {
  it('should return time string', () => {
    const result = formatTime('2026-01-01T14:30:00Z')
    expect(result).toMatch(/\d{2}:\d{2}/)
  })
})

describe('truncateText', () => {
  it('should return original text if shorter than maxLength', () => {
    expect(truncateText('hello', 10)).toBe('hello')
  })

  it('should truncate long text with ellipsis', () => {
    expect(truncateText('hello world this is long', 10)).toBe('hello worl...')
  })

  it('should return original text when equal to maxLength', () => {
    expect(truncateText('hello', 5)).toBe('hello')
  })
})

describe('estimateTokens', () => {
  it('should estimate tokens based on text length', () => {
    expect(estimateTokens('hello world')).toBe(4)
  })

  it('should return 0 for empty string', () => {
    expect(estimateTokens('')).toBe(0)
  })

  it('should round up', () => {
    expect(estimateTokens('ab')).toBe(1)
  })
})

describe('generateId', () => {
  it('should generate a non-empty string', () => {
    const id = generateId()
    expect(id).toBeTruthy()
    expect(typeof id).toBe('string')
  })

  it('should generate unique IDs', () => {
    const ids = new Set(Array.from({ length: 100 }, () => generateId()))
    expect(ids.size).toBe(100)
  })
})