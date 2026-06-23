import { readonly, ref } from 'vue'

export type ConfirmTone = 'error' | 'primary'

export interface ConfirmOptions {
  title: string
  message?: string
  confirmLabel?: string
  cancelLabel?: string
  tone?: ConfirmTone
}

export interface ConfirmRequest extends ConfirmOptions {
  resolve: (confirmed: boolean) => void
}

// Single active confirmation at a time, read by one globally-mounted host.
// Replaces the per-row `delOverlay[]` booleans scattered across list views.
const activeRequest = ref<ConfirmRequest | null>(null)

export const activeConfirm = readonly(activeRequest)

export const confirm = (options: ConfirmOptions): Promise<boolean> =>
  new Promise<boolean>(resolve => {
    // A superseded request resolves as cancelled so its caller never hangs.
    activeRequest.value?.resolve(false)
    activeRequest.value = { ...options, resolve }
  })

export const resolveActiveConfirm = (confirmed: boolean): void => {
  const request = activeRequest.value

  if (!request) return

  activeRequest.value = null
  request.resolve(confirmed)
}

export const useConfirm = () => ({ confirm })
