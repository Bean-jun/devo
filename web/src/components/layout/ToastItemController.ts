import type { Toast } from '@/stores/ui'

export interface ToastItemProps {
  toast: Toast
}

export function useToastItem(props: ToastItemProps, emit: (e: string, ...args: any[]) => void) {
  const iconMap = {
    success: { name: 'check-circle', color: 'var(--color-success)' },
    error: { name: 'x-circle', color: 'var(--color-error)' },
    info: { name: 'info', color: 'var(--color-accent)' },
    warning: { name: 'warning', color: 'var(--color-warning)' },
  } as const

  return { iconMap }
}