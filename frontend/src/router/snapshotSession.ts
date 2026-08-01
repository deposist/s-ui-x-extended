export interface SnapshotSessionDependencies {
  loadSnapshot: () => void | Promise<void>
  connectRealtime: () => void | Promise<void>
  disconnectRealtime: () => void
}

/**
 * Owns the authenticated page lifecycle, not a second polling loop.
 * WsRuntime is the sole owner of degraded polling and bounded reconnects, so a
 * healthy socket never competes with an unconditional router interval.
 */
export class SnapshotSession {
  private active = false

  constructor(private readonly deps: SnapshotSessionDependencies) {}

  enter() {
    if (this.active) return
    this.active = true
    void this.deps.loadSnapshot()
    void this.deps.connectRealtime()
  }

  leave() {
    if (!this.active) return
    this.active = false
    this.deps.disconnectRealtime()
  }
}
