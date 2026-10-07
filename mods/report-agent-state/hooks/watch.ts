import { type HarnessState, stateOf } from './state'

// The engine calls the helpers make. The engine refuses a $ passed across an
// import, so register.tsx hands in closures over its own $ instead.
export type Every = (ms: number, fn: () => void) => { cancel: () => void }
export type Now = () => Promise<number>
export type Run = (
  argv: readonly string[],
  init?: { timeoutMs?: number },
) => Promise<{ exitCode: number; stdout: string; stderr: string }>

// ticks yields once at once, then once per period. A period that ends while
// the consumer is still busy with the last tick is folded into the next one,
// so a slow listing never stacks reads up.
//
// The loop rides on $.clock.every rather than $.clock.sleep: a sleep is charged
// to the budget of the hook that started it, while a timer runs until it is
// cancelled or the module reloads.
export async function* ticks(every: Every, ms: number): AsyncGenerator<void> {
  let wake: (() => void) | undefined
  let pending = false
  const timer = every(ms, () => {
    pending = true
    wake?.()
  })
  try {
    yield
    for (;;) {
      if (!pending) await new Promise<void>(resolve => (wake = resolve))
      wake = undefined
      pending = false
      yield
    }
  } finally {
    timer.cancel()
  }
}

export type WatchOptions = {
  every: Every
  sessionId: () => Promise<string>
  run: Run
  intervalMs: number
  onError?: (message: string) => void
}

// agentStates yields the session's state as `claude agents --json` shows it,
// once per tick. The state is never inferred from this session's own events:
// an interrupted turn fires nothing, and a background subagent, shell or
// monitor keeps the session working after its main turn ended. The listing is
// Claude Code's own account of both.
//
// The session id is read on every tick, since /clear, /resume and their kin
// switch it inside the same process without loading the mod again. A tick
// whose listing does not carry the session yields nothing.
export async function* agentStates(opts: WatchOptions): AsyncGenerator<HarnessState> {
  for await (const _ of ticks(opts.every, opts.intervalMs)) {
    let state: HarnessState | undefined
    try {
      const sessionId = await opts.sessionId()
      // Each read spawns a Claude Code process; one that hangs past the
      // interval would otherwise hold every later tick.
      const listed = await opts.run(['claude', 'agents', '--json'], { timeoutMs: opts.intervalMs })
      if (listed.exitCode !== 0) {
        opts.onError?.(`claude agents --json exited ${listed.exitCode}: ${listed.stderr.trim()}`)
        continue
      }
      state = stateOf(listed.stdout, sessionId)
    } catch (err) {
      opts.onError?.(String(err))
      continue
    }
    if (state !== undefined) yield state
  }
}

// Report hands one state to whatever records it for the room, and rejects when
// it did not get there.
export type Report = (state: HarnessState) => Promise<void>

const REPORT_TIMEOUT_MS = 5000

// reportWithCommand runs argv with the state word appended, in the session's
// environment, and counts a non-zero exit as a failed report.
export function reportWithCommand(run: Run, argv: readonly string[]): Report {
  return async (state) => {
    const r = await run([...argv, state], { timeoutMs: REPORT_TIMEOUT_MS })
    if (r.exitCode !== 0) {
      throw new Error(`${[...argv, state].join(' ')} exited ${r.exitCode}: ${r.stderr.trim()}`)
    }
  }
}

// anyEnvSet reports whether the session's environment sets at least one of
// names to a non-empty value; no names at all is a yes.
//
// It asks printenv rather than $.env.get: the engine takes only a string
// literal as a variable name there, and these names come from the options.
export async function anyEnvSet(run: Run, names: readonly string[]): Promise<boolean> {
  if (names.length === 0) return true
  for (const name of names) {
    const r = await run(['printenv', name], { timeoutMs: REPORT_TIMEOUT_MS })
    if (r.exitCode === 0 && r.stdout.trim() !== '') return true
  }
  return false
}

export type ReportOptions = {
  report: Report
  now: Now
  resendMs: number
  onError?: (message: string) => void
  onReported?: () => void
}

// reportStates reports each state that differs from the last one reported, and
// an unchanged one again once resendMs passed, so a daemon that restarted or
// refused a report hears it again. A failed report is retried on the next
// state, whatever it is.
export async function reportStates(states: AsyncIterable<HarnessState>, opts: ReportOptions): Promise<void> {
  let last: { state: HarnessState; at: number } | undefined
  for await (const state of states) {
    const now = await opts.now()
    if (last?.state === state && now - last.at < opts.resendMs) continue
    try {
      await opts.report(state)
      last = { state, at: now }
      opts.onReported?.()
    } catch (err) {
      last = undefined
      opts.onError?.(err instanceof Error ? err.message : String(err))
    }
  }
}
