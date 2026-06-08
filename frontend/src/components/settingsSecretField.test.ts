import { describe, expect, it } from 'vitest'
import {
  STORED_SECRET_PLACEHOLDER,
  hasStoredSecret,
  normalizeSecretFields,
  stripSecretPlaceholders,
} from '@/components/settingsSecretField'

describe('settings secret field helpers', () => {
  it('detects stored secret markers', () => {
    expect(hasStoredSecret(true)).toBe(true)
    expect(hasStoredSecret('true')).toBe(true)
    expect(hasStoredSecret(false)).toBe(false)
    expect(hasStoredSecret('false')).toBe(false)
    expect(hasStoredSecret(undefined)).toBe(false)
  })

  it('normalizes HasSecret markers to empty editable values', () => {
    const settings = normalizeSecretFields({
      telegramBotTokenHasSecret: 'true',
      telegramProxyPasswordHasSecret: 'false',
      telegramBackupPassphraseHasSecret: 'true',
      telegramBackupPassphrase: STORED_SECRET_PLACEHOLDER,
    })

    expect(settings.telegramBotToken).toBe('')
    expect(settings.telegramProxyPassword).toBe('')
    expect(settings.telegramBackupPassphrase).toBe(STORED_SECRET_PLACEHOLDER)
  })

  it('does not submit the stored placeholder as a secret value', () => {
    const settings = stripSecretPlaceholders({
      telegramBotTokenHasSecret: 'true',
      telegramBotToken: STORED_SECRET_PLACEHOLDER,
      telegramChatID: '42',
    })

    expect(settings.telegramBotToken).toBe('')
    expect(settings.telegramChatID).toBe('42')
  })

  it('keeps the Telegram backup stored marker so save does not clear it', () => {
    const settings = stripSecretPlaceholders({
      telegramBackupPassphraseHasSecret: 'true',
      telegramBackupPassphrase: STORED_SECRET_PLACEHOLDER,
    })

    expect(settings.telegramBackupPassphrase).toBe(STORED_SECRET_PLACEHOLDER)
  })
})
