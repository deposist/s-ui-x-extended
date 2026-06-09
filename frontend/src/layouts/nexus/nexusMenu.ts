// The navigation menu now lives in a single shared source (@/layouts/menu)
// consumed by both the classic and nexus shells. These re-exports preserve
// the existing nexus-facing names for backward compatibility.
export type { MenuItem as NexusMenuItem } from '@/layouts/menu'
import { appMenu, singBoxSettingsPaths } from '@/layouts/menu'

export const nexusMenu = appMenu
export const nexusSingBoxSettingsPaths = singBoxSettingsPaths
