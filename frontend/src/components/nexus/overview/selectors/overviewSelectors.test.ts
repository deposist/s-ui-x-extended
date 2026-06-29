import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  auditDisplayIcons,
  mapAuditDisplayItem,
  mapAuditDisplayItems,
} from './auditMapper'
import { selectKpiSummary } from './kpiSelectors'
import { selectProtocolSummaries } from './protocolSummarySelectors'
import { selectSystemStatus } from './systemStatusSelectors'
import { selectTopClients } from './topClientsSelectors'
import {
  formatTrafficLabel,
  loadTrafficTimeZone,
  persistTrafficTimeZone,
  selectTrafficSeries,
  trafficTimeZoneOptions,
  trafficTimeZoneStorageKey,
} from './trafficSelectors'

const originalSupportedValuesOf = Intl.supportedValuesOf

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  Object.defineProperty(Intl, 'supportedValuesOf', {
    configurable: true,
    value: originalSupportedValuesOf,
  })
})

describe('overview selectors', () => {
  it('returns empty-safe defaults', () => {
    expect(selectKpiSummary()).toEqual({
      liveTrafficBps: 0,
      onlineClients: 0,
      activeInbounds: 0,
      totalInbounds: 0,
      health: 'degraded',
    })
    expect(selectTrafficSeries()).toEqual({
      labels: [],
      download: [],
      upload: [],
      range: '24h',
    })
    expect(selectSystemStatus()).toEqual({
      ipv4: [],
      ipv6: [],
      appVersion: '',
      bootTime: 0,
      uptimeSec: 0,
      singboxRunning: false,
    })
    expect(selectTopClients()).toEqual([])
    expect(selectProtocolSummaries()).toEqual([])
    expect(mapAuditDisplayItems()).toEqual([])
  })

  it('maps typical overview data without mutating traffic sources', () => {
    expect(selectKpiSummary({
      inbounds: [
        { tag: 'vless-in' },
        { tag: 'disabled-in', enable: false },
        { tag: 'trojan-in' },
      ],
      onlines: {
        inbound: ['vless-in'],
        user: ['ada', 'lin', 'ada'],
      },
      liveTraffic: {
        downloadBps: 3500,
        uploadBps: 1500,
      },
      health: {
        online: true,
        singboxRunning: true,
      },
    })).toEqual({
      liveTrafficBps: 5000,
      onlineClients: 2,
      activeInbounds: 1,
      totalInbounds: 2,
      health: 'healthy',
    })

    const trafficStats = [
      { dateTime: 1710000060, direction: true, traffic: 7 },
      { dateTime: 1710000000, direction: false, traffic: 11 },
      { dateTime: 1710000000, direction: true, traffic: 5 },
      { dateTime: 1710000000, direction: true, traffic: 3 },
    ]
    const originalTrafficStats = trafficStats.map((stat) => ({ ...stat }))

    expect(selectTrafficSeries({ stats: trafficStats, range: '7d', timeZone: 'UTC' })).toEqual({
      labels: [
        '2024-03-09 16:00',
        '2024-03-09 16:01',
      ],
      download: [11, 0],
      upload: [8, 7],
      range: '7d',
    })
    expect(trafficStats).toEqual(originalTrafficStats)

    expect(selectSystemStatus({
      sys: {
        ipv4: ['192.0.2.10/24'],
        ipv6: ['2001:db8::10/64'],
        appVersion: '1.6.0',
        bootTime: 1710000000,
      },
      sbd: {
        running: true,
        version: '1.11.0',
        stats: {
          Alloc: 4096,
          Uptime: 73,
        },
      },
    }, 1710000120)).toEqual({
      ipv4: ['192.0.2.10/24'],
      ipv6: ['2001:db8::10/64'],
      appVersion: '1.6.0',
      bootTime: 1710000000,
      uptimeSec: 120,
      singboxRunning: true,
      singboxVersion: '1.11.0',
      singboxAlloc: 4096,
      singboxUptimeSec: 73,
    })
  })

  it('selects top client rows and grouped inbound protocol summaries', () => {
    const input = {
      clients: [
        { id: 2, name: 'lin', totalUp: 40, totalDown: 20, up: 5, down: 5 },
        { id: 1, name: 'ada', totalUp: 50, totalDown: 50, up: 10, down: 10 },
        { id: 3, name: 'ken', totalUp: 1, totalDown: 1, up: 0, down: 0 },
      ],
      onlines: {
        user: ['lin'],
      },
    }

    expect(selectTopClients(input, 2)).toEqual([
      {
        id: 1,
        name: 'ada',
        upload: 60,
        download: 60,
        total: 120,
        online: false,
      },
      {
        id: 2,
        name: 'lin',
        upload: 45,
        download: 25,
        total: 70,
        online: true,
      },
    ])
    expect(input.clients.map((client) => client.name)).toEqual(['lin', 'ada', 'ken'])

    expect(selectProtocolSummaries({
      inbounds: [
        { type: 'vless', tag: 'front-door' },
        { type: 'trojan', tag: 'edge' },
        { type: 'vless', tag: 'workers' },
      ],
      onlines: {
        inbound: ['front-door', 'edge'],
      },
    })).toEqual([
      {
        type: 'trojan',
        activeInbounds: 1,
        totalInbounds: 1,
        tags: ['edge'],
      },
      {
        type: 'vless',
        activeInbounds: 1,
        totalInbounds: 2,
        tags: ['front-door', 'workers'],
      },
    ])
  })

  it('uses exact traffic summary buckets from the dashboard stats endpoint', () => {
    expect(selectTrafficSeries({
      range: '24h',
      summary: {
        startTime: 1710000000,
        endTime: 1710003600,
        download: 40,
        upload: 9,
        buckets: [
          { startTime: 1710000000, endTime: 1710001800, download: 10, upload: 4 },
          { startTime: 1710001800, endTime: 1710003600, download: 30, upload: 5 },
        ],
      },
      bucketCount: 48,
      stats: [
        { dateTime: 1710000000, direction: false, traffic: 999 },
      ],
      timeZone: 'UTC',
    })).toEqual({
      labels: [
        '2024-03-09 16:00',
        '2024-03-09 16:30',
      ],
      download: [10, 30],
      upload: [4, 5],
      range: '24h',
    })
  })

  it('buckets legacy traffic stats into the selected dashboard range', () => {
    expect(selectTrafficSeries({
      range: '1h',
      bucketCount: 4,
      nowMs: 1710003600 * 1000,
      stats: [
        { dateTime: 1710000060, direction: false, traffic: 10 },
        { dateTime: 1710000120, direction: true, traffic: 4 },
        { dateTime: 1710001800, direction: false, traffic: 6 },
        { dateTime: 1709999900, direction: false, traffic: 99 },
      ],
      timeZone: 'UTC',
    })).toEqual({
      labels: [
        '2024-03-09 16:00',
        '2024-03-09 16:15',
        '2024-03-09 16:30',
        '2024-03-09 16:45',
      ],
      download: [10, 0, 6, 0],
      upload: [4, 0, 0, 0],
      range: '1h',
    })
  })

  it('formats and persists traffic timezone labels safely', () => {
    expect(formatTrafficLabel(1710000000, 'UTC')).toBe('2024-03-09 16:00')
    expect(formatTrafficLabel(1710000000, 'Europe/Moscow')).toBe('2024-03-09 19:00')

    const storage = (() => {
      const values = new Map<string, string>()
      return {
        getItem: vi.fn((key: string) => values.get(key) ?? null),
        setItem: vi.fn((key: string, value: string) => values.set(key, value)),
        clear: vi.fn(() => values.clear()),
        key: vi.fn(),
        length: 0,
        removeItem: vi.fn((key: string) => values.delete(key)),
      } satisfies Storage
    })()

    persistTrafficTimeZone('Europe/Moscow', storage)
    expect(storage.setItem).toHaveBeenCalledWith(trafficTimeZoneStorageKey, 'Europe/Moscow')
    expect(loadTrafficTimeZone(storage)).toBe('Europe/Moscow')

    storage.setItem(trafficTimeZoneStorageKey, 'Invalid/Zone')
    expect(loadTrafficTimeZone(storage)).toEqual(expect.any(String))
  })

  it('builds a short curated default traffic timezone list without reading the full IANA list', () => {
    const supportedValuesOf = vi.fn(() => [
      'Africa/Abidjan',
      'America/Adak',
      'Asia/Yerevan',
      'Pacific/Chatham',
    ])
    vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions').mockReturnValue({
      locale: 'en-US',
      calendar: 'gregory',
      numberingSystem: 'latn',
      timeZone: 'Europe/Paris',
    } as Intl.ResolvedDateTimeFormatOptions)
    Object.defineProperty(Intl, 'supportedValuesOf', {
      configurable: true,
      value: supportedValuesOf,
    })

    const options = trafficTimeZoneOptions()
    const values = options.map(option => option.value)

    expect(supportedValuesOf).not.toHaveBeenCalled()
    expect(options.length).toBeLessThan(20)
    expect(values).toEqual(expect.arrayContaining([
      'UTC',
      'Europe/Paris',
      'Europe/Moscow',
      'Asia/Kolkata',
      'America/New_York',
    ]))
    expect(options.every(option => option.label.startsWith(`${option.value} (UTC `))).toBe(true)
  })

  it('filters the full supported IANA traffic timezone list only when searching', () => {
    const supportedValuesOf = vi.fn(() => [
      'Europe/Moscow',
      'Asia/Kolkata',
      'America/New_York',
      'Pacific/Chatham',
    ])
    Object.defineProperty(Intl, 'supportedValuesOf', {
      configurable: true,
      value: supportedValuesOf,
    })

    expect(trafficTimeZoneOptions('UTC', 'KOL')).toEqual([
      { value: 'Asia/Kolkata', label: 'Asia/Kolkata (UTC +5:30)' },
    ])
    expect(supportedValuesOf).toHaveBeenCalledWith('timeZone')
  })

  it('includes a valid selected traffic timezone outside the curated defaults', () => {
    const supportedValuesOf = vi.fn(() => ['Pacific/Chatham'])
    Object.defineProperty(Intl, 'supportedValuesOf', {
      configurable: true,
      value: supportedValuesOf,
    })

    const options = trafficTimeZoneOptions('Africa/Abidjan')
    const values = options.map(option => option.value)

    expect(supportedValuesOf).not.toHaveBeenCalled()
    expect(values.filter(value => value === 'Africa/Abidjan')).toHaveLength(1)
    expect(options).toContainEqual({
      value: 'Africa/Abidjan',
      label: 'Africa/Abidjan (UTC +0)',
    })
  })

  it('filters curated fallback traffic timezones when full IANA support is unavailable', () => {
    Object.defineProperty(Intl, 'supportedValuesOf', {
      configurable: true,
      value: undefined,
    })

    expect(trafficTimeZoneOptions(undefined, 'new')).toEqual(expect.arrayContaining([
      expect.objectContaining({
        value: 'America/New_York',
      }),
    ]))
  })

  it('maps known and unknown audit or partial API payloads to plain display data', () => {
    expect(mapAuditDisplayItem({
      id: 12,
      dateTime: 1710000000,
      actor: '<admin>',
      event: 'login_success',
      resource: 'auth',
      severity: 'info',
      details: {
        ignored: '<b>not displayed</b>',
      },
    })).toEqual({
      id: 12,
      timestamp: 1710000000,
      icon: 'mdi-login',
      tone: 'success',
      text: 'Login succeeded',
      detail: 'actor: admin; resource: auth',
    })

    expect(mapAuditDisplayItem({
      id: 13,
      dateTime: 1710000030,
      actor: 'admin',
      event: 'admin_created',
      resource: 'admin',
      severity: 'warn',
    })).toEqual({
      id: 13,
      timestamp: 1710000030,
      icon: 'mdi-account-plus-outline',
      tone: 'warning',
      text: 'Admin created',
      detail: 'actor: admin; resource: admin',
    })

    expect(mapAuditDisplayItem({
      id: 14,
      dateTime: 1710000035,
      actor: 'admin',
      event: 'admin_deleted',
      resource: 'admin',
      severity: 'warn',
    })).toEqual({
      id: 14,
      timestamp: 1710000035,
      icon: 'mdi-account-remove-outline',
      tone: 'warning',
      text: 'Admin deleted',
      detail: 'actor: admin; resource: admin',
    })

    const unknown = mapAuditDisplayItem({
      id: -5,
      timestamp: 1710000040,
      event: '<svg onload=alert(1)>',
      severity: 'warn',
      resource: '<unknown>',
      privateField: 'ignored',
    })

    expect(unknown).toEqual({
      id: 0,
      timestamp: 1710000040,
      icon: 'mdi-shield-alert-outline',
      tone: 'warning',
      text: 'Audit event',
      detail: 'resource: unknown; event: svg onload=alert(1)',
    })
    expect(auditDisplayIcons).toContain(unknown.icon)
    expect(`${unknown.text} ${unknown.detail}`).not.toMatch(/[<>]/)

    expect(selectTrafficSeries({
      range: 'tomorrow',
      stats: [
        null,
        { dateTime: 1710000000, direction: 'down', traffic: 5 },
        { dateTime: 1710000000, direction: false, traffic: -1 },
      ],
      timeZone: 'UTC',
    })).toEqual({
      labels: ['2024-03-09 16:00'],
      download: [0],
      upload: [0],
      range: '24h',
    })
    expect(selectSystemStatus({
      sys: {
        ipv4: ['<192.0.2.4>', 12],
        appVersion: '<next>',
      },
      sbd: {
        running: 'yes',
        stats: {
          Alloc: -1,
        },
      },
    }, 100)).toEqual({
      ipv4: ['192.0.2.4'],
      ipv6: [],
      appVersion: 'next',
      bootTime: 0,
      uptimeSec: 0,
      singboxRunning: false,
      singboxAlloc: 0,
    })
  })
})
