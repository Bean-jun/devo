import { ref, computed, onMounted } from 'vue'
import { useSkillsStore } from '@/stores/skills'
import type { Skill, SkillSource } from '@/types/skills'
import ConfirmDeleteDialog from '@/components/modal/ConfirmDeleteDialog.vue'
import AppIcon from '@/components/common/AppIcon.vue'

type SourceFilter = 'all' | SkillSource

function sourceLabel(source: string): string {
  switch (source) {
    case 'project': return '项目'
    case 'global': return '全局'
    default: return source
  }
}

export function useSkillsPanel() {
  const skillsStore = useSkillsStore()

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

  onMounted(async () => {
    try {
      await skillsStore.fetchSkills()
    } catch (e: any) {
      error.value = e.message
    }
  })

  return {
    skillsStore,
    sourceFilter,
    error,
    showDeleteDialog,
    deletingSkill,
    showInstallDialog,
    installPath,
    installing,
    installError,
    filteredSkills,
    handleToggle,
    handleDelete,
    confirmDelete,
    cancelDelete,
    handleReload,
    openInstallDialog,
    closeInstallDialog,
    handleInstall,
    sourceLabel,
  }
}