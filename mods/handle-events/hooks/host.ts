// The engine calls the helpers make. The engine refuses a $ passed across an
// import, so register.tsx hands in closures over its own $ instead.
export type Run = (
  argv: readonly string[],
  init?: { stdin?: string; timeoutMs?: number },
) => Promise<{ exitCode: number; stdout: string; stderr: string }>

export type SpawnChunk = { stream: 'stdout' | 'stderr'; text: string }
export type SpawnEnd = { code: number | null; signal: string | null }

// Spawn starts argv and streams its output; the iterator's return value is
// how the child ended. Leaving the iteration early kills the child.
export type Spawn = (argv: readonly string[]) => AsyncIterator<SpawnChunk, SpawnEnd | void>

export type Now = () => Promise<number>

// Wait resolves after ms. register.tsx builds it on $.clock.after rather than
// $.clock.sleep: a sleep is charged to the budget of the hook that started
// it, while a timer runs until the module reloads.
export type Wait = (ms: number) => Promise<void>

// withStderr appends what a command wrote to stderr to a message about it.
export function withStderr(message: string, stderr: string): string {
  const s = stderr.trim()
  return s === '' ? message : `${message}: ${s}`
}

const PRINTENV_TIMEOUT_MS = 5000

// anyEnvSet reports whether the session's environment sets at least one of
// names to a non-empty value; no names at all is a yes.
//
// It asks printenv rather than $.env.get: the engine takes only a string
// literal as a variable name there, and these names come from the options.
export async function anyEnvSet(run: Run, names: readonly string[]): Promise<boolean> {
  if (names.length === 0) return true
  for (const name of names) {
    const r = await run(['printenv', name], { timeoutMs: PRINTENV_TIMEOUT_MS })
    if (r.exitCode === 0 && r.stdout.trim() !== '') return true
  }
  return false
}
