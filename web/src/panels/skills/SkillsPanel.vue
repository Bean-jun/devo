<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useSkillsStore } from '@/stores/skills'
import type { Skill, SkillSource } from '@/types/skills'
import ConfirmDeleteDialog from '@/components/modal/ConfirmDeleteDialog.vue'

const skillsStore = useSkillsStore()

type SourceFilter = 'all' | SkillSource
const sourceFilter = ref<SourceFilter>('all')
const error = ref<string | null>(null)

const showDeleteDialog = ref(false)
const deletingSkill = ref<Skill | null>(null)

const showInstallDialog = ref(false)
const installPath = ref('')
const installing = ref(false)
const installError = ref<string | null>(null)

const filteredSkills = computed<Skill[]>(() => {
  if (sourceFilter.value === 'all') return skillsStore.skills
  return skillsStore.skills.filter(s => s.source === sourceFilter.value)
})

async function handleToggle(skill: Skill) {
  try {
    await skillsStore.toggleSkill(skill.name, !skill.enabled)
  } catch (e: any) {
    error.value = e.message || '切换失败'
  }
}

async function handleDelete(skill: Skill) {
  deletingSkill.value = skill
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingSkill.value) return
  try {
    await skillsStore.deleteSkill(deletingSkill.value.name)
    showDeleteDialog.value = false
    deletingSkill.value = null
  } catch (e: any) {
    error.value = e.message || '删除失败'
  }
}

function cancelDelete() {
  showDeleteDialog.value = false
  deletingSkill.value = null
}

async function handleReload() {
  try {
    await skillsStore.reloadSkills()
  } catch (e: any) {
    error.value = e.message || '刷新失败'
  }
}

function openInstallDialog() {
  installPath.value = ''
  installError.value = null
  showInstallDialog.value = true
}

function closeInstallDialog() {
  showInstallDialog.value = false
  installPath.value = ''
  installError.value = null
}

async function handleInstall() {
  const path = installPath.value.trim()
  if (!path) {
    installError.value = '请输入技能目录路径'
    return
  }
  installing.value = true
  installError.value = null
  try {
    await skillsStore.installSkill({ source: 'local', value: path })
    closeInstallDialog()
  } catch (e: any) {
    installError.value = e.message || '安装失败'
  } finally {
    installing.value = false
  }
}

