export type Scope = 'global' | `workspace:${string}`

export interface Workspace {
  id: string
  name: string
  path: string
  icon?: string
}

export function isGlobalScope(scope: Scope): boolean {
  return scope === 'global'
}

export function workspaceIdFromScope(scope: Scope): string | null {
  if (scope === 'global') return null
  return scope.replace('workspace:', '')
}

export function scopeFromWorkspaceId(id: string | null): Scope {
  return id ? `workspace:${id}` : 'global'
}