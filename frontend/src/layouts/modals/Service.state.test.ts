import { webcrypto } from 'node:crypto'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Service from './Service.vue'
import ServiceDrawer from '@/components/nexus/drawers/ServiceDrawer.vue'

vi.mock('@/plugins/httputil', () => ({ default: {} }))
beforeEach(() => vi.stubGlobal('window', { crypto: webcrypto }))
afterEach(() => vi.unstubAllGlobals())

describe.each([['classic', Service], ['nexus', ServiceDrawer]])('%s service forms', (_mode, component) => {
  it.each(['oom-killer', 'profiler', 'usbip-client'])('uses its own listener defaults when changing to %s', type => {
    const makeData = component.data! as (this: any, vm: any) => Record<string, any>
    const vm: any = {
      ...makeData.call({}, {}),
      $props: { id: 0 },
      srv: { type, tag: 'derp', listen: '::', listen_port: 443 },
    }
    component.methods!.changeType.call(vm)
    if (type === 'profiler') expect(vm.srv.listen).toBe('127.0.0.1:8964')
    else expect(vm.srv).not.toHaveProperty('listen')
    expect(vm.srv).not.toHaveProperty('listen_port')
  })
})
