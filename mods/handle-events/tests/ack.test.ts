import { describe, expect, test } from 'claude-code/testing'
import { ackRecords, fillArgv } from '../hooks/ack'
import type { Run } from '../hooks/host'
import type { EventRecord } from '../hooks/protocol'

const record = (fields: object): EventRecord => ({ type: 'message', message: 'alice: hi', ...fields })

describe('fillArgv', () => {
  test('replaces each placeholder item with the field as text', () => {
    expect(fillArgv(['ack', '{seq}', '{room}', '{inject}'], record({ seq: 5, room: 'r1', inject: true }))).toEqual({
      argv: ['ack', '5', 'r1', 'true'],
    })
  })

  test('leaves an item that is not exactly a placeholder as written', () => {
    expect(fillArgv(['--seq={seq}', '{seq}x', '{a}{b}', '{}'], record({ seq: 5 }))).toEqual({
      argv: ['--seq={seq}', '{seq}x', '{a}{b}', '{}'],
    })
  })

  test('names a field that is missing or not a string, number or boolean', () => {
    expect(fillArgv(['ack', '{seq}'], record({}))).toEqual({ missing: 'seq' })
    expect(fillArgv(['{seq}'], record({ seq: null }))).toEqual({ missing: 'seq' })
    expect(fillArgv(['{seq}'], record({ seq: { n: 5 } }))).toEqual({ missing: 'seq' })
    expect(fillArgv(['{seq}'], record({ seq: [5] }))).toEqual({ missing: 'seq' })
    expect(fillArgv(['{toString}'], record({}))).toEqual({ missing: 'toString' })
  })
})

function fakeRun(exitCodes: Record<string, number> = {}) {
  const ran: string[] = []
  const run: Run = async argv => {
    const line = argv.join(' ')
    ran.push(line)
    if (line.includes('crash')) throw new Error('cannot start')
    const exitCode = exitCodes[line] ?? 0
    return { exitCode, stdout: '', stderr: exitCode === 0 ? '' : 'refused\n' }
  }
  return { run, ran }
}

describe('ackRecords', () => {
  test('runs the command once per record, in order', async () => {
    const r = fakeRun()
    let acked = 0
    await ackRecords([record({ seq: 1 }), record({ seq: 2 }), record({ seq: 3 })], {
      run: r.run,
      argv: ['ack', '{seq}'],
      onAcked: () => acked++,
    })
    expect(r.ran).toEqual(['ack 1', 'ack 2', 'ack 3'])
    expect(acked).toBe(3)
  })

  test('skips a record without the field and reports failed runs, going on with the rest', async () => {
    const r = fakeRun({ 'ack 2': 1 })
    const errors: string[] = []
    await ackRecords([record({}), record({ seq: 2 }), record({ seq: 'crash' }), record({ seq: 4 })], {
      run: r.run,
      argv: ['ack', '{seq}'],
      onError: m => errors.push(m),
    })
    expect(r.ran).toEqual(['ack 2', 'ack crash', 'ack 4'])
    expect(errors).toEqual([
      'ack skipped: the record has no string, number or boolean "seq": alice: hi',
      'ack 2 exited 1: refused',
      'ack crash: cannot start',
    ])
  })

  test('acks nothing with an empty command', async () => {
    const r = fakeRun()
    await ackRecords([record({ seq: 1 })], { run: r.run, argv: [] })
    expect(r.ran).toEqual([])
  })
})
