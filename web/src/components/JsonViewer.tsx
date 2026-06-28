import { useState } from 'react'
import { prettyJson } from '../lib/parse'

/** A collapsible raw JSON viewer for a JSON-string field. */
export function JsonViewer({
  label,
  raw,
  defaultOpen = false,
}: {
  label: string
  raw: string | undefined | null
  defaultOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  const pretty = prettyJson(raw)
  const empty = !pretty || pretty === 'null' || pretty === '{}' || pretty === '[]'

  return (
    <div className="json-viewer">
      <button className="json-toggle" onClick={() => setOpen((o) => !o)} disabled={empty}>
        <span className="caret">{open ? '▾' : '▸'}</span> {label}
        {empty && <span className="dim"> (empty)</span>}
      </button>
      {open && !empty && <pre className="json-block">{pretty}</pre>}
    </div>
  )
}
