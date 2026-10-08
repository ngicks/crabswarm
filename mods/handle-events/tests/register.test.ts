import { describe, expect, mock, test } from 'claude-code/testing'
import type { On } from 'claude-code'

// One spawned source: the test prints lines into it and ends it.
type Source = {
  argv: string
  print: (...records: object[]) => void
  write: (text: string) => void
  exit: (code: number) => void
}

type World = {
  env: Record<string, string>
  sources: Source[]
  prompts: string[]
  statuses: (string | undefined)[]
  logs: string[]
  // Every process.run but printenv, as one line each.
  runs: string[]
  mapper?: (stdin: string) => { exitCode: number; stdout: string }
  dropPrompts?: string
}

const world = (env: Record<string, string>): World => ({
  env,
  sources: [],
  prompts: [],
  statuses: [],
  logs: [],
  runs: [],
})

function engine(on: On, w: World) {
  const clock = mock.clock(on)
  on('session.start', (_$, e) => ({ cwd: e.cwd }))
  on('turn.start', (_$, e) => ({ turnId: e.turnId }))
  on('turn.complete', (_$, e) => ({ text: e.answer }))
  on('prompt.submit', (_$, e) => {
    if (w.dropPrompts !== undefined) return { drop: w.dropPrompts }
    w.prompts.push(e.text)
    return { text: e.text }
  })
  on('ui.status', (_$, e) => {
    w.statuses.push(e.text)
    return { value: undefined } as never
  })
  on('ui.log', (_$, e) => {
    w.logs.push(e.text)
    return { value: undefined } as never
  })
  on('process.run', (_$, e) => {
    const [cmd, ...args] = e.argv
    if (cmd === 'printenv') {
      const value = w.env[args[0] ?? '']
      return { value: { exitCode: value === undefined ? 1 : 0, stdout: value ?? '', stderr: '' } } as never
    }
    w.runs.push(e.argv.join(' '))
    const answer = cmd === 'mapper' && w.mapper ? w.mapper(e.init?.stdin ?? '') : { exitCode: 0, stdout: '' }
    return { value: { ...answer, stderr: '' } } as never
  })
  on('process.spawn', async function* (_$, e) {
    const items: ({ text: string } | { code: number })[] = []
    let wake: (() => void) | undefined
    const push = (item: { text: string } | { code: number }) => {
      items.push(item)
      wake?.()
    }
    w.sources.push({
      argv: e.argv.join(' '),
      print: (...records) => push({ text: records.map(r => JSON.stringify(r) + '\n').join('') }),
      write: text => push({ text }),
      exit: code => push({ code }),
    })
    for (;;) {
      while (items.length === 0) await new Promise<void>(resolve => (wake = resolve))
      const item = items.shift()!
      if ('code' in item) return { value: { code: item.code, signal: null } } as never
      yield { stream: 'stdout' as const, text: item.text }
    }
  })
  return clock
}

const start = { cwd: '/src/repo', surface: 'terminal', isInteractive: true } as const
const turn = { answer: '', durationMs: 1, isAborted: false, reason: 'answer' } as const
const mention = (from: string, text: string, seq: number) => ({
  type: 'message',
  message: `${from}: ${text}`,
  inject: true,
  mentioned_you: true,
  seq,
})

