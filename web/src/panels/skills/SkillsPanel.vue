<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useSkillsStore } from '@/stores/skills'
import { useUiStore } from '@/stores/ui'
import type { Skill } from '@/types/skills'

const skillsStore = useSkillsStore()
const uiStore = useUiStore()

type ScopeFilter = 'all' | 'global' | 'workspace'
const scopeFilter = ref<ScopeFilter>('all')
const error = ref<string | null>(null)

const filteredSkills = computed<Skill[]>(() => {
  let list = skillsStore.skills
  if (scopeFilter.value === 'global') {
    list = skillsStore.globalSkills
  } else if (scopeFilter.value === 'workspace' && uiStore.activeWorkspace) {
    list = skillsStore.workspaceSkills(uiStore.activeWorkspace)
  }
  return list
})

async function handleToggle(skill: Skill) {
  try {
    await skillsStore.toggleSkill(skill.name, skill.scope, skill.status === 'inactive')
  } catch (e: any) {
    error.value = e.message || '切换失败'
  }
}

async function handleDelete(skill: Skill) {
  try {
    await skillsStore.deleteSkill(skill.name, skill.scope)
  } catch (e: any) {
    error.value = e.message || '删除失败'
  }
}

onMounted(async () => {
  try {
    await skillsStore.fetchSkills()
  } catch (e: any) {
    error.value = e.message
  }
})
</script>

<template>
  <div class="skills-panel">
    <div class="skills-toolbar">
      <div class="scope-filter">
        <button class="filter-btn" :class="{ active: scopeFilter === 'all' }" @click="scopeFilter = 'all'">全部</button>
        <button class="filter-btn" :class="{ active: scopeFilter === 'global' }" @click="scopeFilter = 'global'">全局</button>
        <button class="filter-btn" :class="{ active: scopeFilter === 'workspace' }" @click="scopeFilter = 'workspace'">项目</button>
      </div>
    </div>
    <div v-if="error" class="skills-error">{{ error }}</div>
    <div v-else-if="skillsStore.isLoading" class="skills-loading">加载中...</div>
    <div v-else-if="filteredSkills.length === 0" class="skills-empty">暂无技能</div>
    <div v-else class="skills-list">
      <div v-for="skill in filteredSkills" :key="skill.name + ':' + skill.scope" class="skill-card">
        <div class="skill-info">
          <span class="skill-icon">{{ skill.icon }}</span>
          <div class="skill-detail">
            <span class="skill-name">{{ skill.displayName }}</span>
            <span class="skill-desc">{{ skill.description }}</span>
          </div>
        </div>
        <div class="skill-actions">
          <span class="skill-scope-tag">{{ skill.scope === 'global' ? '全局' : '项目' }}</span>
          <button class="skill-toggle" :class="{ active: skill.status === 'active' }" @click="handleToggle(skill)">
            {{ skill.status === 'active' ? 'ON' : 'OFF' }}
          </button>
          <button class="skill-delete" @click="handleDelete(skill)">✕</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.skills-panel { display: flex; flex-direction: column; height: 100%; }
.skills-toolbar { padding: 8px 12px; border-bottom: 1px solid var(--color-border); }
.scope-filter { display: flex; gap: 4px; }
.filter-btn { padding: 4px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; transition: all var(--transition-fast); }
.filter-btn:hover { background: var(--color-bg-hover); }
.filter-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.skills-error { padding: 16px; color: var(--color-error); font-size: 12px; }
.skills-loading { padding: 16px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.skills-empty { padding: 16px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.skills-list { flex: 1; overflow-y: auto; padding: 8px; }
.skill-card { display: flex; align-items: center; justify-content: space-between; padding: 8px; border: 1px solid var(--color-border); border-radius: 6px; margin-bottom: 6px; transition: border-color var(--transition-fast); }
.skill-card:hover { border-color: var(--color-accent); }
.skill-info { display: flex; align-items: center; gap: 8px; min-width: 0; }
.skill-icon { font-size: 18px; }
.skill-detail { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.skill-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.skill-desc { font-size: 11px; color: var(--color-text-tertiary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.skill-actions { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.skill-scope-tag { font-size: 10px; padding: 2px 6px; border-radius: 3px; background: var(--color-bg-secondary); color: var(--color-text-tertiary); }
.skill-toggle { padding: 2px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 10px; font-weight: 600; }
.skill-toggle.active { background: var(--color-success); border-color: var(--color-success); color: white; }
.skill-delete { background: none; border: none; color: var(--color-text-tertiary); cursor: pointer; font-size: 12px; padding: 2px; }
.skill-delete:hover { color: var(--color-error); }
</style>