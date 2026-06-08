import { describe, expect, it } from 'vitest'

import {
  addAdminPasswordsMatch,
  isAddAdminFormComplete,
  isAddAdminFormValid,
  isDeleteAdminFormValid,
  normalizeAdminUsername,
} from './adminForms'

describe('admin forms', () => {
  it('normalizes usernames and validates add admin fields', () => {
    const form = {
      currentPass: 'current',
      username: '  new-admin  ',
      password: 'secret',
      confirmPassword: 'secret',
    }

    expect(normalizeAdminUsername(form.username)).toBe('new-admin')
    expect(isAddAdminFormComplete(form)).toBe(true)
    expect(addAdminPasswordsMatch(form)).toBe(true)
    expect(isAddAdminFormValid(form)).toBe(true)
  })

  it('rejects incomplete add forms and mismatched passwords', () => {
    expect(isAddAdminFormValid({
      currentPass: 'current',
      username: 'new-admin',
      password: 'secret',
      confirmPassword: 'different',
    })).toBe(false)
    expect(isAddAdminFormComplete({
      currentPass: 'current',
      username: '   ',
      password: 'secret',
      confirmPassword: 'secret',
    })).toBe(false)
  })

  it('requires current password before delete', () => {
    expect(isDeleteAdminFormValid({ currentPass: '' })).toBe(false)
    expect(isDeleteAdminFormValid({ currentPass: 'current' })).toBe(true)
  })
})
