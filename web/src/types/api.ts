export interface ApiResponse<T> {
  data: T
  error?: string
}

export interface ApiError {
  error: string
  status: number
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  offset: number
  limit: number
}

export interface UpdateCheckResult {
  has_update: boolean
  current_version: string
  latest_version: string
  release_url: string
  release_name: string
  release_body: string
  published_at: string
  checked_at: string
}