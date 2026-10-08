// One record of the protocol: a JSON object carrying at least a string `type`
// and a string `message`. Every other field rides along untouched; ack reads
// them by name.
export type EventRecord = {
  readonly type: string
  readonly message: string
  readonly [field: string]: unknown
}

// What the mod does with a record.
export type Action = 'status' | 'inject'

// parseRecord reads one line of a source or a mapper as a record. It answers
// undefined for anything that is not a JSON object with a string type and a
// string message.
export function parseRecord(line: string): EventRecord | undefined {
  let v: unknown
  try {
    v = JSON.parse(line)
  } catch {
    return undefined
  }
  if (typeof v !== 'object' || v === null || Array.isArray(v)) return undefined
  const r = v as Record<string, unknown>
  if (typeof r.type !== 'string' || typeof r.message !== 'string') return undefined
  return r as EventRecord
}

// actionOf picks what a record asks for. A type the mod does not know, and a
// message that does not set inject to true, ask for nothing: the protocol lets
// a source carry more than this mod handles.
export function actionOf(record: EventRecord): Action | undefined {
  switch (record.type) {
    case 'status':
      return 'status'
    case 'message':
      return record.inject === true ? 'inject' : undefined
    default:
      return undefined
  }
}
