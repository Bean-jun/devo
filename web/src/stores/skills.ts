import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { Skill, SkillInstallRequest } from '@/types/skills'
import { API_BASE } from '@/utils/constants'
import { useSessionStore } from '@/stores/session'

export const useSkillsStore = defineStore('skills', () => {
  const skills = ref<Skill[]>([])
  const isLoading = ref(false)

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

  async function toggleSkill(name: string, enabled: boolean): Promise<void> {
    const sessionStore = useSessionStore()
    const sessionId = sessionStore.currentSession?.id
    if (!sessionId) {
      const skill = skills.value.find(s => s.name === name)
      if (skill) skill.enabled = enabled
      return
    }
    const body = enabled ? { enable: [name] } : { disable: [name] }
    const res = await fetch(`${API_BASE}/sessions/${sessionId}/skills`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) throw new Error('切换技能失败')
    const skill = skills.value.find(s => s.name === name)
    if (skill) skill.enabled = enabled
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

  async function reloadSkills(): Promise<void> {
    isLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/skills/reload`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error('刷新技能列表失败')
      const data = await res.json()
      const list = Array.isArray(data.skills) ? data.skills : (data.skills || [])
      skills.value = list
    } catch {
      throw new Error('刷新技能列表失败')
    } finally {
      isLoading.value = false
    }
  }

  async function deleteSkill(name: string): Promise<void> {
    const res = await fetch(`${API_BASE}/skills/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: '删除失败' }))
      throw new Error(err.error || '删除技能失败')
    }
    await fetchSkills()
  }

  function updateSkillFromEvent(skill: Skill): void {
    const idx = skills.value.findIndex(s => s.name === skill.name)
    if (idx >= 0) {
      skills.value[idx] = { ...skills.value[idx], ...skill }
    } else {
      skills.value.push(skill)
    }
  }

  return {
    skills,
    isLoading,
    fetchSkills,
    toggleSkill,
    installSkill,
    reloadSkills,
    deleteSkill,
    updateSkillFromEvent,
  }
})