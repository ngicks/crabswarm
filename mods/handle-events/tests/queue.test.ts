import { describe, expect, test } from 'claude-code/testing'
import type { EventRecord } from '../hooks/protocol'
import { injectQueue, promptText } from '../hooks/queue'

const msg = (message: string, seq: number): EventRecord => ({ type: 'message', message, inject: true, seq })

// Lets the queue's unawaited flushes run to their next wait.
async function drain() {
  for (let i = 0; i < 20; i++) await Promise.resolve()
}

// fakeSession holds each submit until the test answers it, and records what
// was handed to onSubmitted.
function fakeSession() {
  const submits: { text: string; accept: () => void; drop: (why: string) => void }[] = []
  const submitted: number[][] = []
  const errors: string[] = []
  const queue = injectQueue({
    prefix: '[chat]',
    submit: text =>
      new Promise<void>((resolve, reject) => {
        submits.push({ text, accept: resolve, drop: why => reject(new Error(why)) })
      }),
    onSubmitted: async records => void submitted.push(records.map(r => r.seq as number)),
    onError: m => errors.push(m),
  })
  return { queue, submits, submitted, errors }
}

describe('promptText', () => {
  test('writes one line per record, prefixed, leaving out repeated lines', () => {
    expect(promptText('[chat]', [msg('alice: hi', 1), msg('bob: yo', 2), msg('alice: hi', 3)])).toBe(
      '[chat] alice: hi\n[chat] bob: yo',
    )
  })

  test('writes the message alone with an empty prefix', () => {
    expect(promptText('', [msg('alice: hi', 1)])).toBe('alice: hi')
  })
})

describe('injectQueue', () => {
  test('submits at once while idle, and hands over the records once it entered', async () => {
    const s = fakeSession()
    s.queue.push(msg('alice: hi', 5))
    await drain()
    expect(s.submits.map(x => x.text)).toEqual(['[chat] alice: hi'])
    expect(s.submitted).toEqual([])
    s.submits[0]?.accept()
    await drain()
    expect(s.submitted).toEqual([[5]])
  })

  test('holds records while a turn runs and submits them as one prompt when it ends', async () => {
    const s = fakeSession()
    s.queue.turnStarted()
    s.queue.push(msg('alice: hi', 1))
    s.queue.push(msg('bob: yo', 2))
    s.queue.push(msg('alice: hi', 3))
    await drain()
    expect(s.submits).toHaveLength(0)
    s.queue.turnCompleted()
    await drain()
    expect(s.submits.map(x => x.text)).toEqual(['[chat] alice: hi\n[chat] bob: yo'])
    s.submits[0]?.accept()
    await drain()
    expect(s.submitted).toEqual([[1, 2, 3]])
  })

  test('holds records that arrive while a prompt is on its way in until its turn ends', async () => {
    const s = fakeSession()
    s.queue.push(msg('alice: hi', 1))
    await drain()
    s.queue.push(msg('bob: yo', 2))
    await drain()
    expect(s.submits).toHaveLength(1)
    s.submits[0]?.accept()
    await drain()
    expect(s.submits).toHaveLength(1)
    s.queue.turnStarted()
    s.queue.turnCompleted()
    await drain()
    expect(s.submits.map(x => x.text)).toEqual(['[chat] alice: hi', '[chat] bob: yo'])
  })

  test('does not wait on a turn that started and ended before the submit resolved', async () => {
    const s = fakeSession()
    s.queue.push(msg('alice: hi', 1))
    await drain()
    s.queue.push(msg('bob: yo', 2))
    s.queue.turnStarted()
    s.queue.turnCompleted()
    s.submits[0]?.accept()
    await drain()
    expect(s.submits.map(x => x.text)).toEqual(['[chat] alice: hi', '[chat] bob: yo'])
  })

  test('reports a dropped prompt, hands nothing over, and goes on with the next record', async () => {
    const s = fakeSession()
    s.queue.push(msg('alice: hi', 1))
    await drain()
    s.submits[0]?.drop('a hook refused it')
    await drain()
    expect(s.errors).toEqual(['1 record(s) not injected: a hook refused it'])
    expect(s.submitted).toEqual([])
    s.queue.push(msg('bob: yo', 2))
    await drain()
    expect(s.submits.map(x => x.text)).toEqual(['[chat] alice: hi', '[chat] bob: yo'])
  })
})
