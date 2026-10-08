import { type Now, type Run, type Spawn, type SpawnEnd, type Wait, withStderr } from './host'
import { lineSplitter, splitLines } from './lines'
import { type EventRecord, parseRecord } from './protocol'

// The pause before restarting a source starts here and doubles after each
// restart up to BACKOFF_MAX_MS.
export const BACKOFF_MIN_MS = 1000
export const BACKOFF_MAX_MS = 30_000

// A source that ran this long before it exited counts as healthy, so its next
// restart waits the shortest pause again. A valid record counts the same.
export const HEALTHY_RUN_MS = 30_000

const MAPPER_TIMEOUT_MS = 10_000

// How much of a dropped line a log message quotes.
const QUOTE_MAX = 200

export type RecordOptions = {
  mapper: readonly string[]
  run: Run
  onDrop?: (message: string) => void
}

// recordsOf turns one line of a source into the records it carries. Without a
// mapper the line is the record. With one, the mapper runs with the line on
// its standard input, and each non-empty line it prints is a record; a mapper
// that exits non-zero drops the line. A line that does not parse as a record
// is dropped. Each drop is reported to onDrop.
export async function recordsOf(line: string, opts: RecordOptions): Promise<EventRecord[]> {
  if (line.trim() === '') return []
  let lines = [line]
  if (opts.mapper.length > 0) {
    let r: Awaited<ReturnType<Run>>
    try {
      r = await opts.run(opts.mapper, { stdin: line + '\n', timeoutMs: MAPPER_TIMEOUT_MS })
    } catch (err) {
      opts.onDrop?.(`mapper failed on ${quote(line)}: ${errorText(err)}`)
      return []
    }
    if (r.exitCode !== 0) {
      opts.onDrop?.(withStderr(`mapper exited ${r.exitCode} on ${quote(line)}`, r.stderr))
      return []
    }
    lines = splitLines(r.stdout)
  }
  const records: EventRecord[] = []
  for (const l of lines) {
    if (l.trim() === '') continue
    const record = parseRecord(l)
    if (record === undefined) {
      opts.onDrop?.(`not a record: ${quote(l)}`)
      continue
    }
    records.push(record)
  }
  return records
}

export type SourceOptions = RecordOptions & {
  argv: readonly string[]
  spawn: Spawn
  now: Now
  wait: Wait
  onRecord: (record: EventRecord) => void
  // onRestart is told why the source ended and how long the restart waits.
  onRestart?: (reason: string, delayMs: number) => void
  // onRecovered is called on the first valid record after a restart.
  onRecovered?: () => void
  signal?: AbortSignal
}

// runSource runs the source for as long as signal allows, restarting it each
// time it ends. Records reach onRecord in the order the source printed them:
// a line waits for the mapper run of the line before it.
export async function runSource(opts: SourceOptions): Promise<void> {
  let delay = BACKOFF_MIN_MS
  let restarting = false
  while (!opts.signal?.aborted) {
    const startedAt = await opts.now()
    let healthy = false
    const reason = await runOnce(opts, record => {
      healthy = true
      if (restarting) {
        restarting = false
        opts.onRecovered?.()
      }
      opts.onRecord(record)
    })
    if (opts.signal?.aborted) return
    if (healthy || (await opts.now()) - startedAt >= HEALTHY_RUN_MS) delay = BACKOFF_MIN_MS
    restarting = true
    opts.onRestart?.(reason, delay)
    await opts.wait(delay)
    delay = Math.min(delay * 2, BACKOFF_MAX_MS)
  }
}

// runOnce runs the source until it ends and answers why it ended.
async function runOnce(opts: SourceOptions, onRecord: (record: EventRecord) => void): Promise<string> {
  const lines = lineSplitter()
  let lastStderr = ''
  const handle = async (line: string) => {
    for (const record of await recordsOf(line, opts)) onRecord(record)
  }
  let end: SpawnEnd | void
  try {
    const child = opts.spawn(opts.argv)
    for (;;) {
      const step = await child.next()
      if (step.done) {
        end = step.value
        break
      }
      if (step.value.stream === 'stderr') {
        lastStderr = lastLine(step.value.text) ?? lastStderr
        continue
      }
      for (const line of lines.push(step.value.text)) await handle(line)
      if (opts.signal?.aborted) {
        await child.return?.()
        return 'stopped'
      }
    }
  } catch (err) {
    return `source failed: ${errorText(err)}`
  }
  const rest = lines.end()
  if (rest !== undefined) await handle(rest)
  return endReason(end, lastStderr)
}

function endReason(end: SpawnEnd | void, lastStderr: string): string {
  let reason = 'source ended'
  if (end?.code !== null && end?.code !== undefined) reason = `source exited with code ${end.code}`
  else if (end?.signal) reason = `source ended by ${end.signal}`
  return withStderr(reason, lastStderr)
}

function lastLine(text: string): string | undefined {
  return splitLines(text)
    .map(l => l.trim())
    .filter(l => l !== '')
    .at(-1)
}

function quote(line: string): string {
  return JSON.stringify(line.length > QUOTE_MAX ? line.slice(0, QUOTE_MAX) + '...' : line)
}

function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}
