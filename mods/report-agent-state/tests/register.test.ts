import { describe, expect, mock, test } from 'claude-code/testing'
import type { On } from 'claude-code'

type World = {
  env: Record<string, string>
  sessionId: string
  status: string
  reports: string[]
  listings: number
}

const world = (env: Record<string, string>): World => ({ env, sessionId: 's1', status: 'busy', reports: [], listings: 0 })

function engine(on: On, w: World) {
  const clock = mock.clock(on)
  on('session.start', (_$, e) => ({ cwd: e.cwd }))
  on('session.id', () => ({ value: w.sessionId }))
  on('process.run', (_$, e) => {
    const [cmd, ...args] = e.argv
    if (cmd === 'printenv') {
      const value = w.env[args[0] ?? '']
      return { value: { exitCode: value === undefined ? 1 : 0, stdout: value ?? '', stderr: '' } } as never
    }
    if (cmd === 'claude') {
      w.listings++
      const stdout = JSON.stringify([{ pid: 2, sessionId: w.sessionId, kind: 'interactive', status: w.status }])
      return { value: { exitCode: 0, stdout, stderr: '' } } as never
    }
    w.reports.push(e.argv.join(' '))
    return { value: { exitCode: 0, stdout: '', stderr: '' } } as never
  })
  return clock
}

const start = { cwd: '/src/repo', surface: 'terminal', isInteractive: true } as const

describe('report-agent-state', () => {
  test('reports the listed state at start and again when it changes', async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    expect(w.reports).toEqual(['crabswarm chat report-state working'])

    await clock.advance(2000)
    expect(w.reports).toHaveLength(1)

    w.status = 'idle'
    await clock.advance(2000)
    expect(w.reports.at(-1)).toBe('crabswarm chat report-state done')
  })

  test('re-sends an unchanged state every ten seconds', async ($, on) => {
    const w = world({ CRABSWARM_CHAT_TOKEN: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    await clock.advance(10_000)
    expect(w.reports).toEqual([
      'crabswarm chat report-state working',
      'crabswarm chat report-state working',
    ])
  })

  test('follows the session id after it changes in the same process', async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    w.sessionId = 's2'
    w.status = 'idle'
    await clock.advance(2000)
    expect(w.reports.at(-1)).toBe('crabswarm chat report-state done')
  })

  test('stays quiet when no required variable is set', async ($, on) => {
    const w = world({})
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.advance(10_000)
    expect(w.listings).toBe(0)
    expect(w.reports).toHaveLength(0)
  })

  test(
    'runs the configured command, gated on the configured variables',
    { options: { reportCommand: ['my-tool', 'state', 'set'], requireEnv: ['MY_TOKEN'] } },
    async ($, on) => {
      const w = world({ MY_TOKEN: 'x' })
      const clock = engine(on, w)
      await $.session.start(start)
      await clock.settle()
      expect(w.reports).toEqual(['my-tool state set working'])
    },
  )

  test('always runs with no required variables', { options: { requireEnv: [] } }, async ($, on) => {
    const w = world({})
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    expect(w.reports).toEqual(['crabswarm chat report-state working'])
  })

  test('does nothing with an empty report command', { options: { reportCommand: [] } }, async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.advance(10_000)
    expect(w.listings).toBe(0)
    expect(w.reports).toHaveLength(0)
  })
})
