// The state words the report command is handed.
export type HarnessState = 'working' | 'waiting' | 'done'

// One entry of `claude agents --json`, the fields a state is read from.
type ClaudeAgent = {
  pid?: number
  sessionId?: string
  status?: string
  state?: string
}

// stateOf picks the entry of sessionId out of a listing and maps it onto a
// state word. It answers undefined when the listing does not carry
// the session or says nothing it can map.
//
// An entry without a pid is never picked. Those come from the background job
// records under <config home>/jobs/, which describe sessions running elsewhere
// or not at all; a session resumed from a job leaves the job's record behind
// under the same id, often with a stale blocked state.
export function stateOf(listing: string, sessionId: string): HarnessState | undefined {
  let entries: unknown
  try {
    entries = JSON.parse(listing)
  } catch {
    return undefined
  }
  if (!Array.isArray(entries)) return undefined
  const entry = (entries as ClaudeAgent[]).find(
    a => typeof a === 'object' && a !== null && !!a.pid && a.sessionId === sessionId,
  )
  return entry === undefined ? undefined : stateOfEntry(entry)
}

// status is what the session is doing at this moment and is read first. An
// entry with no status is a session whose process the listing could not see,
// and its state, the session's own account of its progress, is the fallback.
function stateOfEntry(a: ClaudeAgent): HarnessState | undefined {
  switch (a.status) {
    case 'busy':
      return 'working'
    case 'waiting':
      return 'waiting'
    case 'idle':
      return 'done'
    case undefined:
    case '':
      break
    default:
      return undefined
  }
  switch (a.state) {
    case 'working':
      return 'working'
    case 'blocked':
      return 'waiting'
    case 'done':
    case 'failed':
    case 'stopped':
      return 'done'
    default:
      return undefined
  }
}
