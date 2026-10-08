import type { EventRecord } from './protocol'

// Submit hands one prompt to the session and rejects when it did not enter.
export type Submit = (text: string) => Promise<void>

export type QueueOptions = {
  prefix: string
  submit: Submit
  // onSubmitted runs once the prompt holding the records entered.
  onSubmitted?: (records: readonly EventRecord[]) => Promise<void>
  onError?: (message: string) => void
}

// promptText writes records as one prompt: one line each, the prefix before
// the message, in the order given. A line equal to an earlier one is left out.
export function promptText(prefix: string, records: readonly EventRecord[]): string {
  const seen = new Set<string>()
  const lines: string[] = []
  for (const r of records) {
    const line = prefix === '' ? r.message : `${prefix} ${r.message}`
    if (seen.has(line)) continue
    seen.add(line)
    lines.push(line)
  }
  return lines.join('\n')
}

export type InjectQueue = {
  push: (record: EventRecord) => void
  turnStarted: () => void
  turnCompleted: () => void
}

// injectQueue holds records until the main loop is idle, then submits every
// record it holds as one prompt. A record that arrives while a turn runs, or
// while a prompt is on its way in, waits for the next turn to end.
export function injectQueue(opts: QueueOptions): InjectQueue {
  let pending: EventRecord[] = []
  let turnRunning = false
  // busy covers a submit and the onSubmitted after it, so two records that
  // arrive a moment apart while idle become one prompt.
  let busy = false
  // starts counts turn.start events, so a submit that resolves can tell
  // whether its turn already announced itself.
  let starts = 0

  const fail = (err: unknown) => opts.onError?.(errorText(err))

  async function deliver(batch: readonly EventRecord[]) {
    const startsBefore = starts
    try {
      await opts.submit(promptText(opts.prefix, batch))
    } catch (err) {
      opts.onError?.(`${batch.length} record(s) not injected: ${errorText(err)}`)
      return
    }
    // $.prompt.submit resolves as the prompt's own turn starts. Until that
    // turn's turn.start arrives, the queue counts the turn as running itself.
    if (starts === startsBefore) turnRunning = true
    try {
      await opts.onSubmitted?.(batch)
    } catch (err) {
      fail(err)
    }
  }

  async function flush(): Promise<void> {
    if (busy || turnRunning || pending.length === 0) return
    const batch = pending
    pending = []
    busy = true
    try {
      await deliver(batch)
    } finally {
      busy = false
    }
    await flush()
  }

  const kick = () => void flush().catch(fail)

  return {
    push(record) {
      pending.push(record)
      kick()
    },
    turnStarted() {
      starts++
      turnRunning = true
    },
    turnCompleted() {
      turnRunning = false
      kick()
    },
  }
}

function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}
