<script setup lang="ts">
import { useSettingsPanel } from './SettingsPanelController'
import AppIcon from '@/components/common/AppIcon.vue'
import AddModelForm from '@/components/common/AddModelForm.vue'

const {
  activeTab,
  config,
  configLoading,
  uiStore,
  skillsStore,
  mcpStore,
  toolCallLimit,
  maxContextTokens,
  keepRecent,
  globalToolCallLimit,
  globalMaxContextTokens,
  globalKeepRecent,
  globalMaxTokens,
  APPROVAL_OPERATIONS,
  APPROVAL_LEVELS,
  RISK_LABELS,
  projectApprovalPolicy,
  globalApprovalPolicy,
  getProjectPolicyLevel,
  getGlobalPolicyLevel,
  isDefaultPolicy,
  handleProjectApprovalChange,
  handleGlobalApprovalChange,
  saveProjectParams,
  saveGlobalParams,
  modelStore,
  showAddModelForm,
  testingModelId,
  testResult,
  handleModelActivate,
  handleModelDelete,
  handleModelTest,
  openAddModelForm,
  onModelAdded,
  teamMode,
  teamModeLoading,
  handleTeamModeToggle,
} = useSettingsPanel()
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
        <div class="section-title">LLM 参数</div>
        <div class="setting-item">
          <label>工具调用上限</label>
          <input
            v-model.number="toolCallLimit"
            type="number"
            min="1"
            placeholder="50"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>最大上下文 Tokens</label>
          <input
            v-model.number="maxContextTokens"
            type="number"
            min="1"
            placeholder="128000"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>保留最近轮数</label>
          <input
            v-model.number="keepRecent"
            type="number"
            min="1"
            placeholder="30"
            class="setting-input"
          />
        </div>
        <button class="save-btn" @click="saveProjectParams">保存参数</button>
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

      <div class="setting-section">
        <div class="section-title">审批策略</div>
        <div class="approval-table">
          <div
            v-for="op in APPROVAL_OPERATIONS"
            :key="op.key"
            class="approval-row"
            :class="`risk-${op.risk}`"
          >
            <div class="approval-info">
              <AppIcon :name="op.icon" :size="16" class="approval-icon" />
              <span class="approval-label">{{ op.label }}</span>
              <span class="risk-badge" :class="`risk-${op.risk}`">{{ RISK_LABELS[op.risk] }}</span>
              <span v-if="isDefaultPolicy(projectApprovalPolicy, op.key)" class="approval-default">默认</span>
            </div>
            <div class="approval-pills">
              <button
                v-for="lv in APPROVAL_LEVELS"
                :key="lv.key"
                class="pill-btn"
                :class="{ active: (getProjectPolicyLevel(op.key) || op.defaultLevel) === lv.key }"
                @click="handleProjectApprovalChange(op.key, lv.key)"
              >{{ lv.short }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="settings-content">
      <div class="setting-section">
        <div class="section-title">Team Mode</div>
        <div class="setting-item setting-item-row">
          <div>
            <label>多 Agent 协作</label>
            <div class="setting-hint">开启后主 Agent 可将任务委托给子 Agent</div>
          </div>
          <button
            class="toggle-btn"
            :class="{ active: teamMode }"
            :disabled="teamModeLoading"
            @click="handleTeamModeToggle"
          >
            <span class="toggle-knob" />
            <span class="toggle-label">{{ teamMode ? '开' : '关' }}</span>
          </button>
        </div>
      </div>

      <div class="setting-section">
        <div class="section-title">LLM 参数（全局默认）</div>
        <div class="setting-item">
          <label>工具调用上限</label>
          <input
            v-model.number="globalToolCallLimit"
            type="number"
            min="1"
            placeholder="50"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>最大上下文 Tokens</label>
          <input
            v-model.number="globalMaxContextTokens"
            type="number"
            min="1"
            placeholder="128000"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>保留最近轮数</label>
          <input
            v-model.number="globalKeepRecent"
            type="number"
            min="1"
            placeholder="30"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>最大输出 Tokens</label>
          <input
            v-model.number="globalMaxTokens"
            type="number"
            min="1"
            placeholder="（服务端默认值）"
            class="setting-input"
          />
        </div>
        <button class="save-btn" @click="saveGlobalParams">保存全局参数</button>
      </div>

      <div class="setting-section">
        <div class="section-title">模型管理</div>
        <div v-if="modelStore.isLoading" class="setting-hint">加载中...</div>
        <div v-else>
          <div v-if="modelStore.models.length === 0" class="setting-hint">
            暂无模型配置，请添加模型
          </div>
          <div v-else class="model-list">
            <div
              v-for="m in modelStore.models"
              :key="m.id"
              class="model-card"
              :class="{ active: m.id === modelStore.activeModelId }"
            >
              <div class="model-card-info">
                <div class="model-card-name">
                  {{ m.name }}
                  <span v-if="m.id === modelStore.activeModelId" class="model-card-badge">当前</span>
                </div>
                <div class="model-card-detail">{{ m.model }} @ {{ m.base_url }}</div>
              </div>
              <div class="model-card-actions">
                <button
                  v-if="m.id !== modelStore.activeModelId"
                  class="model-action-btn activate"
                  @click="handleModelActivate(m.id)"
                >激活</button>
                <button
                  class="model-action-btn test"
                  :disabled="testingModelId === m.id"
                  @click="handleModelTest(m.id)"
                >{{ testingModelId === m.id ? '测试中...' : '测试' }}</button>
                <button
                  class="model-action-btn delete"
                  @click="handleModelDelete(m.id)"
                >删除</button>
              </div>
              <div
                v-if="testResult && testResult.id === m.id"
                class="model-test-result"
                :class="{ success: testResult.success, fail: !testResult.success }"
              >{{ testResult.success ? '连接成功' : '连接失败: ' + testResult.error }}</div>
            </div>
          </div>
          <button class="add-model-btn" @click="openAddModelForm">+ 添加模型</button>
        </div>
      </div>

      <div v-if="showAddModelForm" class="model-form-overlay">
        <AddModelForm mode="modal" @submit="onModelAdded" @cancel="showAddModelForm = false" />
      </div>

      <div class="setting-item">
        <label>主题</label>
        <span class="setting-value">{{ uiStore.theme }}</span>
      </div>
      <div class="setting-item">
        <label>快捷键</label>
        <span class="setting-hint">Ctrl+K 命令面板</span>
      </div>

      <div class="setting-section">
        <div class="section-title">审批策略</div>
        <div class="approval-table">
          <div
            v-for="op in APPROVAL_OPERATIONS"
            :key="op.key"
            class="approval-row"
            :class="`risk-${op.risk}`"
          >
            <div class="approval-info">
              <AppIcon :name="op.icon" :size="16" class="approval-icon" />
              <span class="approval-label">{{ op.label }}</span>
              <span class="risk-badge" :class="`risk-${op.risk}`">{{ RISK_LABELS[op.risk] }}</span>
              <span v-if="isDefaultPolicy(globalApprovalPolicy, op.key)" class="approval-default">默认</span>
            </div>
            <div class="approval-pills">
              <button
                v-for="lv in APPROVAL_LEVELS"
                :key="lv.key"
                class="pill-btn"
                :class="{ active: (getGlobalPolicyLevel(op.key) || op.defaultLevel) === lv.key }"
                @click="handleGlobalApprovalChange(op.key, lv.key)"
              >{{ lv.short }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped src="./SettingsPanel.css">
</style>