// Shared markdown rendering used by the canvas node previews and the node detail
// modal. Both render the same body text, so the conversion lives in one place.
import { marked } from 'marked'

// Render markdown breaks as line breaks so short, single-newline notes display the
// way they were typed without needing blank lines between every line.
marked.setOptions({ breaks: true, gfm: true })

// renderMarkdown converts a markdown source string to an HTML string. Empty input
// returns an empty string so callers can decide what to show in its place.
export function renderMarkdown(source: string): string {
  if (!source) {
    return ''
  }
  return marked.parse(source) as string
}
