import type { EngineInterface, PluginOptions, Register } from 'claude-code'
import { type Run, agentStates, anyEnvSet, reportStates, reportWithCommand } from './watch'

// strings reads a `multiple` string option. The engine validates options
// against the manifest before the module loads, so anything else is only an
// unset option.
function strings(v: PluginOptions[string] | undefined): readonly string[] {
  return Array.isArray(v) ? v.filter(s => s !== '') : []
}

// Each read of the listing spawns a Claude Code process, which costs a good
// part of a second, so the interval stays well above that.
const POLL_MS = 2000

// Whatever records the state may have restarted, or refused a report, and has
// to hear the state again even when it did not change.
const RESEND_MS = 10_000

// A reporter that stays down would otherwise put a line in the debug log every
// tick, so the first failure of a run is logged, then one in every WARN_EVERY:
// at the 2s interval, one line every five minutes.
const WARN_EVERY = 150

function throttledLog($: EngineInterface) {
  let failures = 0
  return {
    fail(message: string) {
      failures++
      if (failures === 1 || failures % WARN_EVERY === 0) {
        $.ui.log(`report-agent-state: ${message} (failures: ${failures})`, { to: 'debug' })
      }
    },
    ok() {
      failures = 0
    },
  }
}

export const register: Register = (on, options) => {
  const reportArgv = strings(options.reportCommand)
  const requireEnv = strings(options.requireEnv)

  on('session.start', async ($, e, next) => {
    const result = await next(e)
    if (reportArgv.length === 0) return result
    const run: Run = (argv, init) => $.process.run(argv, init)
    // A session the reporter has no identity for stays quiet instead of
    // failing on every tick.
    if (!(await anyEnvSet(run, requireEnv))) return result
    const log = throttledLog($)
    const states = agentStates({
      every: (ms, fn) => $.clock.every(ms, fn),
      sessionId: () => $.session.id(),
      run,
      intervalMs: POLL_MS,
      onError: log.fail,
    })
    void reportStates(states, {
      report: reportWithCommand(run, reportArgv),
      now: () => $.clock.now(),
      resendMs: RESEND_MS,
      onError: log.fail,
      onReported: log.ok,
    })
    return result
  })
}
