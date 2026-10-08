import { describe, expect, test } from 'claude-code/testing'
import { type Run, anyEnvSet } from '../hooks/host'

function printenv(env: Record<string, string>) {
  const asked: string[] = []
  const run: Run = async argv => {
    const name = argv[1] ?? ''
    asked.push(name)
    const value = env[name]
    return { exitCode: value === undefined ? 1 : 0, stdout: value === undefined ? '' : value + '\n', stderr: '' }
  }
  return { run, asked }
}

describe('anyEnvSet', () => {
  test('answers yes once one variable is set', async () => {
    const p = printenv({ B: 'tok' })
    expect(await anyEnvSet(p.run, ['A', 'B', 'C'])).toBe(true)
    expect(p.asked).toEqual(['A', 'B'])
  })

  test('answers no when every variable is unset or empty', async () => {
    const p = printenv({ A: '' })
    expect(await anyEnvSet(p.run, ['A', 'B'])).toBe(false)
  })

  test('answers yes for no names without asking', async () => {
    const p = printenv({})
    expect(await anyEnvSet(p.run, [])).toBe(true)
    expect(p.asked).toEqual([])
  })
})