describe('handle-events', () => {
  test('runs the source and shows its status records', async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    expect(w.sources.map(s => s.argv)).toEqual(['crabswarm chat follow'])
    w.sources[0]?.print({ type: 'status', message: 'following room' }, { type: 'status', message: '' })
    await clock.settle()
    expect(w.statuses).toEqual(['following room', undefined])
  })

  test('submits a flagged message at once while idle, then acks it', async ($, on) => {
    const w = world({ CRABSWARM_CHAT_TOKEN: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    w.sources[0]?.print(mention('alice', 'hi', 5))
    await clock.settle()
    expect(w.prompts).toEqual(['[crabswarm chat] alice: hi'])
    expect(w.runs).toEqual(['crabswarm chat read --skip 5'])
  })

  test('holds messages while the main turn runs and submits them as one prompt when it ends', async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    await $.turn.start({ text: 'work', turnId: 't1' })
    w.sources[0]?.print(mention('alice', 'hi', 5), mention('bob', 'yo', 6))
    w.sources[0]?.print(mention('alice', 'hi', 7))
    await clock.settle()
    expect(w.prompts).toEqual([])

    await $.turn.complete({ ...turn, turnId: 't1', agentId: 'sub' })
    await clock.settle()
    expect(w.prompts).toEqual([])

    await $.turn.complete({ ...turn, turnId: 't1' })
    await clock.settle()
    expect(w.prompts).toEqual(['[crabswarm chat] alice: hi\n[crabswarm chat] bob: yo'])
    expect(w.runs).toEqual([
      'crabswarm chat read --skip 5',
      'crabswarm chat read --skip 6',
      'crabswarm chat read --skip 7',
    ])
  })

  test('ignores unflagged messages, unknown types and lines that are not records', async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    w.sources[0]?.print(
      { type: 'message', message: 'carol: chatter', inject: false, seq: 1 },
      { type: 'typing', message: 'dave', inject: true },
    )
    w.sources[0]?.write('not json\n{"type":"message","inject":true}\n')
    await clock.settle()
    expect(w.prompts).toEqual([])
    expect(w.runs).toEqual([])
    expect(w.logs).toEqual(['handle-events: not a record: "not json" (drops: 1)'])
  })

  test('maps each line through the mapper', { options: { mapper: ['mapper'] } }, async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    w.mapper = stdin =>
      stdin.startsWith('drop')
        ? { exitCode: 1, stdout: '' }
        : { exitCode: 0, stdout: JSON.stringify({ type: 'message', message: stdin.trim(), inject: true, seq: 9 }) + '\n' }
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    w.sources[0]?.write('drop this\nraw event\n')
    await clock.settle()
    expect(w.prompts).toEqual(['[crabswarm chat] raw event'])
    expect(w.runs).toEqual(['mapper', 'mapper', 'crabswarm chat read --skip 9'])
    expect(w.logs).toEqual(['handle-events: mapper exited 1 on "drop this" (drops: 1)'])
  })

  test(
    'fills the configured ack from each record and skips one without the field',
    { options: { ack: ['ack', '{room}', '{seq}'], prefix: '>>' } },
    async ($, on) => {
      const w = world({ CMDMAN_CMD_ID: 'tok' })
      const clock = engine(on, w)
      await $.session.start(start)
      await clock.settle()
      await $.turn.start({ text: 'work', turnId: 't1' })
      w.sources[0]?.print({ ...mention('alice', 'hi', 5), room: 'r1' }, mention('bob', 'yo', 6))
      await clock.settle()
      await $.turn.complete({ ...turn, turnId: 't1' })
      await clock.settle()
      expect(w.prompts).toEqual(['>> alice: hi\n>> bob: yo'])
      expect(w.runs).toEqual(['ack r1 5'])
      expect(w.logs).toEqual([
        'handle-events: ack skipped: the record has no string, number or boolean "room": bob: yo (ack failures: 1)',
      ])
    },
  )

  test('acks nothing when the prompt is dropped', async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    w.dropPrompts = 'refused'
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    w.sources[0]?.print(mention('alice', 'hi', 5))
    await clock.settle()
    expect(w.runs).toEqual([])
    expect(w.logs).toEqual(['handle-events: 1 record(s) not injected: the prompt was dropped: refused (drops: 1)'])
  })

  test('restarts an exited source with a growing pause and says so on the status line', async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.settle()
    w.sources[0]?.exit(1)
    await clock.settle()
    expect(w.statuses).toEqual(['handle-events: source exited with code 1; restarting in 1s'])

    await clock.advance(999)
    expect(w.sources).toHaveLength(1)
    await clock.advance(1)
    expect(w.sources).toHaveLength(2)

    w.sources[1]?.exit(1)
    await clock.settle()
    expect(w.statuses.at(-1)).toBe('handle-events: source exited with code 1; restarting in 2s')
    await clock.advance(1999)
    expect(w.sources).toHaveLength(2)
    await clock.advance(1)
    expect(w.sources).toHaveLength(3)

    w.sources[2]?.print({ type: 'status', message: 'following room' })
    await clock.settle()
    expect(w.statuses.slice(-2)).toEqual([undefined, 'following room'])
  })

  test('stays quiet when no required variable is set', async ($, on) => {
    const w = world({})
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.advance(60_000)
    expect(w.sources).toHaveLength(0)
  })

  test('always runs with no required variables, one source however often the session starts', { options: { requireEnv: [] } }, async ($, on) => {
    const w = world({})
    const clock = engine(on, w)
    await $.session.start(start)
    await $.session.start(start)
    await clock.settle()
    expect(w.sources).toHaveLength(1)
  })

  test('does nothing with an empty source', { options: { source: [] } }, async ($, on) => {
    const w = world({ CMDMAN_CMD_ID: 'tok' })
    const clock = engine(on, w)
    await $.session.start(start)
    await clock.advance(60_000)
    expect(w.sources).toHaveLength(0)
  })
})