function sourceLabel(source: string): string {
  switch (source) {
    case 'project': return '项目'
    case 'global': return '全局'
    default: return source
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
        <button class="filter-btn" :class="{ active: sourceFilter === 'all' }" @click="sourceFilter = 'all'">全部</button>
        <button class="filter-btn" :class="{ active: sourceFilter === 'project' }" @click="sourceFilter = 'project'">项目</button>
        <button class="filter-btn" :class="{ active: sourceFilter === 'global' }" @click="sourceFilter = 'global'">全局</button>
      </div>
      <div class="toolbar-right">
        <button class="reload-btn" :class="{ spinning: skillsStore.isLoading }" :disabled="skillsStore.isLoading" @click="handleReload" title="重新扫描技能目录">
          ↻
        </button>
        <button class="add-btn" @click="openInstallDialog" title="安装技能">
          添加
        </button>
      </div>
    </div>
    <div v-if="error" class="skills-error">{{ error }}</div>
    <div v-else-if="skillsStore.isLoading" class="skills-loading">加载中...</div>
    <div v-else-if="filteredSkills.length === 0" class="skills-empty">暂无技能</div>
    <div v-else class="skills-list">
      <div v-for="skill in filteredSkills" :key="skill.name" class="skill-card">
        <div class="skill-info">
          <div class="skill-detail">
            <span class="skill-name">{{ skill.name }}</span>
            <span class="skill-source">{{ sourceLabel(skill.source) }} · {{ skill.location || '内置' }}</span>
          </div>
        </div>
        <div class="skill-actions">
          <span class="skill-source-tag" :class="'source-' + skill.source">{{ sourceLabel(skill.source) }}</span>
          <button class="skill-toggle" :class="{ active: skill.enabled }" @click="handleToggle(skill)">
            {{ skill.enabled ? 'ON' : 'OFF' }}
          </button>
          <button class="skill-delete" @click="handleDelete(skill)">✕</button>
        </div>
      </div>
    </div>
  </div>

  <ConfirmDeleteDialog
    :visible="showDeleteDialog"
    :server-name="deletingSkill?.name ?? ''"
    :deleting="false"
    entity-type="技能"
    @confirm="confirmDelete"
    @cancel="cancelDelete"
  />

  <div v-if="showInstallDialog" class="dialog-overlay" @click.self="closeInstallDialog">
    <div class="install-dialog">
      <div class="dialog-header">
        <h3 class="dialog-title">安装技能</h3>
      </div>
      <div class="dialog-body">
        <p class="dialog-hint">输入本地技能目录路径，该目录需包含 SKILL.md 文件。</p>
        <input
          v-model="installPath"
          type="text"
          class="install-input"
          placeholder="例如：/home/user/my-skill"
          @keyup.enter="handleInstall"
        />
        <p v-if="installError" class="install-error">{{ installError }}</p>
      </div>
      <div class="dialog-footer">
        <button class="dialog-btn cancel-btn" @click="closeInstallDialog">取消</button>
        <button class="dialog-btn confirm-btn" :disabled="installing" @click="handleInstall">
          {{ installing ? '安装中...' : '安装' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.skills-panel { display: flex; flex-direction: column; height: 100%; }
.skills-toolbar { display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; border-bottom: 1px solid var(--color-border); }
.scope-filter { display: flex; gap: 4px; flex-wrap: wrap; }
.filter-btn { padding: 4px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; transition: all var(--transition-fast); }
.filter-btn:hover { background: var(--color-bg-hover); }
.filter-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.reload-btn { width: 28px; height: 28px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 14px; display: flex; align-items: center; justify-content: center; transition: all var(--transition-fast); flex-shrink: 0; }
.reload-btn:hover:not(:disabled) { background: var(--color-bg-hover); color: var(--color-accent); border-color: var(--color-accent); }
.reload-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.reload-btn.spinning { animation: spin 0.8s linear infinite; }
.add-btn { height: 28px; padding: 0 8px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 11px; display: flex; align-items: center; justify-content: center; transition: all var(--transition-fast); flex-shrink: 0; }
.add-btn:hover { background: var(--color-bg-hover); color: var(--color-accent); border-color: var(--color-accent); }
.toolbar-right { display: flex; align-items: center; gap: 4px; }

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.skills-error { padding: 16px; color: var(--color-error); font-size: 12px; }
.skills-loading { padding: 16px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.skills-empty { padding: 16px; color: var(--color-text-tertiary); font-size: 12px; text-align: center; }
.skills-list { flex: 1; overflow-y: auto; padding: 8px; }
.skill-card { display: flex; align-items: center; justify-content: space-between; padding: 8px; border: 1px solid var(--color-border); border-radius: 6px; margin-bottom: 6px; transition: border-color var(--transition-fast); }
.skill-card:hover { border-color: var(--color-accent); }
.skill-info { display: flex; align-items: center; gap: 8px; min-width: 0; }
.skill-detail { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.skill-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.skill-source { font-size: 11px; color: var(--color-text-tertiary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.skill-actions { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.skill-source-tag {
  font-size: 9px;
  padding: 0 5px;
  border-radius: 3px;
  border: 1px solid;
  font-weight: 500;
  line-height: 1.6;
}

.source-project {
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.25);
  color: #60a5fa;
}

.source-global {
  background: rgba(139, 92, 246, 0.1);
  border-color: rgba(139, 92, 246, 0.25);
  color: #a78bfa;
}
.skill-toggle { width: 36px; height: 24px; border: 1px solid var(--color-border); border-radius: 12px; background: var(--color-bg-tertiary); color: var(--color-text-tertiary); cursor: pointer; font-size: 10px; font-weight: 600; transition: all var(--transition-fast); }
.skill-toggle.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.skill-delete { background: none; border: none; color: var(--color-text-tertiary); cursor: pointer; font-size: 12px; padding: 2px 4px; border-radius: 4px; transition: all var(--transition-fast); }
.skill-delete:hover { color: var(--color-error); background: var(--color-bg-hover); }

.dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.install-dialog { background: var(--color-bg-primary); border: 1px solid var(--color-border); border-radius: 8px; width: 420px; max-width: 90vw; box-shadow: 0 8px 32px rgba(0,0,0,0.3); }
.dialog-header { padding: 16px 20px 0; }
.dialog-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); margin: 0; }
.dialog-body { padding: 12px 20px; }
.dialog-hint { font-size: 12px; color: var(--color-text-tertiary); margin: 0 0 10px; }
.install-input { width: 100%; padding: 8px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-tertiary); color: var(--color-text-primary); font-size: 13px; outline: none; box-sizing: border-box; }
.install-input:focus { border-color: var(--color-accent); }
.install-error { font-size: 11px; color: var(--color-error); margin: 6px 0 0; }
.dialog-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 12px 20px 16px; }
.dialog-btn { padding: 6px 16px; border-radius: 4px; font-size: 13px; cursor: pointer; transition: all var(--transition-fast); border: 1px solid var(--color-border); }
.cancel-btn { background: transparent; color: var(--color-text-secondary); }
.cancel-btn:hover { background: var(--color-bg-hover); }
.confirm-btn { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.confirm-btn:hover:not(:disabled) { opacity: 0.9; }
.confirm-btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>