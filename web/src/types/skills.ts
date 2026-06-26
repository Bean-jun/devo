import type { Scope } from './workspace'

export type SkillStatus = 'active' | 'inactive'

export interface Skill {
  name: string
  displayName: string
  description: string
  icon: string
  scope: Scope
  status: SkillStatus
  version: string
  source: string
  installedAt: string
}

export interface SkillInstallRequest {
  source: string
  scope: Scope
}

export interface SkillToggleRequest {
  name: string
  scope: Scope
  enabled: boolean
}