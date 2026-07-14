import { describe, expect, it } from 'vitest'

import { createEndpoint } from './endpoints'

describe('managed AWG endpoint marker', () => {
  it('survives endpoint model creation for the editor lock', () => {
    const endpoint = createEndpoint('wireguard', {
      id: 7,
      tag: 'managed-awg',
      awgManaged: true,
      peers: [],
    })

    expect(endpoint.awgManaged).toBe(true)
    expect(endpoint.peers).toEqual([])
  })
})
