import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { Skill, SkillInstallRequest } from '@/types/skills'
import { API_BASE } from '@/utils/constants'

export const useSkillsStore = defineStore('skills', () => {
  const skills = ref<Skill[]>([])
  const isLoading = ref(false)

  const globalSkills = computed(() =>
    skills.value.filter(s => s.scope === 'global')
  )

  function workspaceSkills(workspaceId: string): Skill[] {
    return skills.value.filter(s => s.scope === `workspace:${workspaceId}`)
  }

  async function fetchSkills(): Promise<void> {
    isLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/skills`)
      if (!res.ok) throw new Error(`Failed to fetch skills: ${res.status}`)
      const data = await res.json()
      const list = Array.isArray(data) ? data : (data.skills || [])
      skills.value = list
    } catch {
      throw new Error('获取技能列表失败')
    } finally {
      isLoading.value = false
    }
  }

  async function toggleSkill(name: string, scope: string, enabled: boolean): Promise<void> {
    const skill = skills.value.find(s => s.name === name && s.scope === scope)
    if (skill) {
      skill.status = enabled ? 'active' : 'inactive'
    }
  }

  async function installSkill(request: SkillInstallRequest): Promise<void> {
    const res = await fetch(`${API_BASE}/skills/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    })
    if (!res.ok) throw new Error('安装技能失败')
    await fetchSkills()
  }

  async function deleteSkill(name: string, scope: string): Promise<void> {
    const res = await fetch(`${API_BASE}/skills/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    })
    if (!res.ok) throw new Error('删除技能失败')
    skills.value = skills.value.filter(s => !(s.name === name && s.scope === scope))
  }

  function updateSkillFromEvent(skill: Skill): void {
    const idx = skills.value.findIndex(s => s.name === skill.name && s.scope === skill.scope)
    if (idx >= 0) {
      skills.value[idx] = { ...skills.value[idx], ...skill }
    } else {
      skills.value.push(skill)
    }
  }

  return {
    skills,
    isLoading,
    globalSkills,
    workspaceSkills,
    fetchSkills,
    toggleSkill,
    installSkill,
    deleteSkill,
    updateSkillFromEvent,
  }
})