import { describe, expect, test } from 'claude-code/testing'
import { lineSplitter, splitLines } from '../hooks/lines'

describe('lineSplitter', () => {
  test('joins a line written across pieces', () => {
    const s = lineSplitter()
    expect(s.push('{"type":')).toEqual([])
    expect(s.push('"status"}')).toEqual([])
    expect(s.push('\n')).toEqual(['{"type":"status"}'])
  })

  test('cuts a piece holding several lines and keeps the unfinished tail', () => {
    const s = lineSplitter()
    expect(s.push('a\nb\r\nc')).toEqual(['a', 'b'])
    expect(s.push('d\n\n')).toEqual(['cd', ''])
  })

  test('hands the unfinished tail over once at the end', () => {
    const s = lineSplitter()
    s.push('a\nb')
    expect(s.end()).toBe('b')
    expect(s.end()).toBeUndefined()
  })
})

describe('splitLines', () => {
  test('answers every line, the last one with or without its newline', () => {
    expect(splitLines('a\nb\n')).toEqual(['a', 'b'])
    expect(splitLines('a\r\nb')).toEqual(['a', 'b'])
    expect(splitLines('')).toEqual([])
  })
})
