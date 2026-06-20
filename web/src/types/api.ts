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