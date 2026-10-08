import { describe, expect, test } from 'claude-code/testing'
import type { Run, SpawnChunk, SpawnEnd } from '../hooks/host'
import type { EventRecord } from '../hooks/protocol'
import { recordsOf, runSource } from '../hooks/source'

const out = (text: string): SpawnChunk => ({ stream: 'stdout', text })
const err = (text: string): SpawnChunk => ({ stream: 'stderr', text })
const status = (message: string) => JSON.stringify({ type: 'status', message }) + '\n'

const noRun: Run = async () => {
  throw new Error('no mapper expected')
}

// mapper answers each stdin by the function given, and records what it ran.
function mapper(answer: (stdin: string) => { exitCode: number; stdout: string }) {
  const calls: { argv: string; stdin: string | undefined }[] = []
  const run: Run = async (argv, init) => {
    calls.push({ argv: argv.join(' '), stdin: init?.stdin })
    return { ...answer(init?.stdin ?? ''), stderr: 'mapper says no\n' }
  }
  return { run, calls }
}

describe('recordsOf', () => {
  test('reads the line as the record without a mapper', async () => {
    expect(await recordsOf('{"type":"status","message":"x"}', { mapper: [], run: noRun })).toEqual([
      { type: 'status', message: 'x' },
    ])
  })

  test('drops a line that is not a record, and skips a blank one silently', async () => {
    const drops: string[] = []
    expect(await recordsOf('{"type":"status"}', { mapper: [], run: noRun, onDrop: m => drops.push(m) })).toEqual([])
    expect(await recordsOf('  ', { mapper: [], run: noRun, onDrop: m => drops.push(m) })).toEqual([])
    expect(drops).toEqual(['not a record: "{\\"type\\":\\"status\\"}"'])
  })

  test('hands the line to the mapper and reads each line it prints as a record', async () => {
    const m = mapper(() => ({ exitCode: 0, stdout: status('a') + '\n' + status('b') + 'oops\n' }))
    const drops: string[] = []
    const records = await recordsOf('raw event', { mapper: ['jq', '-c', '.'], run: m.run, onDrop: d => drops.push(d) })
    expect(records).toEqual([
      { type: 'status', message: 'a' },
      { type: 'status', message: 'b' },
    ])
    expect(m.calls).toEqual([{ argv: 'jq -c .', stdin: 'raw event\n' }])
    expect(drops).toEqual(['not a record: "oops"'])
  })

  test('drops the line when the mapper exits non-zero', async () => {
    const m = mapper(() => ({ exitCode: 2, stdout: status('a') }))
    const drops: string[] = []
    expect(await recordsOf('raw event', { mapper: ['map'], run: m.run, onDrop: d => drops.push(d) })).toEqual([])
    expect(drops).toEqual(['mapper exited 2 on "raw event": mapper says no'])
  })
})

type Script = { chunks?: SpawnChunk[]; end?: SpawnEnd; throws?: string; ranMs?: number }

// fakeSource plays one script per spawn, moving its clock by each script's
// ranMs, and stops runSource once the scripts run out.
function fakeSource(scripts: Script[]) {
  let now = 0
  const ctl = new AbortController()
  const waits: number[] = []
  const restarts: string[] = []
  const records: EventRecord[] = []
  const events: string[] = []
  let spawned = 0
  return {
    opts: {
      argv: ['follow'],
      mapper: [],
      run: noRun,
      spawn: (argv: readonly string[]) => {
        spawned++
        const s = scripts.shift()
        return (async function* (): AsyncGenerator<SpawnChunk, SpawnEnd> {
          expect(argv).toEqual(['follow'])
          if (s === undefined) {
            ctl.abort()
            return { code: 0, signal: null }
          }
          if (s.throws !== undefined) throw new Error(s.throws)
          for (const c of s.chunks ?? []) yield c
          now += s.ranMs ?? 0
          return s.end ?? { code: 0, signal: null }
        })()
      },
      now: async () => now,
      wait: async (ms: number) => void waits.push(ms),
      onRecord: (r: EventRecord) => {
        records.push(r)
        events.push(`record ${r.message}`)
      },
      onRestart: (reason: string, delayMs: number) => {
        restarts.push(reason)
        events.push(`restart ${delayMs}`)
      },
      onRecovered: () => void events.push('recovered'),
      signal: ctl.signal,
    },
    waits,
    restarts,
    records,
    events,
    spawned: () => spawned,
  }
}

const exit = (code: number): Script => ({ end: { code, signal: null } })

describe('runSource', () => {
  test('reads records out of lines cut across pieces, the last one unterminated', async () => {
    const f = fakeSource([
      { chunks: [out('{"type":"status",'), out('"message":"a"}\n{"type":"status","message":"b"}\n'), out(status('c').trim())] },
    ])
    await runSource(f.opts)
    expect(f.records.map(r => r.message)).toEqual(['a', 'b', 'c'])
  })

  test('restarts with a pause doubling from one second up to thirty', async () => {
    const f = fakeSource(Array.from({ length: 8 }, () => exit(1)))
    await runSource(f.opts)
    expect(f.waits).toEqual([1000, 2000, 4000, 8000, 16000, 30_000, 30_000, 30_000])
    expect(f.spawned()).toBe(9)
  })

  test('waits the shortest pause again after a run that printed a record', async () => {
    const f = fakeSource([exit(1), exit(1), { chunks: [out(status('up'))], end: { code: 1, signal: null } }, exit(1)])
    await runSource(f.opts)
    expect(f.waits).toEqual([1000, 2000, 1000, 2000])
  })

  test('waits the shortest pause again after a run that lasted', async () => {
    const f = fakeSource([exit(1), exit(1), { ranMs: 30_000, end: { code: 0, signal: null } }, exit(1)])
    await runSource(f.opts)
    expect(f.waits).toEqual([1000, 2000, 1000, 2000])
  })

  test('says why the source ended, with the last line it wrote to stderr', async () => {
    const f = fakeSource([
      { chunks: [err('dialing\nconnection refused\n')], end: { code: 1, signal: null } },
      { end: { code: null, signal: 'SIGTERM' } },
      { throws: 'executable file not found' },
    ])
    await runSource(f.opts)
    expect(f.restarts).toEqual([
      'source exited with code 1: connection refused',
      'source ended by SIGTERM',
      'source failed: executable file not found',
    ])
  })

  test('reports recovery on the first record after a restart, before handing it over', async () => {
    const f = fakeSource([
      { chunks: [out(status('first'))], end: { code: 1, signal: null } },
      { chunks: [out(status('second')), out(status('third'))], end: { code: 1, signal: null } },
    ])
    await runSource(f.opts)
    expect(f.events).toEqual([
      'record first',
      'restart 1000',
      'recovered',
      'record second',
      'record third',
      'restart 1000',
    ])
  })

  test('runs each line through the mapper in order', async () => {
    const m = mapper(stdin => (stdin.startsWith('skip') ? { exitCode: 1, stdout: '' } : { exitCode: 0, stdout: status(stdin.trim()) }))
    const f = fakeSource([{ chunks: [out('one\nskip\ntwo\n')] }])
    const drops: string[] = []
    await runSource({ ...f.opts, mapper: ['map'], run: m.run, onDrop: d => drops.push(d) })
    expect(f.records.map(r => r.message)).toEqual(['one', 'two'])
    expect(drops).toEqual(['mapper exited 1 on "skip": mapper says no'])
  })
})
