import { describe, expect, it } from 'vitest'

import {
  eraseGeneratedAdminSecrets,
  generatedAdminsFromReport,
  type GeneratedAdmin,
  type MigrationReport,
} from './migrateXuiSecrets'

describe('migrate XUI generated-admin secrets', () => {
  it('reads either report alias without widening generated-admin records', () => {
    const camelAdmin: GeneratedAdmin = { username: 'camel-admin', password: 'camel-secret' }
    const snakeAdmin: GeneratedAdmin = { username: 'snake-admin', password: 'snake-secret' }

    expect(generatedAdminsFromReport({ generatedAdmins: [camelAdmin] })).toEqual([camelAdmin])
    expect(generatedAdminsFromReport({ generated_admins: [snakeAdmin] })).toEqual([snakeAdmin])
  })

  it('erases both aliases and scrubs secrets from retained record references', () => {
    const camelAdmin: GeneratedAdmin = { username: 'camel-admin', password: 'camel-secret' }
    const snakeAdmin: GeneratedAdmin = { username: 'snake-admin', password: 'snake-secret' }
    const camelAdmins = [camelAdmin]
    const snakeAdmins = [snakeAdmin]
    const report: MigrationReport = {
      generatedAdmins: camelAdmins,
      generated_admins: snakeAdmins,
    }

    eraseGeneratedAdminSecrets(report)

    expect(report.generatedAdmins).toEqual([])
    expect(report.generated_admins).toEqual([])
    expect(camelAdmins).toEqual([])
    expect(snakeAdmins).toEqual([])
    expect(camelAdmin.password).toBe('')
    expect(snakeAdmin.password).toBe('')
    expect(JSON.stringify({ report, camelAdmin, snakeAdmin })).not.toContain('camel-secret')
    expect(JSON.stringify({ report, camelAdmin, snakeAdmin })).not.toContain('snake-secret')
  })
})
