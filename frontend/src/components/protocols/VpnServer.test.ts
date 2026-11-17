import { describe, expect, it } from 'vitest'
import VpnServer from './VpnServer.vue'

describe('VPN server user addresses', () => {
  it('adds a unique address after a middle user is removed', () => {
    const data = { users: [
      { address: '10.0.0.2', key: 'first' },
      { address: '10.0.0.3', key: 'removed' },
      { address: '10.0.0.4', key: 'last' },
    ] }
    const vm = { $props: { data }, genKey: () => 'new' } as any
    VpnServer.methods!.delUser.call(vm, 1)
    VpnServer.methods!.addUser.call(vm)
    expect(data.users.map(user => user.address)).toEqual(['10.0.0.2', '10.0.0.4', '10.0.0.3'])
  })

  it('does not add an address when the default address range is full', () => {
    const data = { users: Array.from({ length: 253 }, (_, index) => ({ address: `10.0.0.${index + 2}`, key: 'existing' })) }
    const vm = { $props: { data }, genKey: () => 'new' } as any
    VpnServer.methods!.addUser.call(vm)
    expect(data.users).toHaveLength(253)
  })
})
