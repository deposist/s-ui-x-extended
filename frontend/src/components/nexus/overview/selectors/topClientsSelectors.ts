import { isSelectorRecord, nonNegativeNumber, plainText, plainTextList } from './selectorUtils'

export interface TopClientRow {
  id?: number
  name: string
  upload: number
  download: number
  total: number
  online: boolean
}

export interface TopClientsSelectorInput {
  clients?: readonly unknown[] | null
  onlines?: {
    user?: readonly unknown[] | null
  } | null
}

const defaultTopClientLimit = 10

const sumTraffic = (...values: unknown[]): number => {
  return values.reduce<number>((sum, value) => sum + (nonNegativeNumber(value) ?? 0), 0)
}

const compareTopClients = (left: TopClientRow, right: TopClientRow): number => {
  if (left.total !== right.total) return right.total - left.total
  if (left.name === right.name) return (left.id ?? 0) - (right.id ?? 0)
  return left.name < right.name ? -1 : 1
}

const topClientLimit = (limit: number): number => {
  return Number.isFinite(limit) ? Math.max(0, Math.floor(limit)) : defaultTopClientLimit
}

export const selectTopClients = (
  input?: TopClientsSelectorInput | null,
  limit = defaultTopClientLimit,
): TopClientRow[] => {
  const onlineNames = new Set(plainTextList(input?.onlines?.user))

  const rows = (input?.clients ?? []).reduce<TopClientRow[]>((clients, client) => {
    if (!isSelectorRecord(client)) return clients

    const name = plainText(client.name)
    if (!name) return clients

    const upload = sumTraffic(client.totalUp, client.up)
    const download = sumTraffic(client.totalDown, client.down)
    const id = nonNegativeNumber(client.id)
    const row: TopClientRow = {
      name,
      upload,
      download,
      total: upload + download,
      online: onlineNames.has(name),
    }

    if (id !== undefined) row.id = id
    clients.push(row)
    return clients
  }, [])

  return rows.sort(compareTopClients).slice(0, topClientLimit(limit))
}
