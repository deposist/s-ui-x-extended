import HttpUtils from '@/plugins/httputil'
import { defineStore } from 'pinia'
import { push } from 'notivue'
import { i18n } from '@/locales'
import { Inbound } from '@/types/inbounds'
import { Client } from '@/types/clients'
import { FailoverStatusEntry } from '@/types/outbounds'

type ActionableLogLevel = 'warning' | 'error'

const actionableLogLevel = (log: string): ActionableLogLevel | undefined => {
  if (/\b(?:ERROR|FATAL)\b/i.test(log)) return 'error'
  if (/\bWARN(?:ING)?\b/i.test(log)) return 'warning'

  return undefined
}

const serverRevision = (value: unknown): number | undefined => {
  const revision = typeof value === 'number' ? value : Number(value)
  return Number.isSafeInteger(revision) && revision >= 0 ? revision : undefined
}

const pendingLoads = new WeakMap<object, Promise<void>>()

const Data = defineStore('Data', {
  state: () => ({ 
    // Authoritative /api/load acknowledgement cursor returned by the server.
    lastLoad: 0,
    loadGeneration: 0,
    reloadItems: localStorage.getItem("reloadItems")?.split(',')?? <string[]>[],
    subURI: "",
    subJsonURI: "",
    subClashURI: "",
    enableTraffic: false,
    onlines: {inbound: <string[]>[], outbound: <string[]>[], user: <string[]>[], failover: <Record<string, FailoverStatusEntry>>{}, outboundHealth: <Record<string, any>>{}, providerHealth: <Record<string, any>>{}},
    config: <any>{},
    inbounds: <any[]>[],
    outbounds: <any[]>[],
    services: <any[]>[],
    endpoints: <any[]>[],
    clients: <any>[],
    providers: <any[]>[],
    tlsConfigs: <any[]>[],
  }),
  actions: {
    loadData() {
      const pending = pendingLoads.get(this)
      if (pending) return pending

      const generation = this.loadGeneration
      const request: Promise<void> = (async () => {
        const msg = await HttpUtils.get('api/load', this.lastLoad > 0 ? { lu: this.lastLoad } : {})
        if (generation !== this.loadGeneration || !msg.success || !msg.obj) return

      const data = msg.obj as Record<string, any>
      const revision = serverRevision(data.revision)
      // A response older than the snapshot already published by a save or
      // another load must never roll the store back.
      if (revision !== undefined && revision < this.lastLoad) return

      this.setNewData(data)
      if (data.lastLog) {
        const logLevel = actionableLogLevel(String(data.lastLog))

        if (logLevel === 'error') {
          push.error({
            title: i18n.global.t('error.core'),
            duration: 8000,
            message: data.lastLog
          })
        } else if (logLevel === 'warning') {
          push.warning({
            title: i18n.global.t('warning'),
            duration: 6000,
            message: data.lastLog
          })
        }
      }
      })().finally(() => {
        if (pendingLoads.get(this) === request) pendingLoads.delete(this)
      })
      pendingLoads.set(this, request)
      return request
    },
    setNewData(data: Record<string, any>) {
      const revision = serverRevision(data.revision)
      if (revision !== undefined) {
        if (revision < this.lastLoad) return false
        this.lastLoad = revision
      }
      if (Object.hasOwn(data, 'onlines')) this.onlines = data.onlines ?? { inbound: [], outbound: [], user: [], failover: {}, outboundHealth: {}, providerHealth: {} }
      if (Object.hasOwn(data, 'subURI')) this.subURI = data.subURI ?? ""
      else if (Object.hasOwn(data, 'config')) this.subURI = ""
      if (Object.hasOwn(data, 'subJsonURI')) this.subJsonURI = data.subJsonURI ?? ""
      else if (Object.hasOwn(data, 'config')) this.subJsonURI = ""
      if (Object.hasOwn(data, 'subClashURI')) this.subClashURI = data.subClashURI ?? ""
      else if (Object.hasOwn(data, 'config')) this.subClashURI = ""
      if (Object.hasOwn(data, 'enableTraffic')) this.enableTraffic = data.enableTraffic
      if (data.config) this.config = data.config
      if (Object.hasOwn(data, 'clients')) this.clients = data.clients ?? []
      if (Object.hasOwn(data, 'inbounds')) this.inbounds = data.inbounds ?? []
      if (Object.hasOwn(data, 'outbounds')) this.outbounds = data.outbounds ?? []
      if (Object.hasOwn(data, 'services')) this.services = data.services ?? []
      if (Object.hasOwn(data, 'endpoints')) this.endpoints = data.endpoints ?? []
      if (Object.hasOwn(data, 'providers')) this.providers = data.providers ?? []
      if (Object.hasOwn(data, 'tls')) this.tlsConfigs = data.tls ?? []
      return true
    },
    async loadInbounds(ids: number[]): Promise<Inbound[]> {
      const options = ids.length > 0 ? {id: ids.join(",")} : {}
      const msg = await HttpUtils.get('api/inbounds', options)
      if(msg.success) {
        return msg.obj.inbounds
      }
      return <Inbound[]>[]
    },
    async loadClients(id: number): Promise<Client> {
      const options = id > 0 ? {id: id} : {}
      const msg = await HttpUtils.get('api/clients', options)
      if(msg.success) {
        return (msg.obj.clients?.[0] ?? {}) as Client
      }
      return <Client>{}
    },
    async save (object: string, action: string, data: any, initUsers?: number[]): Promise<boolean> {
      let postData = {
        object: object,
        action: action,
        data: JSON.stringify(data, null, 2),
        initUsers: initUsers?.join(',') ?? undefined
      }
      const msg = await HttpUtils.post('api/save', postData)
      // A pre-save load response describes the old configuration. Invalidate it
      // before publishing the authoritative save response so it cannot finish
      // later and overwrite the just-saved state.
      if (msg.success) this.loadGeneration++
      if (msg.success) {
        const objectName = ['tls', 'config'].includes(object) ? object : object.substring(0, object.length - 1)
        push.success({
          title: i18n.global.t('success'),
          duration: 5000,
          message: i18n.global.t('actions.' + action) + " " + i18n.global.t('objects.' + objectName)
        })
        this.setNewData(msg.obj)
      }
      return msg.success
    },
    // Check duplicate client name
    checkClientName (id: number, newName: string): boolean {
      const oldName = id > 0 ? this.clients.findLast((i: any) => i.id == id)?.name : null
      if (newName != oldName && this.clients.findIndex((c: any) => c.name == newName) != -1) {
        push.error({
          message: i18n.global.t('error.dplData') + ": " + i18n.global.t('client.name')
        })
        return true
      }
      return false
    },
    // Check bulk client names
    checkBulkClientNames (names: string[]): boolean {
      const newNames = new Set(names)
      const oldNames = new Set(this.clients.map((c: any) => c.name))
      const allNames = new Set([...oldNames, ...newNames])
      if (newNames.size != names.length || oldNames.size + newNames.size != allNames.size) {
        push.error({
          message: i18n.global.t('error.dplData') + ": " + i18n.global.t('client.name')
        })
        return true
      }
      return false
    },
    // check duplicate tag
    checkTag (object: string, id: number, tag: string): boolean {
      let objects = <any[]>[]
      switch (object) {
        case 'inbound':
          objects = this.inbounds
          break
        case 'outbound':
          objects = this.outbounds
          break
        case 'service':
          objects = this.services
          break
        case 'endpoint':
          objects = this.endpoints
          break
        case 'provider':
          objects = this.providers
          break
        case 'tls':
          objects = this.tlsConfigs
          break
        default:
          return false
      }
      const identity = object === 'tls' ? 'name' : 'tag'
      const oldObject = id > 0 ? objects.findLast((i: any) => i.id == id) : null
      if (tag != oldObject?.[identity] && objects.findIndex((i: any) => i[identity] == tag) != -1) {
        push.error({
          message: i18n.global.t('error.dplData') + ": " + i18n.global.t('objects.tag')
        })
        return true
      }
      return false
    },
  }
})

export default Data
