import { describe, expect, it } from 'vitest'

import { AWG_DEFAULT_LISTEN_PORT, isLocalHostname, suggestAWGPublicEndpoint } from './awgPublicEndpoint'

describe('suggestAWGPublicEndpoint', () => {
  it('combines the panel hostname with the endpoint listen port', () => {
    expect(suggestAWGPublicEndpoint('hello.watafafa.ru', 38799)).toBe('hello.watafafa.ru:38799')
    expect(suggestAWGPublicEndpoint('203.0.113.10', 51820)).toBe('203.0.113.10:51820')
  })

  it('falls back to the default AWG port when the listen port is unset', () => {
    expect(suggestAWGPublicEndpoint('203.0.113.10', 0)).toBe(`203.0.113.10:${AWG_DEFAULT_LISTEN_PORT}`)
    expect(suggestAWGPublicEndpoint('203.0.113.10', undefined)).toBe(`203.0.113.10:${AWG_DEFAULT_LISTEN_PORT}`)
    expect(suggestAWGPublicEndpoint('203.0.113.10', null)).toBe(`203.0.113.10:${AWG_DEFAULT_LISTEN_PORT}`)
  })

  it('suggests nothing for addresses devices could never reach', () => {
    expect(suggestAWGPublicEndpoint('localhost', 51820)).toBe('')
    expect(suggestAWGPublicEndpoint('127.0.0.1', 51820)).toBe('')
    expect(suggestAWGPublicEndpoint('', 51820)).toBe('')
  })

  it('trims the hostname before combining', () => {
    expect(suggestAWGPublicEndpoint('  vpn.example.com  ', 51820)).toBe('vpn.example.com:51820')
  })
})

describe('isLocalHostname', () => {
  it('matches loopback names case-insensitively', () => {
    expect(isLocalHostname('LOCALHOST')).toBe(true)
    expect(isLocalHostname('::1')).toBe(true)
    expect(isLocalHostname('vpn.example.com')).toBe(false)
  })
})
