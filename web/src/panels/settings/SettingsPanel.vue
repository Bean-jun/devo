<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useUiStore } from '@/stores/ui'
import { useSkillsStore } from '@/stores/skills'
import { useMcpStore } from '@/stores/mcp'
import { API_BASE } from '@/utils/constants'

const uiStore = useUiStore()
const skillsStore = useSkillsStore()
const mcpStore = useMcpStore()

type SubTab = 'project' | 'global'
const activeTab = ref<SubTab>('project')

interface ProjectConfig {
  skills: string[]
  mcp: string[]
}

const config = ref<ProjectConfig>({ skills: [], mcp: [] })
const configLoading = ref(false)

async function fetchConfig() {
  configLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/project/config`)
    if (res.ok) {
      config.value = await res.json()
    }
  } catch {
    // ignore
  } finally {
    configLoading.value = false
  }
}

onMounted(async () => {
  await fetchConfig()
  await skillsStore.fetchSkills()
  await mcpStore.fetchServers()
})
</script>

<template>
  <div class="settings-panel">
    <div class="settings-tabs">
      <button
        class="subtab-btn"
        :class="{ active: activeTab === 'project' }"
        @click="activeTab = 'project'"
      >项目设置</button>
      <button
        class="subtab-btn"
        :class="{ active: activeTab === 'global' }"
        @click="activeTab = 'global'"
      >全局设置</button>
    </div>

    <div v-if="activeTab === 'project'" class="settings-content">
      <div class="setting-section">
        <div class="section-title">工作目录</div>
        <div class="setting-item">
          <span class="setting-value">{{ uiStore.activeWorkspace || '无' }}</span>
        </div>
      </div>

      <div class="setting-section">
        <div class="section-title">技能 (Skills)</div>
        <div v-if="configLoading" class="setting-hint">加载中...</div>
        <div v-else-if="config.skills.length === 0" class="setting-hint">全部启用（使用默认配置）</div>
        <div v-else class="config-list">
          <span
            v-for="name in config.skills"
            :key="name"
            class="config-tag"
            :class="{ active: skillsStore.skills.find(s => s.name === name)?.enabled }"
          >{{ name }}</span>
        </div>
      </div>

      <div class="setting-section">
        <div class="section-title">MCP 服务器</div>
        <div v-if="configLoading" class="setting-hint">加载中...</div>
        <div v-else-if="config.mcp.length === 0" class="setting-hint">无 MCP 服务器配置</div>
        <div v-else class="config-list">
          <span
            v-for="id in config.mcp"
            :key="id"
            class="config-tag"
            :class="{ active: mcpStore.servers.find(s => s.server_id === id)?.status === 'connected' }"
          >{{ id }}</span>
        </div>
      </div>
    </div>

    <div v-else class="settings-content">
      <div class="setting-item">
        <label>主题</label>
        <span class="setting-value">{{ uiStore.theme }}</span>
      </div>
      <div class="setting-item">
        <label>快捷键</label>
        <span class="setting-hint">Ctrl+K 命令面板</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-panel { display: flex; flex-direction: column; height: 100%; }
.settings-tabs { display: flex; gap: 4px; padding: 8px 12px; border-bottom: 1px solid var(--color-border); }
.subtab-btn { padding: 5px 14px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 12px; }
.subtab-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.settings-content { flex: 1; overflow-y: auto; padding: 12px; }
.setting-section { margin-bottom: 16px; }
.section-title { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.5px; }
.setting-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid var(--color-border); }
.setting-item label { font-size: 13px; color: var(--color-text-primary); }
.setting-value { font-size: 12px; color: var(--color-text-secondary); font-family: var(--font-mono); }
.setting-hint { font-size: 12px; color: var(--color-text-tertiary); padding: 8px 0; }
.config-list { display: flex; flex-wrap: wrap; gap: 4px; padding: 4px 0; }
.config-tag { font-size: 11px; padding: 3px 8px; border-radius: 4px; background: var(--color-bg-tertiary); color: var(--color-text-tertiary); border: 1px solid var(--color-border); }
.config-tag.active { background: rgba(59, 130, 246, 0.12); border-color: rgba(59, 130, 246, 0.3); color: var(--color-accent); }
</style>