import { useId } from 'react'

export interface BarChartProps {
  /** One value per bucket, in chronological order. */
  values: number[]
  /** Optional sparse x-axis labels, aligned to the start/end of the range. */
  startLabel?: string
  endLabel?: string
  /** Bar/area color; defaults to the accent color. */
  color?: string
  /** Render as a filled area instead of discrete bars. */
  area?: boolean
  /** Formatter for the max-value label shown at the top-left. */
  formatMax?: (v: number) => string
  /** Accessible title / hover summary for the chart. */
  label?: string
  height?: number
}

const VIEW_W = 600
const PAD_TOP = 8
const PAD_BOTTOM = 2

/**
 * Hand-rolled, dependency-free inline SVG chart. Renders a horizontal baseline,
 * scaled bars (or a filled area), and a max-value label. Responsive via a
 * viewBox + preserveAspectRatio="none"; the wrapper controls the rendered size.
 * Handles the empty / all-zero case by drawing just the baseline.
 */
export function BarChart({
  values,
  startLabel,
  endLabel,
  color = 'var(--accent-2)',
  area = false,
  formatMax,
  label,
  height = 120,
}: BarChartProps) {
  const gradId = useId()
  const max = values.reduce((m, v) => (v > m ? v : m), 0)
  const n = values.length
  const plotH = height - PAD_TOP - PAD_BOTTOM
  const baselineY = height - PAD_BOTTOM

  const scaleY = (v: number) => {
    if (max <= 0) return baselineY
    return baselineY - (v / max) * plotH
  }

  let body: React.ReactNode = null
  if (n > 0 && max > 0) {
    if (area) {
      const step = VIEW_W / Math.max(n - 1, 1)
      const points = values.map((v, i) => `${(i * step).toFixed(2)},${scaleY(v).toFixed(2)}`)
      const linePath = `M ${points.join(' L ')}`
      const fillPath =
        `M 0,${baselineY} L ${points.join(' L ')} L ${VIEW_W},${baselineY} Z`
      body = (
        <>
          <path d={fillPath} fill={`url(#${gradId})`} />
          <path d={linePath} fill="none" stroke={color} strokeWidth={1.5} vectorEffect="non-scaling-stroke" />
        </>
      )
    } else {
      const slot = VIEW_W / n
      const gap = n > 80 ? 0 : Math.min(slot * 0.2, 3)
      const barW = Math.max(slot - gap, 0.5)
      body = (
        <g fill={color}>
          {values.map((v, i) => {
            const y = scaleY(v)
            const h = baselineY - y
            return (
              <rect
                key={i}
                x={i * slot + gap / 2}
                y={y}
                width={barW}
                height={Math.max(h, v > 0 ? 1 : 0)}
                rx={barW > 4 ? 1 : 0}
              />
            )
          })}
        </g>
      )
    }
  }

  return (
    <div className="chart">
      <div className="chart-scale">
        <span className="chart-max">{max > 0 ? (formatMax ? formatMax(max) : String(max)) : '0'}</span>
        <span className="chart-min dim">0</span>
      </div>
      <svg
        className="chart-svg"
        viewBox={`0 0 ${VIEW_W} ${height}`}
        preserveAspectRatio="none"
        role="img"
        aria-label={label}
        style={{ height }}
      >
        {area && (
          <defs>
            <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity={0.35} />
              <stop offset="100%" stopColor={color} stopOpacity={0.02} />
            </linearGradient>
          </defs>
        )}
        {body}
        <line
          x1={0}
          y1={baselineY}
          x2={VIEW_W}
          y2={baselineY}
          stroke="var(--border-2)"
          strokeWidth={1}
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      {(startLabel || endLabel) && (
        <div className="chart-xlabels">
          <span className="dim">{startLabel}</span>
          <span className="dim">{endLabel}</span>
        </div>
      )}
    </div>
  )
}
