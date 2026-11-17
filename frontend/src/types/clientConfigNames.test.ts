import { describe, expect, it } from 'vitest'
import { updateConfigs } from './clients'

describe('client config names', () => {
  it('renames configs with a top-level hasOwnProperty entry', () => {
    const configs = JSON.parse('{"hasOwnProperty":{"name":"old"},"vless":{"name":"old"}}')
    expect(() => updateConfigs(configs, 'new')).not.toThrow()
    expect(configs.vless.name).toBe('new')
    expect(configs.hasOwnProperty.name).toBe('new')
  })

  it('renames configs containing a hasOwnProperty field', () => {
    const configs = { vless: { name: 'old', hasOwnProperty: 'preserved' } }
    expect(() => updateConfigs(configs, 'new')).not.toThrow()
    expect(configs.vless).toEqual({ name: 'new', hasOwnProperty: 'preserved' })
  })
})
