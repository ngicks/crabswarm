import { describe, expect, test } from 'claude-code/testing'
import { stateOf } from '../hooks/state'

const SESSION = '43b36af6-9b3b-4985-b31d-39f5d8cbda7d'

const listing = (...entries: object[]) => JSON.stringify(entries)

describe('stateOf', () => {
  test('maps the live status of the session', () => {
    expect(stateOf(listing({ pid: 2, sessionId: SESSION, status: 'busy' }), SESSION)).toBe('working')
    expect(stateOf(listing({ pid: 2, sessionId: SESSION, status: 'waiting', waitingFor: 'permission prompt' }), SESSION)).toBe('waiting')
    expect(stateOf(listing({ pid: 2, sessionId: SESSION, status: 'idle' }), SESSION)).toBe('done')
  })

  test('falls back to the progress state when there is no status', () => {
    expect(stateOf(listing({ pid: 2, sessionId: SESSION, state: 'working' }), SESSION)).toBe('working')
    expect(stateOf(listing({ pid: 2, sessionId: SESSION, state: 'blocked' }), SESSION)).toBe('waiting')
    expect(stateOf(listing({ pid: 2, sessionId: SESSION, state: 'stopped' }), SESSION)).toBe('done')
  })

  test('skips job records that carry no pid', () => {
    const l = listing(
      { sessionId: SESSION, kind: 'background', state: 'blocked' },
      { pid: 2, sessionId: SESSION, kind: 'interactive', status: 'idle' },
    )
    expect(stateOf(l, SESSION)).toBe('done')
    expect(stateOf(listing({ sessionId: SESSION, state: 'blocked' }), SESSION)).toBeUndefined()
  })

  test('answers nothing for another session, an unknown word or a broken listing', () => {
    expect(stateOf(listing({ pid: 2, sessionId: 'other', status: 'busy' }), SESSION)).toBeUndefined()
    expect(stateOf(listing({ pid: 2, sessionId: SESSION, status: 'sleeping' }), SESSION)).toBeUndefined()
    expect(stateOf('not json', SESSION)).toBeUndefined()
    expect(stateOf('{}', SESSION)).toBeUndefined()
  })
})
