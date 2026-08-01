import { describe, expect, it } from 'vitest'
import modalSource from '@/components/IpHistoryModal.vue?raw'
import { createLatestRequestRunner } from '@/components/asyncRequestFence'

const deferred = <T>() => {
  let resolve!: (value: T | PromiseLike<T>) => void
  const promise = new Promise<T>(done => {
    resolve = done
  })
  return { promise, resolve }
}

describe('IP history request lifecycle', () => {
  it('does not let an older client promise overwrite newer rows', async () => {
    let loading = false
    let rows: string[] = []
    let olderSignal: AbortSignal | undefined
    const older = deferred<string[]>()
    const newer = deferred<string[]>()
    const runner = createLatestRequestRunner(value => {
      loading = value
    })

    const olderRequest = runner.start(signal => {
      olderSignal = signal
      return older.promise
    }, value => {
      rows = value
    })
    const newerRequest = runner.start(() => newer.promise, value => {
      rows = value
    })

    expect(olderSignal?.aborted).toBe(true)
    expect(loading).toBe(true)

    newer.resolve(['newer-client'])
    await newerRequest.done
    expect(rows).toEqual(['newer-client'])

    older.resolve(['older-client'])
    await olderRequest.done
    expect(rows).toEqual(['newer-client'])
  })

  it('aborts the active request for both close and unmount cleanup', async () => {
    for (const cleanup of ['close', 'unmount']) {
      let loading = false
      let signal: AbortSignal | undefined
      const response = deferred<string[]>()
      const runner = createLatestRequestRunner(value => {
        loading = value
      })
      const request = runner.start(currentSignal => {
        signal = currentSignal
        return response.promise
      }, () => {})

      runner.abort()

      expect(signal?.aborted, cleanup).toBe(true)
      expect(loading, cleanup).toBe(false)
      response.resolve([])
      await request.done
    }
  })

  it('does not let a stale completion clear a newer loading state', async () => {
    let loading = false
    const older = deferred<void>()
    const newer = deferred<void>()
    const runner = createLatestRequestRunner(value => {
      loading = value
    })

    const olderRequest = runner.start(() => older.promise, () => {})
    const newerRequest = runner.start(() => newer.promise, () => {})

    older.resolve()
    await olderRequest.done
    expect(loading).toBe(true)

    newer.resolve()
    await newerRequest.done
    expect(loading).toBe(false)
  })

  it('fences a stale clear response from rows loaded for a newer client', async () => {
    let rows = ['current-client']
    let clearSignal: AbortSignal | undefined
    const clearResponse = deferred<void>()
    const newerResponse = deferred<string[]>()
    const runner = createLatestRequestRunner(() => {})

    const clearRequest = runner.start(signal => {
      clearSignal = signal
      return clearResponse.promise
    }, () => {
      rows = []
    })
    const newerRequest = runner.start(() => newerResponse.promise, value => {
      rows = value
    })

    expect(clearSignal?.aborted).toBe(true)
    newerResponse.resolve(['newer-client'])
    await newerRequest.done
    clearResponse.resolve()
    await clearRequest.done

    expect(rows).toEqual(['newer-client'])
  })

  it('wires watcher cleanup, close, and unmount to request cancellation', () => {
    expect(modalSource).toContain('onCleanup(request.abort)')
    expect(modalSource).toMatch(/if \(!value\) cancelRequests\(\)/)
    expect(modalSource).toContain('onBeforeUnmount(cancelRequests)')
  })
})
