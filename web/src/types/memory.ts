export type MemoryType = 'user' | 'project'
export type MemorySource = 'user' | 'project'

export interface Memory {
  id: string
  type: MemoryType
  key: string
  content: string
  source: MemorySource
  createdAt: string
  updatedAt: string
}

export interface MemoryCreateRequest {
  type: MemoryType
  key: string
  content: string
}

export interface MemoryUpsertRequest {
  type: MemoryType
  key: string
  content: string
  action?: 'upsert' | 'delete'
}
