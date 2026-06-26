import type { Scope } from './workspace'

export interface Memory {
  id: string
  key: string
  value: string
  scope: Scope
  createdAt: string
  updatedAt: string
}

export interface MemoryCreateRequest {
  key: string
  value: string
  scope: Scope
}

export interface MemoryUpdateRequest {
  value: string
}