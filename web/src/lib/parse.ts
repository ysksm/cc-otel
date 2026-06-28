// Centralized helpers for the backend's JSON-encoded-as-string fields.

/** Parse a JSON-string array like "[\"gpt-4o\"]" into a string[]; tolerant of junk. */
export function parseJsonArray(s: string | undefined | null): string[] {
  if (!s) return []
  try {
    const v = JSON.parse(s)
    if (Array.isArray(v)) return v.map((x) => String(x)).filter((x) => x.length > 0)
    return []
  } catch {
    return []
  }
}

/** Parse an arbitrary JSON-string field into an unknown value; returns undefined on failure. */
export function parseJson<T = unknown>(s: string | undefined | null): T | undefined {
  if (s == null || s === '') return undefined
  try {
    return JSON.parse(s) as T
  } catch {
    return undefined
  }
}

export interface ChatMessage {
  role: string
  content: unknown // may be a string or an array of content parts
}

/**
 * Parse inputMessages / outputMessages JSON strings into an array of
 * {role, content}. Content may be a string or array; left as-is for the
 * renderer to handle.
 */
export function parseMessages(s: string | undefined | null): ChatMessage[] {
  const v = parseJson<unknown>(s)
  if (!Array.isArray(v)) return []
  const out: ChatMessage[] = []
  for (const item of v) {
    if (item && typeof item === 'object') {
      const rec = item as Record<string, unknown>
      const role = typeof rec.role === 'string' ? rec.role : 'unknown'
      out.push({ role, content: rec.content })
    } else if (typeof item === 'string') {
      out.push({ role: 'unknown', content: item })
    }
  }
  return out
}

/**
 * Render message content (string or array of parts) into a flat display string.
 * Handles common content-part shapes: {type:'text', text}, {text}, tool_use/tool_result.
 */
export function contentToText(content: unknown): string {
  if (content == null) return ''
  if (typeof content === 'string') return content
  if (Array.isArray(content)) {
    return content
      .map((part) => {
        if (typeof part === 'string') return part
        if (part && typeof part === 'object') {
          const p = part as Record<string, unknown>
          if (typeof p.text === 'string') return p.text
          if (p.type === 'text' && typeof p.text === 'string') return p.text
          if (p.type === 'tool_use') {
            const name = typeof p.name === 'string' ? p.name : 'tool'
            return `[tool_use: ${name}] ${JSON.stringify(p.input ?? {})}`
          }
          if (p.type === 'tool_result') {
            return `[tool_result] ${stringifyUnknown(p.content)}`
          }
          return JSON.stringify(p)
        }
        return String(part)
      })
      .join('\n')
  }
  return stringifyUnknown(content)
}

export function stringifyUnknown(v: unknown): string {
  if (v == null) return ''
  if (typeof v === 'string') return v
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}

/** Pretty-print a parsed JSON string field, or return the raw string if unparsable. */
export function prettyJson(s: string | undefined | null): string {
  if (s == null || s === '') return ''
  const v = parseJson(s)
  if (v === undefined) return s
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return s
  }
}
