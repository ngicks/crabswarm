import { type Run, withStderr } from './host'
import type { EventRecord } from './protocol'

const ACK_TIMEOUT_MS = 10_000

// An argv item that is exactly `{name}` stands for the record's field `name`.
const PLACEHOLDER = /^\{([^{}]+)\}$/

// fillArgv replaces each placeholder item of argv with the record's top-level
// field of that name, written as text. It answers the name of the first field
// that is missing or is not a string, a number or a boolean instead.
export function fillArgv(argv: readonly string[], record: EventRecord): { argv: string[] } | { missing: string } {
  const filled: string[] = []
  for (const item of argv) {
    const name = PLACEHOLDER.exec(item)?.[1]
    if (name === undefined) {
      filled.push(item)
      continue
    }
    const v = Object.hasOwn(record, name) ? record[name] : undefined
    if (typeof v !== 'string' && typeof v !== 'number' && typeof v !== 'boolean') return { missing: name }
    filled.push(String(v))
  }
  return { argv: filled }
}

export type AckOptions = {
  run: Run
  argv: readonly string[]
  onError?: (message: string) => void
  onAcked?: () => void
}

// ackRecords runs the ack command once per record, in order, each run after
// the one before it ended. A record that lacks a field the command names is
// skipped; a run that exits non-zero is reported and the next one still runs.
// An empty argv acks nothing.
export async function ackRecords(records: readonly EventRecord[], opts: AckOptions): Promise<void> {
  if (opts.argv.length === 0) return
  for (const record of records) {
    const filled = fillArgv(opts.argv, record)
    if ('missing' in filled) {
      opts.onError?.(`ack skipped: the record has no string, number or boolean "${filled.missing}": ${record.message}`)
      continue
    }
    const command = filled.argv.join(' ')
    try {
      const r = await opts.run(filled.argv, { timeoutMs: ACK_TIMEOUT_MS })
      if (r.exitCode !== 0) {
        opts.onError?.(withStderr(`${command} exited ${r.exitCode}`, r.stderr))
        continue
      }
    } catch (err) {
      opts.onError?.(`${command}: ${err instanceof Error ? err.message : String(err)}`)
      continue
    }
    opts.onAcked?.()
  }
}
