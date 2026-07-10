<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useSkillsStore } from '@/stores/skills'
import { useMcpStore } from '@/stores/mcp'
import { API_BASE } from '@/utils/constants'
import AppIcon from '@/components/common/AppIcon.vue'

const skillsStore = useSkillsStore()
const mcpStore = useMcpStore()

interface ProjectConfig {
  skills: string[]
  mcp: string[]
}

const config = ref<ProjectConfig>({ skills: [], mcp: [] })
const loading = ref(true)
const error = ref<string | null>(null)

async function fetchConfig() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/project/config`)
    if (!res.ok) throw new Error('Failed to load config')
    config.value = await res.json()
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

const enabledSkills = computed(() => {
  return skillsStore.skills.filter(s => s.enabled)
})

const disabledSkills = computed(() => {
  return skillsStore.skills.filter(s => !s.enabled)
})

const connectedServers = computed(() => {
  return mcpStore.servers.filter(s => s.status === 'connected')
})

const disconnectedServers = computed(() => {
  return mcpStore.servers.filter(s => s.status !== 'connected')
})

onMounted(async () => {
  await Promise.all([
    fetchConfig(),
    skillsStore.fetchSkills(),
    mcpStore.fetchServers(),
  ])
})
</script>

<template>
  <div class="project-settings-view">
    <h2>项目设置</h2>
    <p class="subtitle">当前项目的 Skills 和 MCP 配置</p>

    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="error" class="error-banner">{{ error }}</div>

    <div v-else class="settings-grid">
      <section class="config-section">
        <h3 class="section-title">
          <AppIcon name="lightning" :size="16" class="section-icon" />
          启用中的技能
          <span class="badge">{{ enabledSkills.length }}</span>
        </h3>
        <div v-if="enabledSkills.length === 0" class="empty-hint">暂无启用技能</div>
        <div v-else class="item-list">
          <div v-for="skill in enabledSkills" :key="skill.name" class="item-row">
            <span class="item-name">{{ skill.name }}</span>
            <span class="item-source">{{ skill.source }}</span>
          </div>
        </div>
      </section>

      <section class="config-section">
        <h3 class="section-title">
          <AppIcon name="plug" :size="16" class="section-icon" />
          已连接的 MCP 服务器
          <span class="badge">{{ connectedServers.length }}</span>
        </h3>
        <div v-if="connectedServers.length === 0" class="empty-hint">暂无连接的 MCP 服务器</div>
        <div v-else class="item-list">
          <div v-for="server in connectedServers" :key="server.server_id" class="item-row">
            <span class="item-name">{{ server.server_id }}</span>
            <span class="item-source">{{ server.tool_count }} 工具</span>
          </div>
        </div>
      </section>

      <section class="config-section">
        <h3 class="section-title">
          <AppIcon name="moon" :size="16" class="section-icon" />
          已禁用的技能
          <span class="badge muted">{{ disabledSkills.length }}</span>
        </h3>
        <div v-if="disabledSkills.length === 0" class="empty-hint">无禁用技能</div>
        <div v-else class="item-list">
          <div v-for="skill in disabledSkills" :key="skill.name" class="item-row muted">
            <span class="item-name">{{ skill.name }}</span>
            <span class="item-source">{{ skill.source }}</span>
          </div>
        </div>
      </section>

      <section class="config-section">
        <h3 class="section-title">
          <AppIcon name="pause" :size="16" class="section-icon" />
          未连接的 MCP 服务器
          <span class="badge muted">{{ disconnectedServers.length }}</span>
        </h3>
        <div v-if="disconnectedServers.length === 0" class="empty-hint">无未连接的 MCP 服务器</div>
        <div v-else class="item-list">
          <div v-for="server in disconnectedServers" :key="server.server_id" class="item-row muted">
            <span class="item-name">{{ server.server_id }}</span>
            <span class="item-source">{{ server.source }}</span>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.project-settings-view {
  padding: 24px;
  max-width: 900px;
  margin: 0 auto;
}

.project-settings-view h2 {
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 4px;
}

.subtitle {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin: 0 0 20px;
}

.loading {
  text-align: center;
  padding: 40px;
  color: var(--color-text-tertiary);
}

.error-banner {
  padding: 12px 16px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 6px;
  color: var(--color-error);
  font-size: 13px;
  margin-bottom: 16px;
}

.settings-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

@media (max-width: 640px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }
}

.config-section {
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.section-icon {
  font-size: 16px;
}

.badge {
  font-size: 11px;
  padding: 1px 8px;
  border-radius: 10px;
  background: rgba(59, 130, 246, 0.12);
  color: var(--color-accent);
  font-weight: 500;
  margin-left: auto;
}

.badge.muted {
  background: var(--color-bg-tertiary);
  color: var(--color-text-tertiary);
}

.empty-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
  padding: 8px 0;
}

.item-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.item-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  border-radius: 4px;
  background: var(--color-bg-secondary);
  font-size: 13px;
}

.item-row.muted {
  opacity: 0.6;
}

.item-name {
  color: var(--color-text-primary);
  font-weight: 500;
}

.item-source {
  font-size: 11px;
  color: var(--color-text-tertiary);
}
</style>