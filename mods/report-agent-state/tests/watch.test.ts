import { describe, expect, test } from 'claude-code/testing'
import type { HarnessState } from '../hooks/state'
import { type Run, agentStates, reportStates } from '../hooks/watch'

type Result = Awaited<ReturnType<Run>>

// fakeWatch answers each listing from results in order, and fires its one
// timer only when the test calls tick.
function fakeWatch(results: Result[]) {
  let fire: (() => void) | undefined
  let cancelled = false
  return {
    opts: {
      every: (_ms: number, fn: () => void) => {
        fire = fn
        return { cancel: () => void (cancelled = true) }
      },
      sessionId: async () => 's1',
      run: async () => results.shift() ?? { exitCode: 0, stdout: '[]', stderr: '' },
      intervalMs: 2000,
    },
    tick: () => fire?.(),
    cancelled: () => cancelled,
  }
}

const listed = (status: string): Result => ({
  exitCode: 0,
  stdout: JSON.stringify([{ pid: 2, sessionId: 's1', status }]),
  stderr: '',
})

describe('agentStates', () => {
  test('yields the listed state at once, then once per tick', async () => {
    const f = fakeWatch([listed('busy'), listed('waiting'), listed('idle')])
    const it = agentStates(f.opts)
    expect((await it.next()).value).toBe('working')
    const second = it.next()
    f.tick()
    expect((await second).value).toBe('waiting')
    const third = it.next()
    f.tick()
    expect((await third).value).toBe('done')
    await it.return(undefined)
    expect(f.cancelled()).toBe(true)
  })

  test('folds ticks that arrive while the consumer is busy into one', async () => {
    const f = fakeWatch([listed('busy'), listed('idle'), listed('busy')])
    const it = agentStates(f.opts)
    await it.next()
    f.tick()
    f.tick()
    expect((await it.next()).value).toBe('done')
    let settled = false
    const pending = it.next()
    void pending.then(() => (settled = true))
    for (let i = 0; i < 10; i++) await Promise.resolve()
    expect(settled).toBe(false)
    f.tick()
    expect((await pending).value).toBe('working')
  })

  test('reports a failed listing and yields the next good one', async () => {
    const f = fakeWatch([{ exitCode: 1, stdout: '', stderr: 'boom' }, listed('busy')])
    const errors: string[] = []
    const it = agentStates({ ...f.opts, onError: m => errors.push(m) })
    const first = it.next()
    f.tick()
    expect((await first).value).toBe('working')
    expect(errors).toEqual(['claude agents --json exited 1: boom'])
  })
})

describe('reportStates', () => {
  test('hands states to the report, skipping repeats inside the resend window', async () => {
    let now = 0
    async function* states(): AsyncGenerator<HarnessState> {
      yield 'working'
      yield 'working'
      now += 10_000
      yield 'working'
      yield 'done'
    }
    const reported: HarnessState[] = []
    await reportStates(states(), {
      report: async s => void reported.push(s),
      now: async () => now,
      resendMs: 10_000,
    })
    expect(reported).toEqual(['working', 'working', 'done'])
  })

  test('retries after a failed report', async () => {
    async function* states(): AsyncGenerator<HarnessState> {
      yield 'working'
      yield 'working'
    }
    let calls = 0
    const errors: string[] = []
    await reportStates(states(), {
      report: async () => {
        if (calls++ === 0) throw new Error('daemon down')
      },
      now: async () => 0,
      resendMs: 10_000,
      onError: m => errors.push(m),
    })
    expect(calls).toBe(2)
    expect(errors).toEqual(['daemon down'])
  })
})
