import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { createInbound, InTypes, type Call, type Shadowsocks, type ShadowTLS, type Sudoku } from './inbounds'

beforeEach(() => {
  vi.stubGlobal('window', { crypto: globalThis.crypto })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('createInbound generated secrets', () => {
  it('generates a secure Shadowsocks method and password for new inbounds', () => {
    const inbound = createInbound(InTypes.Shadowsocks, { id: 0, tag: 'ss', listen_port: 10000 }) as Shadowsocks

    expect(inbound.method).toBe('2022-blake3-aes-256-gcm')
    expect(inbound.password).toBeTruthy()
    expect(inbound.password).not.toBe('')
  })

  it('preserves explicit Shadowsocks method and password', () => {
    const inbound = createInbound(InTypes.Shadowsocks, {
      id: 0,
      tag: 'ss',
      listen_port: 10000,
      method: 'none',
      password: '',
    }) as Shadowsocks

    expect(inbound.method).toBe('none')
    expect(inbound.password).toBe('')

    const explicitEmpty = createInbound(InTypes.Shadowsocks, {
      id: 0,
      tag: 'ss-empty',
      listen_port: 10001,
      method: '',
    } as Partial<Shadowsocks>) as Shadowsocks

    expect(explicitEmpty.method).toBe('')
    expect(explicitEmpty.password).toBeUndefined()
  })

  it('leaves a missing Sudoku master key for backend generation', () => {
    const sudoku = createInbound(InTypes.Sudoku, { id: 0, tag: 'sudoku', listen_port: 10001 }) as Sudoku
    const shadowTlsV3 = createInbound(InTypes.ShadowTLS, { id: 0, tag: 'shadowtls', listen_port: 10002 }) as ShadowTLS
    const shadowTlsV2 = createInbound(InTypes.ShadowTLS, { id: 0, tag: 'shadowtls-v2', listen_port: 10003, version: 2 }) as ShadowTLS

    expect(sudoku.key).toBeUndefined()
    expect(shadowTlsV3.version).toBe(3)
    expect(shadowTlsV3.password).toBeUndefined()
    expect(shadowTlsV2.password).toBeTruthy()

    const existingSudoku = createInbound(InTypes.Sudoku, { id: 0, tag: 'sudoku', listen_port: 10004, key: '' }) as Sudoku
    const existingShadowTls = createInbound(InTypes.ShadowTLS, { id: 0, tag: 'shadowtls', listen_port: 10005, version: 2, password: '' }) as ShadowTLS

    expect(existingSudoku.key).toBe('')
    expect(existingShadowTls.password).toBe('')
  })

  it('creates a call inbound with platform defaults', () => {
    const call = createInbound(InTypes.Call, { id: 0, tag: 'call', listen_port: 10010 }) as Call

    expect(call.type).toBe('call')
    expect(call.platform).toBe('dion')
    expect(call.read_buffer).toBe(32768)
    expect(call.join_link).toBeUndefined()
    expect(call.cookies).toBeUndefined()
  })
})
