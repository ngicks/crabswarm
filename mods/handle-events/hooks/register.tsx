import type { EngineInterface, PluginOptions, Register, Timer } from 'claude-code'
import { ackRecords } from './ack'
import { type Run, type Spawn, anyEnvSet } from './host'
import { actionOf } from './protocol'
import { type InjectQueue, injectQueue } from './queue'
import { runSource } from './source'

// strings reads a `multiple` string option. The engine validates options
// against the manifest before the module loads, so anything else is only an
// unset option.
function strings(v: PluginOptions[string] | undefined): readonly string[] {
  return Array.isArray(v) ? v.filter(s => s !== '') : []
}

// A failure that repeats (a source that keeps dying, a source printing lines
// the mod cannot read, an ack command that keeps failing) would otherwise put
// a line in the debug log each time, so the first failure of a run is logged,
// then one in every WARN_EVERY.
const WARN_EVERY = 100

// A status line entry is cleared this long after it was shown. A source
// reports a moment rather than a lasting state, and an entry left in place
// reads as current long after it stopped being true.
const STATUS_HOLD_MS = 5000

function throttledLog($: EngineInterface, counted: string) {
  let failures = 0
  return {
    fail(message: string) {
      failures++
      if (failures === 1 || failures % WARN_EVERY === 0) {
        $.ui.log(`handle-events: ${message} (${counted}: ${failures})`, { to: 'debug' })
      }
    },
    ok() {
      failures = 0
    },
  }
}

export const register: Register = (on, options) => {
  const sourceArgv = strings(options.source)
  const mapperArgv = strings(options.mapper)
  const prefix = typeof options.prefix === 'string' ? options.prefix : ''
  const ackArgv = strings(options.ack)
  const requireEnv = strings(options.requireEnv)

  let queue: InjectQueue | undefined
  // session.start may fire again in the same load; one source is enough.
  let started = false

  on('session.start', async ($, e, next) => {
    const result = await next(e)
    if (started || sourceArgv.length === 0) return result
    started = true
    const run: Run = (argv, init) => $.process.run(argv, init)
    // A session the source has no identity for stays quiet instead of
    // restarting a failing source forever.
    if (!(await anyEnvSet(run, requireEnv))) return result

    const drops = throttledLog($, 'drops')
    const ackFailures = throttledLog($, 'ack failures')
    const restarts = throttledLog($, 'restarts')
    let clearing: Timer | undefined
    const status = (text: string | undefined) => {
      clearing?.cancel()
      clearing = text === undefined ? undefined : $.clock.after(STATUS_HOLD_MS, () => void $.ui.status(undefined))
      void $.ui.status(text)
    }

    const q = injectQueue({
      prefix,
      submit: async text => {
        const r = await $.prompt.submit({ text })
        if (r.drop !== undefined) throw new Error(`the prompt was dropped: ${r.drop}`)
      },
      onSubmitted: records =>
        ackRecords(records, { run, argv: ackArgv, onError: ackFailures.fail, onAcked: ackFailures.ok }),
      onError: drops.fail,
    })
    queue = q

    const spawn: Spawn = argv => $.process.spawn({ argv })
    void runSource({
      argv: sourceArgv,
      mapper: mapperArgv,
      run,
      spawn,
      now: () => $.clock.now(),
      wait: ms => new Promise<void>(resolve => void $.clock.after(ms, () => resolve())),
      onRecord: record => {
        drops.ok()
        switch (actionOf(record)) {
          case 'status':
            status(record.message === '' ? undefined : record.message)
            break
          case 'inject':
            q.push(record)
            break
        }
      },
      onDrop: drops.fail,
      onRestart: (reason, delayMs) => {
        restarts.fail(reason)
        status(`handle-events: ${reason}; restarting in ${Math.ceil(delayMs / 1000)}s`)
      },
      onRecovered: () => {
        restarts.ok()
        status(undefined)
      },
    })
    return result
  })

  // A subagent's run raises no turn.start, so every one is the main loop's.
  on('turn.start', async (_$, e, next) => {
    queue?.turnStarted()
    return next(e)
  })

  on('turn.complete', async (_$, e, next) => {
    const result = await next(e)
    // The flush is not awaited here: the prompt it submits starts its turn
    // only once the session is idle, which is after this hook returns.
    if (e.agentId === undefined) queue?.turnCompleted()
    return result
  })
}
