// lineSplitter cuts a child's output into lines. A piece of output ends
// wherever the child's write did, so a line may span pieces and a piece may
// hold several lines; push keeps the unfinished tail for the next piece.
//
// A line loses its "\n" and a "\r" before it.
export function lineSplitter() {
  let tail = ''
  return {
    push(text: string): string[] {
      const parts = (tail + text).split('\n')
      tail = parts.pop() ?? ''
      return parts.map(trimCR)
    },
    // end answers the text after the last "\n", once: what a child wrote
    // before it exited without ending its last line.
    end(): string | undefined {
      const rest = tail
      tail = ''
      return rest === '' ? undefined : trimCR(rest)
    },
  }
}

// splitLines cuts a whole output into its lines, the last one whether or not
// it ends in "\n".
export function splitLines(text: string): string[] {
  const s = lineSplitter()
  const lines = s.push(text)
  const rest = s.end()
  return rest === undefined ? lines : [...lines, rest]
}

function trimCR(line: string): string {
  return line.endsWith('\r') ? line.slice(0, -1) : line
}
