import { useMemo, useState } from "react"
import { GitBranch, Network } from "lucide-react"

import type { ChangedFileView } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  fileBasename,
  groupFilesByArea,
  type ChangeAreaGroup,
} from "@/lib/change-areas"
import { cn } from "@/lib/utils"

const BRANCH_COLORS = [
  "oklch(0.65 0.18 25)", // coral
  "oklch(0.62 0.16 300)", // magenta
  "oklch(0.62 0.14 250)", // indigo
  "oklch(0.68 0.15 55)", // amber
  "oklch(0.62 0.14 160)", // teal
  "oklch(0.60 0.16 340)", // rose
  "oklch(0.58 0.12 220)", // blue
  "oklch(0.65 0.14 130)", // green
]

const MAX_LEAVES_PER_BRANCH = 6
const VIEW = 640
const CX = VIEW / 2
const CY = VIEW / 2
const BRANCH_R = 168
const LEAF_R = 268

type LeafNode = {
  file: ChangedFileView
  x: number
  y: number
  label: string
  overflow?: number
}

type BranchLayout = {
  key: string
  label: string
  color: string
  bx: number
  by: number
  leaves: LeafNode[]
  totalFiles: number
}

function polar(cx: number, cy: number, r: number, angle: number) {
  return {
    x: cx + r * Math.cos(angle),
    y: cy + r * Math.sin(angle),
  }
}

function buildLayout(
  groups: ChangeAreaGroup<ChangedFileView>[],
): BranchLayout[] {
  const n = groups.length
  if (n === 0) return []

  const start = -Math.PI / 2
  return groups.map((g, i) => {
    const angle = start + (i * 2 * Math.PI) / n
    const { x: bx, y: by } = polar(CX, CY, BRANCH_R, angle)
    const color = BRANCH_COLORS[i % BRANCH_COLORS.length]
    const visible = g.files.slice(0, MAX_LEAVES_PER_BRANCH)
    const overflow = g.files.length - visible.length
    const leafCount = visible.length + (overflow > 0 ? 1 : 0)
    const fan =
      leafCount <= 1
        ? [0]
        : Array.from({ length: leafCount }, (_, j) => {
            const span = Math.min(1.1, 0.22 * leafCount)
            return -span / 2 + (j * span) / Math.max(1, leafCount - 1)
          })

    const leaves: LeafNode[] = visible.map((file, j) => {
      const a = angle + fan[j]!
      const { x, y } = polar(CX, CY, LEAF_R, a)
      return { file, x, y, label: fileBasename(file.path) }
    })

    if (overflow > 0) {
      const a = angle + fan[leafCount - 1]!
      const { x, y } = polar(CX, CY, LEAF_R, a)
      leaves.push({
        file: g.files[MAX_LEAVES_PER_BRANCH]!,
        x,
        y,
        label: `+${overflow}`,
        overflow,
      })
    }

    return {
      key: g.key,
      label: g.label,
      color,
      bx,
      by,
      leaves,
      totalFiles: g.files.length,
    }
  })
}

function curvePath(x1: number, y1: number, x2: number, y2: number) {
  const mx = (x1 + x2) / 2
  const my = (y1 + y2) / 2
  // Soft quadratic control offset for organic branches
  const dx = x2 - x1
  const dy = y2 - y1
  const cx = mx - dy * 0.12
  const cy = my + dx * 0.12
  return `M ${x1} ${y1} Q ${cx} ${cy} ${x2} ${y2}`
}

function truncate(s: string, max: number) {
  if (s.length <= max) return s
  return s.slice(0, max - 1) + "…"
}

