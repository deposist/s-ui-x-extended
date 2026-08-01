export type GeneratedAdmin = {
  username: string
  password: string
}

export type MigrationReport = {
  summary?: Record<string, unknown>
  warnings?: string[]
  backupPath?: string
  generatedAdmins?: GeneratedAdmin[]
  generated_admins?: GeneratedAdmin[]
}

const generatedAdminAliases = ['generatedAdmins', 'generated_admins'] as const

export function generatedAdminsFromReport(report: MigrationReport | null | undefined): GeneratedAdmin[] {
  const camelAdmins = report?.generatedAdmins
  const snakeAdmins = report?.generated_admins
  if (Array.isArray(camelAdmins) && camelAdmins.length > 0) return camelAdmins
  if (Array.isArray(snakeAdmins)) return snakeAdmins
  return Array.isArray(camelAdmins) ? camelAdmins : []
}

export function eraseGeneratedAdminSecrets(report: MigrationReport | null | undefined): void {
  if (!report) return

  for (const alias of generatedAdminAliases) {
    const admins = report[alias]
    if (!Array.isArray(admins)) continue

    for (const admin of admins) {
      admin.password = ''
    }
    admins.splice(0, admins.length)
  }
}
