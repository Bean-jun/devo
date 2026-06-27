export type SkillSource = 'project' | 'global'

export interface Skill {
  name: string
  source: SkillSource
  priority: number
  enabled: boolean
  location: string
  installedAt: string
}

export interface SkillInstallRequest {
  source: string
  value: string
}

export interface SkillToggleRequest {
  name: string
  enabled: boolean
}

export interface SetSessionSkillsRequest {
  enable: string[]
  disable: string[]
}