export function ChangeMindMapCard({
  files,
  centerLabel,
  onSelect,
  className,
}: {
  files: ChangedFileView[]
  centerLabel: string
  onSelect: (f: ChangedFileView) => void
  className?: string
}) {
  const [hover, setHover] = useState<ChangedFileView | null>(null)

  const groups = useMemo(() => groupFilesByArea(files), [files])
  const branches = useMemo(() => buildLayout(groups), [groups])

  const centerText = truncate(centerLabel || "working tree", 18)

  return (
    <Card className={cn("flex min-h-0 flex-col overflow-hidden", className)}>
      <CardHeader className="shrink-0">
        <CardTitle className="flex items-center gap-2 text-sm">
          <Network className="size-4 text-muted-foreground" />
          Mapa de alterações
          {groups.length > 0 ? (
            <Badge variant="outline" className="font-mono text-[10px]">
              {groups.length} área{groups.length === 1 ? "" : "s"}
            </Badge>
          ) : null}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col gap-2 overflow-hidden">
        {files.length === 0 ? (
          <p className="flex flex-1 items-center justify-center px-1 text-center text-sm text-muted-foreground">
            Sem alterações para mapear.
          </p>
        ) : (
          <>
            <div className="relative min-h-0 flex-1 overflow-hidden rounded-lg border border-border/50 bg-muted/15">
              <svg
                viewBox={`0 0 ${VIEW} ${VIEW}`}
                className="h-full w-full"
                role="img"
                aria-label="Mapa mental das áreas e arquivos alterados"
              >
                <defs>
                  <filter id="mm-soft" x="-20%" y="-20%" width="140%" height="140%">
                    <feDropShadow
                      dx="0"
                      dy="1"
                      stdDeviation="1.5"
                      floodOpacity="0.18"
                    />
                  </filter>
                </defs>

                {/* Edges: center → branch → leaf */}
                {branches.map((b) => (
                  <g key={`edges-${b.key}`}>
                    <path
                      d={curvePath(CX, CY, b.bx, b.by)}
                      fill="none"
                      stroke={b.color}
                      strokeWidth={3.5}
                      strokeLinecap="round"
                      opacity={0.85}
                    />
                    {b.leaves.map((leaf, i) => (
                      <path
                        key={`e-${b.key}-${i}`}
                        d={curvePath(b.bx, b.by, leaf.x, leaf.y)}
                        fill="none"
                        stroke={b.color}
                        strokeWidth={1.6}
                        strokeLinecap="round"
                        opacity={0.55}
                      />
                    ))}
                  </g>
                ))}

                {/* Branch hubs */}
                {branches.map((b) => (
                  <g key={`hub-${b.key}`}>
                    <circle
                      cx={b.bx}
                      cy={b.by}
                      r={22}
                      fill={b.color}
                      opacity={0.18}
                    />
                    <circle
                      cx={b.bx}
                      cy={b.by}
                      r={7}
                      fill={b.color}
                      filter="url(#mm-soft)"
                    />
                    <text
                      x={b.bx}
                      y={b.by - 16}
                      textAnchor="middle"
                      className="fill-foreground"
                      style={{ fontSize: 11, fontWeight: 600 }}
                    >
                      {truncate(b.label, 16)}
                    </text>
                    <text
                      x={b.bx}
                      y={b.by + 22}
                      textAnchor="middle"
                      className="fill-muted-foreground"
                      style={{ fontSize: 9 }}
                    >
                      {b.totalFiles} arq.
                    </text>
                  </g>
                ))}

                {/* Leaves */}
                {branches.map((b) =>
                  b.leaves.map((leaf, i) => {
                    const isOverflow = Boolean(leaf.overflow)
                    const active = hover?.path === leaf.file.path
                    return (
                      <g
                        key={`leaf-${b.key}-${i}`}
                        className="cursor-pointer"
                        onMouseEnter={() => setHover(leaf.file)}
                        onMouseLeave={() => setHover(null)}
                        onClick={() => onSelect(leaf.file)}
                      >
                        <circle
                          cx={leaf.x}
                          cy={leaf.y}
                          r={isOverflow ? 14 : 11}
                          fill={
                            isOverflow
                              ? "var(--muted)"
                              : "var(--card)"
                          }
                          stroke={b.color}
                          strokeWidth={active ? 2.4 : 1.5}
                          filter="url(#mm-soft)"
                        />
                        <text
                          x={leaf.x}
                          y={leaf.y + (isOverflow ? 26 : 24)}
                          textAnchor="middle"
                          className="fill-foreground"
                          style={{
                            fontSize: 10,
                            fontFamily: "ui-monospace, monospace",
                            fontWeight: active ? 600 : 400,
                          }}
                        >
                          {truncate(leaf.label, 14)}
                        </text>
                      </g>
                    )
                  }),
                )}

                {/* Center */}
                <g>
                  <circle
                    cx={CX}
                    cy={CY}
                    r={42}
                    className="fill-card stroke-border"
                    strokeWidth={1.5}
                    filter="url(#mm-soft)"
                  />
                  <circle
                    cx={CX}
                    cy={CY}
                    r={42}
                    fill="var(--muted)"
                    opacity={0.35}
                  />
                  <foreignObject
                    x={CX - 36}
                    y={CY - 18}
                    width={72}
                    height={36}
                  >
                    <div className="flex h-full flex-col items-center justify-center gap-0.5 text-center">
                      <GitBranch className="size-3.5 text-muted-foreground" />
                      <span className="max-w-full truncate px-0.5 font-mono text-[10px] font-medium leading-tight text-foreground">
                        {centerText}
                      </span>
                    </div>
                  </foreignObject>
                </g>
              </svg>
            </div>

            <div className="shrink-0 truncate font-mono text-[11px] text-muted-foreground">
              {hover ? (
                <span title={hover.path}>
                  {hover.path}
                  {(hover.insertions > 0 || hover.deletions > 0) && (
                    <>
                      {" · "}
                      <span className="text-emerald-500">+{hover.insertions}</span>
                      {" "}
                      <span className="text-rose-400">−{hover.deletions}</span>
                    </>
                  )}
                </span>
              ) : (
                <span>Passe o mouse ou clique em um arquivo para abrir o diff</span>
              )}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
