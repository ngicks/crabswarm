import { describe, expect, test } from 'claude-code/testing'
import { actionOf, parseRecord } from '../hooks/protocol'

describe('parseRecord', () => {
  test('reads an object with a string type and message, keeping every field', () => {
    expect(parseRecord('{"type":"message","message":"alice: hi","inject":true,"seq":5,"room":{"id":"r"}}')).toEqual({
      type: 'message',
      message: 'alice: hi',
      inject: true,
      seq: 5,
      room: { id: 'r' },
    })
  })

  test('refuses anything else', () => {
    expect(parseRecord('not json')).toBeUndefined()
    expect(parseRecord('')).toBeUndefined()
    expect(parseRecord('null')).toBeUndefined()
    expect(parseRecord('"text"')).toBeUndefined()
    expect(parseRecord('[{"type":"status","message":"x"}]')).toBeUndefined()
    expect(parseRecord('{"message":"x"}')).toBeUndefined()
    expect(parseRecord('{"type":"status"}')).toBeUndefined()
    expect(parseRecord('{"type":1,"message":"x"}')).toBeUndefined()
    expect(parseRecord('{"type":"status","message":{"text":"x"}}')).toBeUndefined()
  })
})

describe('actionOf', () => {
  const record = (fields: object) => ({ type: 'message', message: 'm', ...fields })

  test('shows a status record', () => {
    expect(actionOf({ type: 'status', message: 'following room' })).toBe('status')
  })

  test('injects a message flagged inject true', () => {
    expect(actionOf(record({ inject: true }))).toBe('inject')
  })

  test('ignores a message not flagged, and an unknown type', () => {
    expect(actionOf(record({}))).toBeUndefined()
    expect(actionOf(record({ inject: false }))).toBeUndefined()
    expect(actionOf(record({ inject: 'true' }))).toBeUndefined()
    expect(actionOf({ type: 'typing', message: 'alice', inject: true })).toBeUndefined()
  })
})
