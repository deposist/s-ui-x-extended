export interface LatestRequest {
  readonly signal: AbortSignal
  readonly done: Promise<void>
  abort: () => void
}

export interface LatestRequestRunner {
  start: <T>(operation: (signal: AbortSignal) => Promise<T>, apply: (result: T) => void) => LatestRequest
  abort: () => void
}

interface RequestToken {
  readonly generation: number
  readonly controller: AbortController
}

export const createLatestRequestRunner = (setLoading: (loading: boolean) => void): LatestRequestRunner => {
  let generation = 0
  let active: RequestToken | undefined

  const isCurrent = (request: RequestToken): boolean => {
    return active === request
      && request.generation === generation
      && !request.controller.signal.aborted
  }

  const abortRequest = (request: RequestToken): void => {
    request.controller.abort()
    if (active !== request) return

    active = undefined
    generation++
    setLoading(false)
  }

  const abort = (): void => {
    if (active) abortRequest(active)
  }

  const start = <T>(
    operation: (signal: AbortSignal) => Promise<T>,
    apply: (result: T) => void,
  ): LatestRequest => {
    abort()

    const request: RequestToken = {
      generation: ++generation,
      controller: new AbortController(),
    }
    active = request
    setLoading(true)

    const done = (async () => {
      try {
        const result = await operation(request.controller.signal)
        if (isCurrent(request)) apply(result)
      } finally {
        if (isCurrent(request)) {
          active = undefined
          setLoading(false)
        }
      }
    })()

    return {
      signal: request.controller.signal,
      done,
      abort: () => abortRequest(request),
    }
  }

  return { start, abort }
}
