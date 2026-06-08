import { describe, expect, it } from 'vitest'

import {
  auditDisplayIcons,
  mapAuditDisplayItem,
  mapAuditDisplayItems,
} from './auditMapper'
import { selectKpiSummary } from './kpiSelectors'
import { selectProtocolSummaries } from './protocolSummarySelectors'
import { selectSystemStatus } from './systemStatusSelectors'
import { selectTopClients } from './topClientsSelectors'
import { selectTrafficSeries } from './trafficSelectors'

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

    expect(selectTrafficSeries({ stats: trafficStats, range: '7d' })).toEqual({
      labels: [
        '2024-03-09T16:00:00.000Z',
        '2024-03-09T16:01:00.000Z',
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
    })).toEqual({
      labels: ['2024-03-09T16:00:00.000Z'],
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
