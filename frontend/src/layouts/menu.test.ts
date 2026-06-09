import { describe, expect, it } from 'vitest'

import { appMenu, singBoxSettingsPaths } from './menu'

const EXPECTED_PATHS = [
  '/',
  '/inbounds',
  '/clients',
  '/outbounds',
  '/endpoints',
  '/providers',
  '/services',
  '/tls',
  '/basics',
  '/rules',
  '/dns',
  '/admins',
  '/telegram',
  '/paid-subscriptions',
  '/audit',
  '/settings',
]

describe('shared app menu', () => {
  it('is the single source consumed by both classic and nexus shells', () => {
    expect(appMenu.map(item => item.path)).toEqual(EXPECTED_PATHS)
  })

  it('keeps both previously-diverging tabs present (no drift back)', () => {
    const paths = appMenu.map(item => item.path)

    expect(paths).toContain('/providers')
    expect(paths).toContain('/paid-subscriptions')
  })

  it('keeps each menu path unique', () => {
    const paths = appMenu.map(item => item.path)

    expect(new Set(paths).size).toBe(paths.length)
  })

  it('marks the sing-box editor surfaces for the nexus shell', () => {
    expect(singBoxSettingsPaths).toEqual([
      '/inbounds',
      '/outbounds',
      '/endpoints',
      '/services',
      '/tls',
      '/basics',
      '/rules',
      '/dns',
    ])
  })
})
