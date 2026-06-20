import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import type { Session } from '@/types/session'

export function useSession() {
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()

  async function createAndSwitch(name?: string): Promise<Session | null> {
    try {
      const session = await sessionStore.createSession({
        title: name,
        workingDirectory: sessionStore.workingDirectory,
      })
      uiStore.showToast('success', `会话 "${session.title}" 创建成功`)
      return session
    } catch (err) {
      uiStore.showToast('error', `创建会话失败: ${(err as Error).message}`)
      return null
    }
  }

  async function switchTo(id: string): Promise<boolean> {
    try {
      const ok = await sessionStore.switchSessionById(id)
      if (ok) {
        uiStore.showToast('info', '已切换会话')
      } else {
        uiStore.showToast('error', '会话不存在')
      }
      return ok
    } catch (err) {
      uiStore.showToast('error', `切换会话失败: ${(err as Error).message}`)
      return false
    }
  }

  async function rename(id: string, name: string): Promise<void> {
    try {
      await sessionStore.renameSession(id, name)
      uiStore.showToast('success', '会话已重命名')
    } catch (err) {
      uiStore.showToast('error', `重命名失败: ${(err as Error).message}`)
    }
  }

  async function archive(id: string): Promise<void> {
    try {
      await sessionStore.archiveSession(id)
      uiStore.showToast('info', '会话已归档')
    } catch (err) {
      uiStore.showToast('error', `归档失败: ${(err as Error).message}`)
    }
  }

  return { createAndSwitch, switchTo, rename, archive }
}