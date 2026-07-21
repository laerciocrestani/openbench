import { useEffect, useMemo, useRef, useState } from "react"

import { AppService } from "../bindings/github.com/laerciocrestani/openbench"
import type { UpdateCheckResult } from "../bindings/github.com/laerciocrestani/openbench"
import type {
  BranchView,
  ChangedFileView,
  CommitContextIndex,
  CommitPreview,
  Dashboard,
  FileDiffView,
  OnboardingStatus,
  Prefs,
  ProjectStatus,
  PRPreview,
  SyncModeView,
  SyncResult,
} from "../bindings/github.com/laerciocrestani/openbench/internal/desktop"
import { Events, Window } from "@wailsio/runtime"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Switch } from "@/components/ui/switch"
import { Separator } from "@/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useTheme } from "@/components/theme-provider"
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { DockerEnvironmentSheet } from "@/components/docker-environment-sheet"
import { DockerGlobalPanel } from "@/components/docker-global-panel"
import { ProjectChatPanel } from "@/components/project-chat-panel"
import {
  TerminalPanel,
  type TerminalSessionSpec,
} from "@/components/terminal-panel"
import { TerminalChatSplit } from "@/components/terminal-chat-split"
import { SidebarWidthRail, useSidebarWidth } from "@/components/sidebar-width-resize"
import { UsageChartPanel } from "@/components/usage-chart"
import {
  BRANCH_TEMPLATES,
  isValidBranchName,
  templateNameSeed,
  type BranchTemplate,
} from "@/lib/branch-templates"
import {
  ArrowDownUp,
  ChartColumn,
  ChevronDown,
  ChevronLeft,
  CircleHelp,
  Container,
  Download,
  ExternalLink,
  FileText,
  FolderOpen,
  GitBranch,
  GitCommit,
  GitPullRequest,
  Loader2,
  Pin,
  PinOff,
  Play,
  Plus,
  RefreshCw,
  Settings,
  Square,
  X,
} from "lucide-react"

/* ------------------------------------------------------------------ */
/* Helpers                                                             */
/* ------------------------------------------------------------------ */

function errText(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === "string") return e
  try {
    return JSON.stringify(e)
  } catch {
    return String(e)
  }
}

type DiffRow =
  | { kind: "hunk"; text: string }
  | { kind: "ctx" | "add" | "del"; oldNo: number | null; newNo: number | null; text: string }

type CommitAction = "commit" | "push" | "pr" | "branch-commit" | "branch-commit-push"

type CreateBranchStep = "from" | "template" | "name"

function isOnBase(dash: Dashboard | null): boolean {
  if (!dash || dash.detached) return false
  const base = (dash.baseBranch || "").trim()
  const branch = (dash.branch || "").trim()
  if (!base || !branch) return false
  return branch === base
}

function parseUnifiedDiff(unified: string): DiffRow[] {
  const rows: DiffRow[] = []
  if (!unified) return rows

  const lines = unified.split("\n")
  let oldNo = 0
  let newNo = 0

  for (const raw of lines) {
    if (
      raw.startsWith("diff --git") ||
      raw.startsWith("index ") ||
      raw.startsWith("--- ") ||
      raw.startsWith("+++ ") ||
      raw.startsWith("new file") ||
      raw.startsWith("deleted file") ||
      raw.startsWith("old mode") ||
      raw.startsWith("new mode") ||
      raw.startsWith("similarity ") ||
      raw.startsWith("rename ") ||
      raw.startsWith("copy ")
    ) {
      continue
    }

    if (raw.startsWith("@@")) {
      const m = /@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(raw)
      if (m) {
        oldNo = parseInt(m[1], 10)
        newNo = parseInt(m[2], 10)
      }
      rows.push({ kind: "hunk", text: raw })
      continue
    }

    if (raw.startsWith("\\")) {
      // "\ No newline at end of file"
      continue
    }

    if (raw.startsWith("+")) {
      rows.push({ kind: "add", oldNo: null, newNo, text: raw.slice(1) })
      newNo++
    } else if (raw.startsWith("-")) {
      rows.push({ kind: "del", oldNo, newNo: null, text: raw.slice(1) })
      oldNo++
    } else {
      const text = raw.startsWith(" ") ? raw.slice(1) : raw
      rows.push({ kind: "ctx", oldNo, newNo, text })
      oldNo++
      newNo++
    }
  }

  return rows
}

/* ------------------------------------------------------------------ */
/* Presentational sub-components                                       */
/* ------------------------------------------------------------------ */

function StatusBadge({ label, dirty }: { label: string; dirty: boolean }) {
  return (
    <Badge variant={dirty ? "secondary" : "outline"} className="font-normal">
      {label || (dirty ? "dirty" : "clean")}
    </Badge>
  )
}

function FileStatusBadge({ status }: { status: string }) {
  const s = (status || "").trim().toLowerCase()
  let letter = "·"
  let variant: "default" | "secondary" | "destructive" | "outline" = "outline"

  switch (s) {
    case "untracked":
      letter = "?"
      variant = "outline"
      break
    case "deleted":
      letter = "D"
      variant = "destructive"
      break
    case "new":
    case "staged":
      letter = "A"
      variant = "default"
      break
    case "modified":
    case "staged+modified":
    case "changed":
      letter = "M"
      variant = "secondary"
      break
    case "renamed":
      letter = "R"
      variant = "outline"
      break
    default:
      letter = s ? s.charAt(0).toUpperCase() : "·"
  }

  return (
    <Badge variant={variant} className="w-6 justify-center font-mono">
      {letter}
    </Badge>
  )
}

function contextLevelStyles(level: string): {
  bar: string
  badge: "default" | "destructive" | "outline"
  text: string
} {
  switch (level) {
    case "critical":
      return {
        bar: "bg-destructive",
        badge: "destructive",
        text: "text-destructive",
      }
    case "attention":
      return {
        bar: "bg-amber-500",
        badge: "outline",
        text: "text-amber-600 dark:text-amber-400",
      }
    default:
      return {
        bar: "bg-emerald-500",
        badge: "outline",
        text: "text-muted-foreground",
      }
  }
}

function formatContextBytes(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return "0 B"
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function ContextIndexPanel({
  index,
  onRecommendCommit,
  busy,
}: {
  index: CommitContextIndex
  onRecommendCommit: () => void
  busy: boolean
}) {
  const styles = contextLevelStyles(index.level)
  const score = Math.max(0, Math.min(100, index.score ?? 0))
  const estimated = formatContextBytes(index.estimatedBytes ?? 0)
  const maxDiff = formatContextBytes(index.maxDiffBytes ?? 0)
  const modelWindow = index.modelContextWindow?.trim() || ""
  const model = index.model?.trim() || ""

  return (
    <div className="flex flex-col gap-2 rounded-lg border bg-muted/20 px-3 py-2.5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <button
                  type="button"
                  className="inline-flex items-center gap-1.5 text-xs font-medium text-foreground hover:text-foreground/90"
                />
              }
            >
              Índice de contexto
              <CircleHelp className="size-3.5 text-muted-foreground" aria-hidden />
            </TooltipTrigger>
            <TooltipContent
              side="bottom"
              align="start"
              className="flex max-w-xs flex-col items-stretch gap-2 px-3 py-2.5 text-left leading-snug"
            >
              <p className="font-medium">O que mede</p>
              <p className="opacity-90">
                O peso do diff que a IA verá no próximo commit.
              </p>

              <p className="font-medium">Como calcula</p>
              <p className="opacity-90">
                Linhas (+/−), arquivos, áreas distintas e proximidade do truncamento
                (`max_diff_bytes`).
              </p>

              <p className="font-medium">Estimado agora</p>
              <p className="font-mono opacity-90">{estimated}</p>

              <p className="font-medium">Limite enviado</p>
              <p className="font-mono opacity-90">{maxDiff}</p>

              {model ? (
                <>
                  <p className="font-medium">Modelo</p>
                  <p className="font-mono opacity-90">{model}</p>
                </>
              ) : null}

              {modelWindow ? (
                <>
                  <p className="font-medium">Janela do modelo</p>
                  <p className="font-mono opacity-90">~{modelWindow}</p>
                </>
              ) : null}

              <p className="font-medium">Dica</p>
              <p className="opacity-90">
                Janela maior aguenta mais escopo; o openbench ainda respeita
                `max_diff_bytes`. Em atenção/crítico, prefira commits menores.
              </p>
            </TooltipContent>
          </Tooltip>
          <Badge variant={styles.badge} className="font-mono text-[10px]">
            {score}%
          </Badge>
        </div>
        <span className={`text-xs ${styles.text}`}>{index.label}</span>
      </div>

      <div
        className="h-2 overflow-hidden rounded-full bg-muted"
        role="meter"
        aria-valuenow={score}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label="Índice de contexto do próximo commit"
      >
        <div
          className={`h-full rounded-full transition-[width] duration-300 ${styles.bar}`}
          style={{ width: `${score}%` }}
        />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span className="font-mono">
          +{index.insertions} −{index.deletions} · {index.fileCount} arquivo
          {index.fileCount === 1 ? "" : "s"}
          {index.areaCount > 1 ? ` · ${index.areaCount} áreas` : ""}
          {index.nearTruncate ? " · perto do limite da IA" : ""}
        </span>
        {index.recommendCommit && (
          <Button
            size="sm"
            variant={index.level === "critical" ? "default" : "outline"}
            className="h-7 text-xs"
            disabled={busy}
            onClick={onRecommendCommit}
          >
            <GitCommit className="size-3.5" />
            Recomenda-se commit
          </Button>
        )}
      </div>
    </div>
  )
}

function ChangedFilesTable({
  files,
  onSelect,
}: {
  files: ChangedFileView[]
  onSelect: (f: ChangedFileView) => void
}) {
  if (files.length === 0) {
    return (
      <p className="px-1 py-6 text-center text-sm text-muted-foreground">
        Nenhuma alteração na árvore de trabalho.
      </p>
    )
  }

  return (
    <Table>
      <TableHeader className="sticky top-0 z-10 bg-card">
        <TableRow>
          <TableHead className="w-10">St</TableHead>
          <TableHead>Arquivo</TableHead>
          <TableHead className="w-20 text-right">+</TableHead>
          <TableHead className="w-20 text-right">−</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {files.map((f) => (
          <TableRow
            key={f.path}
            className="cursor-pointer"
            onClick={() => onSelect(f)}
          >
            <TableCell>
              <FileStatusBadge status={f.status} />
            </TableCell>
            <TableCell className="font-mono text-xs">{f.path}</TableCell>
            <TableCell className="text-right font-mono text-xs text-emerald-500">
              {f.insertions > 0 ? `+${f.insertions}` : "0"}
            </TableCell>
            <TableCell className="text-right font-mono text-xs text-destructive">
              {f.deletions > 0 ? `−${f.deletions}` : "0"}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function DiffViewer({ diff }: { diff: FileDiffView }) {
  const rows = useMemo(() => parseUnifiedDiff(diff.unified), [diff.unified])

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border">
      <div className="flex shrink-0 items-center justify-between gap-3 border-b bg-muted/40 px-3 py-2">
        <span className="truncate font-mono text-xs">{diff.path}</span>
        <div className="flex shrink-0 items-center gap-2">
          <Badge variant="outline" className="font-mono text-emerald-500">
            +{diff.insertions}
          </Badge>
          <Badge variant="outline" className="font-mono text-destructive">
            −{diff.deletions}
          </Badge>
        </div>
      </div>

      {diff.binary ? (
        <div className="flex flex-1 items-center justify-center p-8 text-sm text-muted-foreground">
          Arquivo binário — diff não disponível.
        </div>
      ) : rows.length === 0 ? (
        <div className="flex flex-1 items-center justify-center p-8 text-sm text-muted-foreground">
          Sem diferenças para exibir.
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-auto">
          <div className="w-max min-w-full font-mono text-xs leading-5">
            {rows.map((row, i) => {
              if (row.kind === "hunk") {
                return (
                  <div
                    key={i}
                    className="bg-muted/60 px-3 py-1 text-muted-foreground select-none"
                  >
                    {row.text}
                  </div>
                )
              }

              const bg =
                row.kind === "add"
                  ? "bg-emerald-500/10"
                  : row.kind === "del"
                    ? "bg-destructive/10"
                    : ""
              const sign = row.kind === "add" ? "+" : row.kind === "del" ? "−" : " "
              const signColor =
                row.kind === "add"
                  ? "text-emerald-500"
                  : row.kind === "del"
                    ? "text-destructive"
                    : "text-muted-foreground"

              return (
                <div key={i} className={`flex ${bg}`}>
                  <span className="w-12 shrink-0 px-2 text-right text-muted-foreground/60 tabular-nums select-none">
                    {row.oldNo ?? ""}
                  </span>
                  <span className="w-12 shrink-0 px-2 text-right text-muted-foreground/60 tabular-nums select-none">
                    {row.newNo ?? ""}
                  </span>
                  <span className={`w-4 shrink-0 text-center select-none ${signColor}`}>
                    {sign}
                  </span>
                  <span className="whitespace-pre px-2">{row.text}</span>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}

function ProjectTabs({
  statuses,
  activePath,
  onSwitch,
  onUnpin,
}: {
  statuses: ProjectStatus[]
  activePath: string | null
  onSwitch: (path: string) => void
  onUnpin: (path: string) => void
}) {
  if (statuses.length === 0) return null

  return (
    <div className="flex items-center gap-1.5 overflow-x-auto pb-0.5">
      {statuses.map((s) => {
        const active = s.active || s.path === activePath
        return (
          <div
            key={s.path}
            className={`group flex items-center gap-1 rounded-lg border px-2 py-1 text-xs transition-colors ${
              active
                ? "border-primary/40 bg-primary/10 text-foreground"
                : "border-border bg-background text-muted-foreground hover:bg-muted"
            }`}
          >
            <button
              type="button"
              className="flex items-center gap-1.5 outline-none"
              onClick={() => onSwitch(s.path)}
              title={s.path}
            >
              <span className="font-medium">{s.alias || s.repoName}</span>
              {s.dirty && <span className="size-1.5 rounded-full bg-amber-500" />}
              {s.hasOpenPR && <GitPullRequest className="size-3 text-sky-500" />}
            </button>
            <Button
              variant="ghost"
              size="icon-xs"
              className="opacity-40 group-hover:opacity-100"
              onClick={() => onUnpin(s.path)}
              title="Desafixar projeto"
            >
              <PinOff />
            </Button>
          </div>
        )
      })}
    </div>
  )
}

function DashboardView({
  dash,
  busy,
  onSelectFile,
  onOpenBranches,
  onRecommendCommit,
}: {
  dash: Dashboard
  busy: boolean
  onSelectFile: (f: ChangedFileView) => void
  onOpenBranches: () => void
  onRecommendCommit: () => void
}) {
  const files = dash.changedFiles ?? []
  const contextIndex = dash.contextIndex

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="grid shrink-0 grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <button
          type="button"
          onClick={onOpenBranches}
          className="rounded-xl text-left transition-colors hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <Card size="sm" className="pointer-events-none h-full">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-sm">
                <GitBranch className="size-4 text-muted-foreground" />
                Branch
                <span className="ml-auto text-xs font-normal text-muted-foreground">
                  trocar…
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-1.5">
              <span className="font-mono text-sm">
                {dash.branch}
                {dash.detached && " (detached)"}
              </span>
              <div className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                <span>base: {dash.baseBranch || "—"}</span>
                {dash.ahead > 0 && <Badge variant="outline">↑{dash.ahead}</Badge>}
                {dash.behind > 0 && <Badge variant="outline">↓{dash.behind}</Badge>}
                {dash.commitsAheadOfBase > 0 && (
                  <Badge variant="outline">{dash.commitsAheadOfBase} vs base</Badge>
                )}
              </div>
            </CardContent>
          </Card>
        </button>

        <Card size="sm">
          <CardHeader>
            <CardTitle className="text-sm">Status</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-2">
            <StatusBadge label={dash.statusLabel} dirty={dash.dirty} />
            <div className="flex flex-wrap gap-1.5 text-xs text-muted-foreground">
              <Badge variant="outline">staged {dash.staged}</Badge>
              <Badge variant="outline">mod {dash.modified}</Badge>
              <Badge variant="outline">untracked {dash.untracked}</Badge>
            </div>
          </CardContent>
        </Card>

        <Card size="sm">
          <CardHeader>
            <CardTitle className="text-sm">IA</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-1.5 text-xs text-muted-foreground">
            <div className="flex items-center gap-2">
              <Badge variant={dash.aiReady ? "default" : "destructive"}>
                {dash.aiReady ? "pronto" : "config necessária"}
              </Badge>
            </div>
            <span>
              {dash.provider || "—"} · {dash.model || "—"}
            </span>
            {dash.openPR && (
              <a
                href={dash.openPR.url}
                target="_blank"
                rel="noreferrer"
                className="flex items-center gap-1 text-sky-500 hover:underline"
              >
                <ExternalLink className="size-3" />
                PR #{dash.openPR.number}
              </a>
            )}
          </CardContent>
        </Card>
      </div>

      <Card className="flex min-h-0 flex-1 flex-col overflow-hidden">
        <CardHeader className="shrink-0 gap-3">
          <CardTitle className="flex items-center gap-2 text-sm">
            <FileText className="size-4 text-muted-foreground" />
            Arquivos alterados ({files.length})
          </CardTitle>
          {contextIndex && (
            <ContextIndexPanel
              index={contextIndex}
              busy={busy}
              onRecommendCommit={onRecommendCommit}
            />
          )}
        </CardHeader>
        <CardContent className="min-h-0 flex-1 overflow-y-auto">
          <ChangedFilesTable files={files} onSelect={onSelectFile} />
        </CardContent>
      </Card>
    </div>
  )
}

function projectDisplayName(path: string): string {
  const trimmed = path.replace(/[/\\]+$/, "")
  const parts = trimmed.split(/[/\\]/)
  return parts[parts.length - 1] || path
}

function Welcome({
  recent,
  pinned,
  busy,
  onOpenDialog,
  onOpenPath,
  onPin,
  onUnpin,
}: {
  recent: string[]
  pinned: { path: string; alias?: string }[]
  busy: boolean
  onOpenDialog: () => void
  onOpenPath: (path: string) => void
  onPin: (path: string) => void
  onUnpin: (path: string) => void
}) {
  const pinnedPaths = new Set(pinned.map((p) => p.path))
  const recentOnly = recent.filter((p) => !pinnedPaths.has(p)).slice(0, 8)

  return (
    <div className="grid h-full min-h-0 grid-cols-1 gap-4 lg:grid-cols-2">
      <div className="flex min-h-0 flex-col gap-4 overflow-y-auto rounded-xl border bg-card p-5">
        <div className="flex flex-col gap-2">
          <div className="flex size-10 items-center justify-center rounded-xl bg-primary/10">
            <FolderOpen className="size-5 text-primary" />
          </div>
          <h1 className="font-heading text-lg font-medium">openbench</h1>
          <p className="text-sm text-muted-foreground">
            Abra um repositório Git para ver o dashboard.
          </p>
        </div>

        <Button size="lg" className="w-full sm:w-auto" onClick={onOpenDialog} disabled={busy}>
          {busy ? <Loader2 className="animate-spin" /> : <FolderOpen />}
          Abrir projeto…
        </Button>

        {pinned.length > 0 && (
          <div className="flex flex-col gap-2">
            <h2 className="text-sm font-medium text-muted-foreground">Fixados</h2>
            <div className="grid grid-cols-1 gap-2">
              {pinned.map((p) => (
                <div key={p.path} className="group relative">
                  <button
                    type="button"
                    className="flex w-full items-center gap-2 rounded-xl border bg-background px-4 py-3 text-left transition-colors hover:bg-muted/50"
                    onClick={() => onOpenPath(p.path)}
                    title={p.path}
                    disabled={busy}
                  >
                    <Pin className="size-3.5 shrink-0 text-primary" />
                    <span className="truncate font-medium">
                      {p.alias || projectDisplayName(p.path)}
                    </span>
                  </button>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    className="absolute top-2 right-2 opacity-0 group-hover:opacity-100"
                    onClick={(e) => {
                      e.stopPropagation()
                      onUnpin(p.path)
                    }}
                    title="Desafixar"
                    disabled={busy}
                  >
                    <PinOff />
                  </Button>
                </div>
              ))}
            </div>
          </div>
        )}

        {recentOnly.length > 0 && (
          <div className="flex min-h-0 flex-1 flex-col gap-2">
            <h2 className="text-sm font-medium text-muted-foreground">Recentes</h2>
            <div className="flex flex-col overflow-hidden rounded-xl border">
              {recentOnly.map((p) => (
                <div
                  key={p}
                  className="group flex items-center gap-1 border-b px-2 last:border-0"
                >
                  <button
                    type="button"
                    className="flex min-w-0 flex-1 items-center gap-2 px-2 py-2.5 text-left text-xs hover:bg-muted"
                    onClick={() => onOpenPath(p)}
                    disabled={busy}
                  >
                    <FolderOpen className="size-3.5 shrink-0 text-muted-foreground" />
                    <span className="truncate font-mono" title={p}>
                      {p}
                    </span>
                  </button>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    className="shrink-0 opacity-50 group-hover:opacity-100"
                    onClick={() => onPin(p)}
                    title="Fixar projeto"
                    disabled={busy}
                  >
                    <Pin />
                  </Button>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      <div className="min-h-[22rem] lg:min-h-0">
        <DockerGlobalPanel active />
      </div>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Main App                                                            */
/* ------------------------------------------------------------------ */

function App() {
  const { theme, setTheme } = useTheme()
  const [version, setVersion] = useState("")
  const [prefsPath, setPrefsPath] = useState("")

  const [prefs, setPrefs] = useState<Prefs | null>(null)
  const [dash, setDash] = useState<Dashboard | null>(null)
  const [statuses, setStatuses] = useState<ProjectStatus[]>([])

  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Diff sheet
  const [fileDiff, setFileDiff] = useState<FileDiffView | null>(null)
  const [diffOpen, setDiffOpen] = useState(false)

  // Branches sheet
  const [branchesOpen, setBranchesOpen] = useState(false)
  const [branches, setBranches] = useState<BranchView[]>([])
  const [branchesLoading, setBranchesLoading] = useState(false)
  const [branchFilter, setBranchFilter] = useState("")
  const [checkoutBusy, setCheckoutBusy] = useState(false)
  const [checkoutConfirm, setCheckoutConfirm] = useState<string | null>(null)
  const [dockerLoading, setDockerLoading] = useState(false)

  // Create branch wizard (inside branches sheet)
  const [createBranchStep, setCreateBranchStep] = useState<CreateBranchStep | null>(null)
  const [createBranchFrom, setCreateBranchFrom] = useState("")
  const [createBranchTemplate, setCreateBranchTemplate] = useState<BranchTemplate | null>(null)
  const [createBranchName, setCreateBranchName] = useState("")
  const [createBranchBusy, setCreateBranchBusy] = useState(false)

  // Modals
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [usageOpen, setUsageOpen] = useState(false)
  const [onboardingOpen, setOnboardingOpen] = useState(false)
  const [commitOpen, setCommitOpen] = useState(false)
  const [prOpen, setPrOpen] = useState(false)
  const [recreateOpen, setRecreateOpen] = useState(false)
  const [recreateService, setRecreateService] = useState("")
  const [dockerEnvOpen, setDockerEnvOpen] = useState(false)
  const [termSession, setTermSession] = useState<TerminalSessionSpec>({ kind: "host" })

  // Sync dialog
  const [syncOpen, setSyncOpen] = useState(false)
  const [syncModes, setSyncModes] = useState<SyncModeView[]>([])
  const [syncMode, setSyncMode] = useState("standard")
  const [syncBusy, setSyncBusy] = useState(false)
  const [syncResult, setSyncResult] = useState<SyncResult | null>(null)

  // Commit modal
  const [commitPreview, setCommitPreview] = useState<CommitPreview | null>(null)
  const [commitMessage, setCommitMessage] = useState("")
  const [commitBusy, setCommitBusy] = useState(false)
  const [commitAction, setCommitAction] = useState<CommitAction>("commit")

  // New branch (before commit on base)
  const [newBranchOpen, setNewBranchOpen] = useState(false)
  const [newBranchName, setNewBranchName] = useState("")
  const [newBranchFrom, setNewBranchFrom] = useState("main")
  const [newBranchBusy, setNewBranchBusy] = useState(false)

  // PR modal
  const [prPreview, setPrPreview] = useState<PRPreview | null>(null)
  const [prTitle, setPrTitle] = useState("")
  const [prBody, setPrBody] = useState("")
  const [prDraft, setPrDraft] = useState(false)
  const [prBusy, setPrBusy] = useState(false)

  // Onboarding modal
  const [onboarding, setOnboarding] = useState<OnboardingStatus | null>(null)
  const [obProvider, setObProvider] = useState("openai")
  const [obApiKey, setObApiKey] = useState("")
  const [obModel, setObModel] = useState("")
  const [obBusy, setObBusy] = useState(false)

  // Settings — update check
  const [updateResult, setUpdateResult] = useState<UpdateCheckResult | null>(null)
  const [updateBusy, setUpdateBusy] = useState(false)
  const [aliasDrafts, setAliasDrafts] = useState<Record<string, string>>({})

  // Settings — IA tab
  const [aiProvider, setAiProvider] = useState("openrouter")
  const [aiApiKey, setAiApiKey] = useState("")
  const [aiKeyMasked, setAiKeyMasked] = useState("")
  const [aiGitModel, setAiGitModel] = useState("")
  const [aiGitFallback, setAiGitFallback] = useState("")
  const [aiChatModel, setAiChatModel] = useState("")
  const [aiChatFallback, setAiChatFallback] = useState("")
  const [aiSuggestions, setAiSuggestions] = useState<string[]>([])
  const [aiConfigPath, setAiConfigPath] = useState("")
  const [aiBusy, setAiBusy] = useState(false)

  // Terminal sidebar — always available (home → user home dir; project → repo root)
  const [terminalOpen, setTerminalOpen] = useState(true)

  /* --------------------------- data loaders --------------------------- */

  const refreshStatuses = async () => {
    try {
      const st = await AppService.ListProjectStatuses()
      setStatuses(st ?? [])
    } catch (e) {
      setError(errText(e))
    }
  }

  const reloadPrefs = async () => {
    try {
      const p = await AppService.GetPrefs()
      setPrefs(p)
    } catch (e) {
      setError(errText(e))
    }
  }

  /* ----------------------------- actions ----------------------------- */

  const openDialog = async () => {
    setBusy(true)
    setError(null)
    try {
      const d = await AppService.OpenProjectDialog()
      if (d) {
        setDash(d)
        setTerminalOpen(true)
        setTermSession({ kind: "host" })
        await refreshStatuses()
        await reloadPrefs()
      }
    } catch (e) {
      setError(errText(e))
    } finally {
      setBusy(false)
    }
  }

  const openPath = async (path: string) => {
    setBusy(true)
    setError(null)
    try {
      const d = await AppService.OpenProject(path)
      if (d) {
        setDash(d)
        setTerminalOpen(true)
        setTermSession({ kind: "host" })
        await refreshStatuses()
        await reloadPrefs()
      }
    } catch (e) {
      setError(errText(e))
    } finally {
      setBusy(false)
    }
  }

  const switchProject = async (path: string) => {
    setBusy(true)
    setError(null)
    try {
      const d = await AppService.SwitchProject(path)
      if (d) {
        setDash(d)
        setTerminalOpen(true)
        setTermSession({ kind: "host" })
      }
      await refreshStatuses()
      await reloadPrefs()
    } catch (e) {
      setError(errText(e))
    } finally {
      setBusy(false)
    }
  }

  const unpinProject = async (path: string) => {
    try {
      const d = await AppService.UnpinProject(path)
      setDash(d ?? null)
      await refreshStatuses()
      await reloadPrefs()
    } catch (e) {
      setError(errText(e))
    }
  }

  const pinProject = async (path: string) => {
    try {
      await AppService.PinProject(path)
      await refreshStatuses()
      await reloadPrefs()
    } catch (e) {
      setError(errText(e))
    }
  }

  const refresh = async () => {
    if (!dash) return
    setBusy(true)
    setError(null)
    try {
      const d = await AppService.RefreshDashboard()
      if (d) setDash(d)
      await AppService.RefreshProjectStatuses()
      await refreshStatuses()
    } catch (e) {
      setError(errText(e))
    } finally {
      setBusy(false)
    }
  }

  const closeProject = async () => {
    try {
      await AppService.CloseProject()
      setDash(null)
      setTermSession({ kind: "host" })
      await refreshStatuses()
      await reloadPrefs()
    } catch (e) {
      setError(errText(e))
    }
  }

  const openFileDiff = async (f: ChangedFileView) => {
    setError(null)
    setDiffOpen(true)
    setFileDiff(null)
    try {
      const d = await AppService.FileDiff(f.path)
      if (d) setFileDiff(d)
    } catch (e) {
      setError(errText(e))
      setDiffOpen(false)
    }
  }

  const loadBranches = async () => {
    setBranchesLoading(true)
    setError(null)
    try {
      const list = await AppService.ListBranches()
      setBranches(list ?? [])
    } catch (e) {
      setError(errText(e))
      setBranches([])
    } finally {
      setBranchesLoading(false)
    }
  }

  const openBranches = async () => {
    setBranchesOpen(true)
    setBranchFilter("")
    setCheckoutConfirm(null)
    setCreateBranchStep(null)
    setCreateBranchTemplate(null)
    setCreateBranchName("")
    await loadBranches()
  }

  const startCreateBranch = () => {
    const current = branches.find((b) => b.current)?.name ?? dash?.branch ?? ""
    const preferred =
      branches.find((b) => b.name === (dash?.baseBranch || "").trim())?.name ||
      current ||
      branches[0]?.name ||
      dash?.baseBranch ||
      "main"
    setCreateBranchFrom(preferred)
    setCreateBranchTemplate(null)
    setCreateBranchName("")
    setCreateBranchStep("from")
  }

  const cancelCreateBranch = () => {
    setCreateBranchStep(null)
    setCreateBranchTemplate(null)
    setCreateBranchName("")
  }

  const selectCreateFrom = (name: string) => {
    setCreateBranchFrom(name)
    setCreateBranchStep("template")
  }

  const selectCreateTemplate = (tpl: BranchTemplate) => {
    setCreateBranchTemplate(tpl)
    setCreateBranchName(templateNameSeed(tpl))
    setCreateBranchStep("name")
  }

  const confirmCreateBranch = async () => {
    const name = createBranchName.trim()
    const from = createBranchFrom.trim()
    if (!name || !from || !isValidBranchName(name)) return
    setCreateBranchBusy(true)
    setError(null)
    try {
      const d = await AppService.CreateBranch(name, from)
      if (d) setDash(d)
      cancelCreateBranch()
      await loadBranches()
      await refreshStatuses()
    } catch (e) {
      setError(errText(e))
    } finally {
      setCreateBranchBusy(false)
    }
  }

  const requestCheckout = (name: string) => {
    if (!name || !dash) return
    const current = branches.find((b) => b.current)?.name ?? dash.branch
    if (name === current) return
    if (dash.dirty) {
      setCheckoutConfirm(name)
      return
    }
    void doCheckout(name)
  }

  const doCheckout = async (name: string) => {
    setCheckoutBusy(true)
    setError(null)
    setCheckoutConfirm(null)
    try {
      const d = await AppService.CheckoutBranch(name)
      if (d) setDash(d)
      await loadBranches()
      await refreshStatuses()
    } catch (e) {
      setError(errText(e))
    } finally {
      setCheckoutBusy(false)
    }
  }

  const openSync = async () => {
    setError(null)
    setSyncResult(null)
    setSyncMode("standard")
    try {
      const modes = await AppService.SyncModes()
      setSyncModes(modes ?? [])
      if (modes && modes.length > 0) setSyncMode(modes[0].id)
    } catch (e) {
      setError(errText(e))
      return
    }
    setSyncOpen(true)
  }

  const runSync = async () => {
    if (!dash) return
    if (dash.dirty) {
      setError("Working tree dirty — commit ou stash antes de sincronizar")
      return
    }
    setSyncBusy(true)
    setError(null)
    setSyncResult(null)
    try {
      const res = await AppService.RunSync(syncMode, dash.baseBranch || "main")
      if (res) {
        setSyncResult(res)
        if (res.dashboard) setDash(res.dashboard)
        await refreshStatuses()
      }
    } catch (e) {
      setError(errText(e))
    } finally {
      setSyncBusy(false)
    }
  }

  const filteredBranches = useMemo(() => {
    const q = branchFilter.trim().toLowerCase()
    if (!q) return branches
    return branches.filter((b) => b.name.toLowerCase().includes(q))
  }, [branches, branchFilter])

  const selectedSyncMode = useMemo(
    () => syncModes.find((m) => m.id === syncMode) ?? null,
    [syncModes, syncMode],
  )

  /* --------------------------- onboarding --------------------------- */

  const ensureOnboarding = async (): Promise<boolean> => {
    try {
      const st = await AppService.CheckOnboarding()
      if (st && st.needsOnboarding) {
        setOnboarding(st)
        setObProvider(st.provider || "openai")
        setObModel(st.model || "")
        setObApiKey("")
        setOnboardingOpen(true)
        return false
      }
      return true
    } catch (e) {
      setError(errText(e))
      return false
    }
  }

  const saveOnboarding = async () => {
    setObBusy(true)
    setError(null)
    try {
      await AppService.SaveAIConfig(obProvider, obApiKey, obModel)
      setOnboardingOpen(false)
      if (dash) await refresh()
    } catch (e) {
      setError(errText(e))
    } finally {
      setObBusy(false)
    }
  }

  /* ----------------------------- commit ----------------------------- */

  const openCommitPreview = async () => {
    setCommitOpen(true)
    setCommitPreview(null)
    setCommitMessage("")
    setCommitBusy(true)
    setError(null)
    try {
      const p = await AppService.PreviewCommit()
      if (p) {
        setCommitPreview(p)
        setCommitMessage(p.message)
      }
    } catch (e) {
      setError(errText(e))
      setCommitOpen(false)
    } finally {
      setCommitBusy(false)
    }
  }

  const startCommitAction = async (action: CommitAction) => {
    if (!dash) return
    if (!(await ensureOnboarding())) return
    setCommitAction(action)
    if (action === "branch-commit" || action === "branch-commit-push") {
      setNewBranchFrom(dash.baseBranch || "main")
      setNewBranchName("")
      setNewBranchOpen(true)
      return
    }
    await openCommitPreview()
  }

  const startCommit = async () => {
    await startCommitAction("commit")
  }

  const confirmNewBranch = async () => {
    const name = newBranchName.trim()
    if (!name) return
    setNewBranchBusy(true)
    setError(null)
    try {
      const d = await AppService.CreateBranch(name, newBranchFrom || "main")
      if (d) setDash(d)
      setNewBranchOpen(false)
      await openCommitPreview()
    } catch (e) {
      setError(errText(e))
    } finally {
      setNewBranchBusy(false)
    }
  }

  const confirmCommit = async () => {
    setCommitBusy(true)
    setError(null)
    try {
      if (commitAction === "push" || commitAction === "branch-commit-push") {
        await AppService.ConfirmCommitAndPush(commitMessage)
        setCommitOpen(false)
        await refresh()
      } else if (commitAction === "pr") {
        await AppService.ConfirmCommit(commitMessage)
        setCommitOpen(false)
        await refresh()
        await startPR()
      } else {
        await AppService.ConfirmCommit(commitMessage)
        setCommitOpen(false)
        await refresh()
      }
    } catch (e) {
      setError(errText(e))
    } finally {
      setCommitBusy(false)
    }
  }

  const commitConfirmLabel =
    commitAction === "push" || commitAction === "branch-commit-push"
      ? "Commit & Push"
      : commitAction === "pr"
        ? "Commit & Create PR"
        : "Confirmar commit"

  /* ------------------------------- PR ------------------------------- */

  const startPR = async () => {
    if (!dash) return
    if (!(await ensureOnboarding())) return
    setPrOpen(true)
    setPrPreview(null)
    setPrTitle("")
    setPrBody("")
    setPrBusy(true)
    setError(null)
    try {
      const p = await AppService.PreviewPR(prDraft)
      if (p) {
        setPrPreview(p)
        setPrTitle(p.title)
        setPrBody(p.body)
        setPrDraft(p.draft)
      }
    } catch (e) {
      setError(errText(e))
    } finally {
      setPrBusy(false)
    }
  }

  const confirmPR = async () => {
    setPrBusy(true)
    setError(null)
    try {
      const out = await AppService.ConfirmPR(prTitle, prBody, prDraft)
      setPrOpen(false)
      await refresh()
      if (out?.url) window.open(out.url, "_blank")
    } catch (e) {
      setError(errText(e))
    } finally {
      setPrBusy(false)
    }
  }

  const openRecreate = () => {
    const services = dash?.docker?.services ?? []
    const def = dash?.docker?.defaultService || services[0]?.name || ""
    setRecreateService(def)
    setRecreateOpen(true)
  }

  const dockerAction = async (fn: () => Promise<unknown>) => {
    setBusy(true)
    setError(null)
    try {
      const res = (await fn()) as { dashboard?: Dashboard | null } | null
      if (res?.dashboard) setDash(res.dashboard)
      else await refresh()
    } catch (e) {
      setError(errText(e))
    } finally {
      setBusy(false)
    }
  }

  const confirmRecreate = async () => {
    const svc = recreateService.trim()
    if (!svc) {
      setError("Selecione um serviço para recreate.")
      return
    }
    setRecreateOpen(false)
    await dockerAction(() => AppService.DockerRecreate(svc))
  }

  const openDockerShell = (service: string, presetId?: string) => {
    setTermSession({
      kind: "docker",
      service,
      presetId: presetId?.trim() || undefined,
    })
    setTerminalOpen(true)
  }

  /* ---------------------------- settings ---------------------------- */

  const openSettings = async () => {
    setSettingsOpen(true)
    setUpdateResult(null)
    setAiApiKey("")
    await reloadPrefs()
    try {
      const pp = await AppService.PrefsPathString()
      setPrefsPath(pp)
    } catch {
      /* ignore */
    }
    try {
      const ai = await AppService.GetAIConfig()
      if (ai) {
        setAiProvider(ai.provider || "openrouter")
        setAiKeyMasked(ai.apiKeyMasked || "")
        setAiGitModel(ai.gitModel || "")
        setAiGitFallback(ai.gitFallback || "")
        setAiChatModel(ai.chatModel || "")
        setAiChatFallback(ai.chatFallback || "")
        setAiSuggestions(ai.modelSuggestions ?? [])
        setAiConfigPath(ai.configPath || "")
      }
    } catch (e) {
      setError(errText(e))
    }
  }

  const saveAISettings = async () => {
    setAiBusy(true)
    setError(null)
    try {
      await AppService.SaveAISettings(
        aiProvider,
        aiApiKey,
        aiGitModel,
        aiGitFallback,
        aiChatModel,
        aiChatFallback,
      )
      setAiApiKey("")
      const ai = await AppService.GetAIConfig()
      if (ai) {
        setAiKeyMasked(ai.apiKeyMasked || "")
        setAiGitModel(ai.gitModel || "")
        setAiGitFallback(ai.gitFallback || "")
        setAiChatModel(ai.chatModel || "")
        setAiChatFallback(ai.chatFallback || "")
        setAiSuggestions(ai.modelSuggestions ?? [])
      }
    } catch (e) {
      setError(errText(e))
    } finally {
      setAiBusy(false)
    }
  }

  const setValidateCommit = async (enabled: boolean) => {
    try {
      await AppService.SetValidateCommit(enabled)
      await reloadPrefs()
    } catch (e) {
      setError(errText(e))
    }
  }

  const setValidatePR = async (enabled: boolean) => {
    try {
      await AppService.SetValidatePR(enabled)
      await reloadPrefs()
    } catch (e) {
      setError(errText(e))
    }
  }

  const saveAlias = async (path: string) => {
    const alias = aliasDrafts[path] ?? ""
    try {
      await AppService.SetPinnedAlias(path, alias)
      await refreshStatuses()
      await reloadPrefs()
    } catch (e) {
      setError(errText(e))
    }
  }

  const checkForUpdates = async () => {
    setUpdateBusy(true)
    setError(null)
    try {
      const res = await AppService.CheckForUpdates()
      setUpdateResult(res)
    } catch (e) {
      setError(errText(e))
    } finally {
      setUpdateBusy(false)
    }
  }

  const installUpdate = async () => {
    setUpdateBusy(true)
    setError(null)
    try {
      await AppService.InstallUpdate()
      await AppService.RestartAfterUpdate()
    } catch (e) {
      setError(errText(e))
    } finally {
      setUpdateBusy(false)
    }
  }

  /* --------------------------- lifecycle ---------------------------- */

  // Host shell when switching / closing projects.
  useEffect(() => {
    setTermSession({ kind: "host" })
    setDockerEnvOpen(false)
  }, [dash?.path])

  // After the fast dashboard lands, load Docker + open PR off the critical path.
  useEffect(() => {
    if (!dash?.path) {
      setDockerLoading(false)
      return
    }
    const path = dash.path
    const token = `${path}|${dash.headHash}|${dash.branch}`
    let cancelled = false
    setDockerLoading(!!dash.hasDocker)
    ;(async () => {
      try {
        const tasks: Promise<void>[] = []
        if (dash.hasDocker) {
          tasks.push(
            AppService.RefreshDockerStatus().then((docker) => {
              if (cancelled || !docker) return
              setDash((prev) => {
                if (!prev || prev.path !== path) return prev
                if (`${prev.path}|${prev.headHash}|${prev.branch}` !== token) return prev
                return { ...prev, docker, hasDocker: docker.available }
              })
            }),
          )
        }
        tasks.push(
          AppService.RefreshOpenPR().then((pr) => {
            if (cancelled) return
            setDash((prev) => {
              if (!prev || prev.path !== path) return prev
              if (`${prev.path}|${prev.headHash}|${prev.branch}` !== token) return prev
              return { ...prev, openPR: pr ?? undefined }
            })
          }),
        )
        await Promise.all(tasks)
      } catch (e) {
        if (!cancelled) setError(errText(e))
      } finally {
        if (!cancelled) setDockerLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [dash?.path, dash?.headHash, dash?.branch, dash?.hasDocker])

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        await AppService.Ping()
        const [ver, p] = await Promise.all([
          AppService.Version(),
          AppService.GetPrefs(),
        ])
        if (cancelled) return
        setVersion(ver)
        setPrefs(p)
        await refreshStatuses()
      } catch (e) {
        if (!cancelled) setError(errText(e))
      }
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Latest-action ref so tray/event handlers avoid stale closures.
  const actionsRef = useRef({
    openDialog,
    refresh,
    startCommit,
    startPR,
    openSettings,
    refreshStatuses,
    dockerUp: () => dockerAction(() => AppService.DockerUp(false)),
  })
  actionsRef.current = {
    openDialog,
    refresh,
    startCommit,
    startPR,
    openSettings,
    refreshStatuses,
    dockerUp: () => dockerAction(() => AppService.DockerUp(false)),
  }

  useEffect(() => {
    const offTray = Events.On("tray:action", (ev) => {
      const action = String((ev?.data as unknown) ?? "")
      const A = actionsRef.current
      switch (action) {
        case "open":
          void A.openDialog()
          break
        case "refresh":
          void A.refresh()
          break
        case "commit":
          void A.startCommit()
          break
        case "pr":
          void A.startPR()
          break
        case "docker":
          void A.dockerUp()
          break
        case "settings":
          void A.openSettings()
          break
        default:
          break
      }
    })

    const offStatus = Events.On("project:status", () => {
      void actionsRef.current.refreshStatuses()
    })

    const offDashboard = Events.On("project:dashboard", (ev) => {
      const raw = ev?.data ?? ev
      const d = raw as Dashboard | null
      if (d && typeof d === "object" && "path" in d) {
        setDash(d)
        void actionsRef.current.refreshStatuses()
      }
    })

    return () => {
      offTray()
      offStatus()
      offDashboard()
    }
  }, [])

  /* ----------------------------- render ----------------------------- */

  const recent = prefs?.recent ?? []
  const dockerVisible = dash?.docker?.visible ?? false
  const { widthPx, commitWidth, style: sidebarWidthStyle } = useSidebarWidth()

  return (
    <SidebarProvider
      open={terminalOpen}
      onOpenChange={setTerminalOpen}
      defaultOpen
      className="h-svh max-h-svh min-h-0 overflow-hidden"
      style={sidebarWidthStyle}
    >
      <SidebarInset className="flex h-svh max-h-svh min-h-0 flex-col overflow-hidden bg-background text-foreground">
      {/* Header (draggable) */}
      <header
        className="relative flex h-12 shrink-0 items-center border-b px-4 pl-20 [--wails-draggable:drag]"
        onDoubleClick={() => void Window.ToggleMaximise()}
      >
        <span className="pointer-events-none absolute inset-x-0 text-center text-sm font-medium">
          openbench
        </span>
        {dash && (
          <span
            className="relative z-10 max-w-[40%] truncate font-mono text-xs text-muted-foreground [--wails-draggable:no-drag]"
            title={dash.path}
          >
            {dash.repoName}
          </span>
        )}
        <div className="relative z-10 ml-auto flex items-center gap-1 [--wails-draggable:no-drag]">
          <SidebarTrigger title={terminalOpen ? "Fechar terminal (⌘B)" : "Abrir terminal (⌘B)"} />
          {dash && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => void closeProject()}
              title="Fechar projeto"
            >
              <X />
              Fechar projeto
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => setUsageOpen(true)}
            title="Uso de tokens"
          >
            <ChartColumn />
          </Button>
          <Button variant="ghost" size="icon-sm" onClick={() => void openSettings()} title="Configurações">
            <Settings />
          </Button>
        </div>
      </header>

      {/* Body — altura travada no viewport; scroll externo só se o chrome não couber */}
      <div
        className={
          dash
            ? "flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4"
            : "flex min-h-0 flex-1 flex-col gap-4 overflow-hidden p-4"
        }
      >
        {error && (
          <Alert variant="destructive" className="shrink-0">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span className="truncate">{error}</span>
              <Button variant="ghost" size="xs" onClick={() => setError(null)}>
                Fechar
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {dash ? (
          <>
            {statuses.length > 0 && (
              <div className="shrink-0">
                <ProjectTabs
                  statuses={statuses}
                  activePath={dash.path}
                  onSwitch={(p) => void switchProject(p)}
                  onUnpin={(p) => void unpinProject(p)}
                />
              </div>
            )}

            {/* Toolbar */}
            <div className="flex shrink-0 flex-wrap items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => void openDialog()} disabled={busy}>
                <FolderOpen />
                Abrir
              </Button>
              <Button variant="outline" size="sm" onClick={() => void refresh()} disabled={busy}>
                {busy ? <Loader2 className="animate-spin" /> : <RefreshCw />}
                Atualizar
              </Button>
              <Separator orientation="vertical" className="h-6" />
              <div className="flex items-stretch">
                <Button
                  size="sm"
                  className="rounded-r-none"
                  onClick={() => void startCommit()}
                  disabled={busy}
                >
                  <GitCommit />
                  Commit
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <Button
                        size="sm"
                        className="rounded-l-none border-l border-primary-foreground/20 px-1.5"
                        disabled={busy}
                        aria-label="Mais opções de commit"
                      />
                    }
                  >
                    <ChevronDown className="size-3.5" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start" className="min-w-56">
                    {isOnBase(dash) ? (
                      <>
                        <DropdownMenuLabel>Na base ({dash.baseBranch})</DropdownMenuLabel>
                        <DropdownMenuItem
                          onClick={() => void startCommitAction("branch-commit")}
                        >
                          Create Branch & Commit
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => void startCommitAction("branch-commit-push")}
                        >
                          Create Branch, Commit & Push
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onClick={() => void startCommitAction("commit")}>
                          Commit
                        </DropdownMenuItem>
                      </>
                    ) : (
                      <>
                        <DropdownMenuLabel>Nesta branch</DropdownMenuLabel>
                        <DropdownMenuItem onClick={() => void startCommitAction("commit")}>
                          Commit
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => void startCommitAction("push")}>
                          Commit & Push
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => void startCommitAction("pr")}>
                          Commit & Create PR
                        </DropdownMenuItem>
                      </>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <Button size="sm" variant="secondary" onClick={() => void startPR()} disabled={busy}>
                <GitPullRequest />
                Pull Request
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => void openSync()}
                disabled={busy || syncBusy || dash.dirty}
                title={
                  dash.dirty
                    ? "Working tree dirty — commit ou stash antes de sincronizar"
                    : `Sincronizar ${dash.baseBranch || "main"}`
                }
              >
                <ArrowDownUp />
                Sync
              </Button>
            </div>

            {dockerVisible && (
              <Card size="sm" className="shrink-0">
                <CardHeader
                  className="cursor-pointer rounded-t-xl hover:bg-muted/40"
                  onClick={() => !dockerLoading && setDockerEnvOpen(true)}
                  title="Abrir containers, shell e presets"
                >
                  <CardTitle className="flex items-center gap-2 text-sm">
                    <Container className="size-4 text-muted-foreground" />
                    Docker
                    {dockerLoading ? (
                      <Badge variant="outline" className="ml-1 gap-1 font-normal">
                        <Loader2 className="size-3 animate-spin" />
                        carregando
                      </Badge>
                    ) : (
                      <>
                        <Badge variant="outline" className="ml-1 font-normal">
                          {dash.docker.running}/{dash.docker.total}
                        </Badge>
                        <span className="ml-1 truncate text-xs font-normal text-muted-foreground">
                          {dash.docker.summary}
                        </span>
                        <span className="ml-auto text-[11px] font-normal text-muted-foreground">
                          containers →
                        </span>
                      </>
                    )}
                  </CardTitle>
                </CardHeader>
                <CardContent className="flex flex-wrap gap-2">
                  {dockerLoading ? (
                    <p className="text-xs text-muted-foreground">Consultando Docker / Compose…</p>
                  ) : (
                    <>
                      <Button
                        size="xs"
                        onClick={(e) => {
                          e.stopPropagation()
                          void dockerAction(() => AppService.DockerUp(false))
                        }}
                        disabled={busy}
                      >
                        <Play />
                        Up
                      </Button>
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={(e) => {
                          e.stopPropagation()
                          void dockerAction(() => AppService.DockerUp(true))
                        }}
                        disabled={busy}
                      >
                        Up --build
                      </Button>
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={(e) => {
                          e.stopPropagation()
                          void dockerAction(() => AppService.DockerStart())
                        }}
                        disabled={busy}
                      >
                        Start
                      </Button>
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={(e) => {
                          e.stopPropagation()
                          void dockerAction(() => AppService.DockerStop())
                        }}
                        disabled={busy}
                      >
                        <Square />
                        Stop
                      </Button>
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={(e) => {
                          e.stopPropagation()
                          openRecreate()
                        }}
                        disabled={busy || !(dash.docker.services?.length)}
                      >
                        <RefreshCw />
                        Recreate
                      </Button>
                      <Button
                        size="xs"
                        variant="destructive"
                        onClick={(e) => {
                          e.stopPropagation()
                          void dockerAction(() => AppService.DockerDown())
                        }}
                        disabled={busy}
                      >
                        Down
                      </Button>
                    </>
                  )}
                </CardContent>
              </Card>
            )}

            <DashboardView
              dash={dash}
              busy={busy}
              onSelectFile={(f) => void openFileDiff(f)}
              onOpenBranches={() => void openBranches()}
              onRecommendCommit={() => void startCommit()}
            />
          </>
        ) : (
          <Welcome
            recent={recent}
            pinned={prefs?.pinned ?? []}
            busy={busy}
            onOpenDialog={() => void openDialog()}
            onOpenPath={(p) => void openPath(p)}
            onPin={(p) => void pinProject(p)}
            onUnpin={(p) => void unpinProject(p)}
          />
        )}
      </div>

      {/* Diff — bottom sheet limitado ao viewport; scroll só no viewer */}
      <Sheet open={diffOpen} onOpenChange={setDiffOpen}>
        <SheetContent
          side="bottom"
          className="flex h-[92svh] max-h-svh flex-col gap-0 overflow-hidden p-0"
        >
          <SheetHeader className="shrink-0 border-b">
            <SheetTitle className="flex items-center gap-2 pr-8">
              <FileText className="size-4 shrink-0" />
              <span className="truncate">
                {fileDiff ? fileDiff.path : "Carregando diff…"}
              </span>
            </SheetTitle>
          </SheetHeader>
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden p-4">
            {fileDiff ? (
              <DiffViewer diff={fileDiff} />
            ) : (
              <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
                <Loader2 className="mr-2 size-4 animate-spin" />
                Carregando diff…
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>

      {/* Branches — list + checkout / create wizard */}
      <Sheet
        open={branchesOpen}
        onOpenChange={(open) => {
          setBranchesOpen(open)
          if (!open) {
            setCheckoutConfirm(null)
            cancelCreateBranch()
          }
        }}
      >
        <SheetContent side="right" className="flex w-full flex-col p-0 sm:max-w-md">
          <SheetHeader className="border-b">
            <SheetTitle className="flex items-center gap-2">
              {createBranchStep ? (
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="shrink-0"
                  onClick={() => {
                    if (createBranchStep === "from") cancelCreateBranch()
                    else if (createBranchStep === "template") setCreateBranchStep("from")
                    else setCreateBranchStep("template")
                  }}
                  disabled={createBranchBusy}
                  title="Voltar"
                >
                  <ChevronLeft />
                </Button>
              ) : (
                <GitBranch className="size-4" />
              )}
              {createBranchStep === "from"
                ? "Nova branch · From"
                : createBranchStep === "template"
                  ? "Nova branch · Preset"
                  : createBranchStep === "name"
                    ? "Nova branch · Nome"
                    : "Branches"}
            </SheetTitle>
          </SheetHeader>
          <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
            {createBranchStep === null ? (
              <>
                <Input
                  placeholder="Filtrar branches…"
                  value={branchFilter}
                  onChange={(e) => setBranchFilter(e.target.value)}
                  disabled={branchesLoading || checkoutBusy}
                />
                {dash?.dirty && (
                  <Alert>
                    <AlertDescription>
                      Working tree dirty — checkout pode falhar se houver conflito.
                    </AlertDescription>
                  </Alert>
                )}
                <ScrollArea className="min-h-0 flex-1">
                  {branchesLoading ? (
                    <div className="flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground">
                      <Loader2 className="size-4 animate-spin" />
                      Carregando branches…
                    </div>
                  ) : filteredBranches.length === 0 ? (
                    <p className="py-8 text-center text-sm text-muted-foreground">
                      Nenhuma branch encontrada.
                    </p>
                  ) : (
                    <div className="flex flex-col gap-1 pr-3">
                      {filteredBranches.map((b) => (
                        <button
                          key={b.name}
                          type="button"
                          disabled={checkoutBusy || b.current}
                          onClick={() => requestCheckout(b.name)}
                          className="flex flex-col gap-1 rounded-lg border px-3 py-2 text-left transition-colors hover:bg-muted/50 disabled:cursor-default disabled:opacity-100 data-[current=true]:border-primary/40 data-[current=true]:bg-primary/5"
                          data-current={b.current ? "true" : undefined}
                        >
                          <div className="flex items-center gap-2">
                            <span className="truncate font-mono text-sm">{b.name}</span>
                            {b.current && (
                              <Badge variant="secondary" className="shrink-0 font-normal">
                                atual
                              </Badge>
                            )}
                          </div>
                          <div className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                            {b.upstream ? (
                              <span className="truncate">{b.upstream}</span>
                            ) : (
                              <span>sem upstream</span>
                            )}
                            {b.ahead > 0 && <Badge variant="outline">↑{b.ahead}</Badge>}
                            {b.behind > 0 && <Badge variant="outline">↓{b.behind}</Badge>}
                          </div>
                        </button>
                      ))}
                    </div>
                  )}
                </ScrollArea>
                {checkoutBusy && (
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="size-4 animate-spin" />
                    Fazendo checkout…
                  </div>
                )}
                <Button
                  className="shrink-0"
                  onClick={startCreateBranch}
                  disabled={branchesLoading || checkoutBusy || createBranchBusy}
                >
                  <Plus />
                  Nova branch
                </Button>
              </>
            ) : createBranchStep === "from" ? (
              <>
                <p className="shrink-0 text-sm text-muted-foreground">
                  Escolha a branch de origem (from).
                </p>
                <ScrollArea className="min-h-0 flex-1">
                  <div className="flex flex-col gap-1 pr-3">
                    {branches.map((b) => (
                      <button
                        key={b.name}
                        type="button"
                        onClick={() => selectCreateFrom(b.name)}
                        className="flex flex-col gap-1 rounded-lg border px-3 py-2 text-left transition-colors hover:bg-muted/50 data-[active=true]:border-primary/50 data-[active=true]:bg-primary/5"
                        data-active={createBranchFrom === b.name ? "true" : undefined}
                      >
                        <div className="flex items-center gap-2">
                          <span className="truncate font-mono text-sm">{b.name}</span>
                          {b.current && (
                            <Badge variant="secondary" className="shrink-0 font-normal">
                              atual
                            </Badge>
                          )}
                          {dash?.baseBranch === b.name && (
                            <Badge variant="outline" className="shrink-0 font-normal">
                              base
                            </Badge>
                          )}
                        </div>
                      </button>
                    ))}
                  </div>
                </ScrollArea>
              </>
            ) : createBranchStep === "template" ? (
              <>
                <p className="shrink-0 text-xs text-muted-foreground">
                  From: <span className="font-mono text-foreground">{createBranchFrom}</span>
                </p>
                <ScrollArea className="min-h-0 flex-1">
                  <div className="flex flex-col gap-1 pr-3">
                    {BRANCH_TEMPLATES.map((tpl) => (
                      <div key={tpl.id}>
                        {tpl.separatorBefore && (
                          <div className="my-2 flex items-center gap-2 px-1 text-[11px] text-muted-foreground uppercase tracking-wide">
                            <Separator className="flex-1" />
                            mais
                            <Separator className="flex-1" />
                          </div>
                        )}
                        <button
                          type="button"
                          onClick={() => selectCreateTemplate(tpl)}
                          className="flex w-full flex-col gap-0.5 rounded-lg border px-3 py-2 text-left transition-colors hover:bg-muted/50"
                        >
                          <span className="font-mono text-sm">
                            {tpl.other ? "Other (livre)" : tpl.prefix}
                          </span>
                          <span className="text-xs text-muted-foreground">
                            {tpl.usage} · <span className="font-mono">{tpl.example}</span>
                          </span>
                        </button>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              </>
            ) : (
              <>
                <div className="flex shrink-0 flex-col gap-1 text-xs text-muted-foreground">
                  <span>
                    From: <span className="font-mono text-foreground">{createBranchFrom}</span>
                  </span>
                  <span>
                    Preset:{" "}
                    <span className="font-mono text-foreground">
                      {createBranchTemplate?.other
                        ? "Other"
                        : createBranchTemplate?.prefix || "—"}
                    </span>
                  </span>
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="create-branch-name">Nome da branch</Label>
                  <Input
                    id="create-branch-name"
                    value={createBranchName}
                    onChange={(e) => setCreateBranchName(e.target.value)}
                    placeholder={createBranchTemplate?.example || "feature/minha-mudanca"}
                    autoFocus
                    disabled={createBranchBusy}
                    onKeyDown={(e) => {
                      if (
                        e.key === "Enter" &&
                        isValidBranchName(createBranchName) &&
                        !createBranchBusy
                      ) {
                        void confirmCreateBranch()
                      }
                    }}
                  />
                  {createBranchName.trim() && !isValidBranchName(createBranchName) && (
                    <p className="text-xs text-destructive">Nome de branch inválido.</p>
                  )}
                  <p className="text-xs text-muted-foreground">
                    Preview:{" "}
                    <span className="font-mono text-foreground">
                      {createBranchName.trim() || "(digite um nome)"}
                    </span>
                  </p>
                </div>
                <div className="mt-auto flex shrink-0 gap-2">
                  <Button
                    variant="outline"
                    className="flex-1"
                    onClick={() => setCreateBranchStep("template")}
                    disabled={createBranchBusy}
                  >
                    Voltar
                  </Button>
                  <Button
                    className="flex-1"
                    onClick={() => void confirmCreateBranch()}
                    disabled={
                      createBranchBusy ||
                      !createBranchFrom.trim() ||
                      !isValidBranchName(createBranchName)
                    }
                  >
                    {createBranchBusy ? <Loader2 className="animate-spin" /> : <GitBranch />}
                    Criar branch
                  </Button>
                </div>
              </>
            )}
          </div>
        </SheetContent>
      </Sheet>

      <Dialog
        open={checkoutConfirm !== null}
        onOpenChange={(open) => {
          if (!open) setCheckoutConfirm(null)
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Working tree dirty</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Há alterações não commitadas. Continuar o checkout para{" "}
            <span className="font-mono text-foreground">{checkoutConfirm}</span>?
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCheckoutConfirm(null)} disabled={checkoutBusy}>
              Cancelar
            </Button>
            <Button
              onClick={() => {
                if (checkoutConfirm) void doCheckout(checkoutConfirm)
              }}
              disabled={checkoutBusy || !checkoutConfirm}
            >
              {checkoutBusy ? <Loader2 className="animate-spin" /> : null}
              Continuar checkout
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Sync dialog */}
      <Dialog
        open={syncOpen}
        onOpenChange={(open) => {
          setSyncOpen(open)
          if (!open) setSyncResult(null)
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <ArrowDownUp className="size-4" />
              Sync · {dash?.baseBranch || "main"}
            </DialogTitle>
          </DialogHeader>

          {dash?.dirty ? (
            <Alert variant="destructive">
              <AlertDescription>
                Working tree dirty — commit ou stash antes de sincronizar.
              </AlertDescription>
            </Alert>
          ) : syncResult ? (
            <div className="flex flex-col gap-3">
              <Alert>
                <AlertDescription>{syncResult.message}</AlertDescription>
              </Alert>
              {syncResult.logs && syncResult.logs.length > 0 && (
                <ScrollArea className="max-h-48 rounded-md border">
                  <div className="space-y-1 p-3 font-mono text-xs text-muted-foreground">
                    {syncResult.logs.map((line, i) => (
                      <div key={`${i}-${line}`}>{line}</div>
                    ))}
                  </div>
                </ScrollArea>
              )}
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              <p className="text-sm text-muted-foreground">
                Escolha o modo. A base{" "}
                <span className="font-mono text-foreground">{dash?.baseBranch || "main"}</span> será
                atualizada com origin (fast-forward).
              </p>
              <div className="flex flex-col gap-2">
                {syncModes.map((m) => (
                  <button
                    key={m.id}
                    type="button"
                    disabled={syncBusy}
                    onClick={() => setSyncMode(m.id)}
                    className="rounded-lg border px-3 py-2.5 text-left transition-colors hover:bg-muted/50 data-[active=true]:border-primary/50 data-[active=true]:bg-primary/5"
                    data-active={syncMode === m.id ? "true" : undefined}
                  >
                    <div className="text-sm font-medium">{m.label}</div>
                    <div className="mt-0.5 text-xs text-muted-foreground">{m.summary}</div>
                  </button>
                ))}
              </div>
              {selectedSyncMode && (
                <p className="text-xs text-muted-foreground">{selectedSyncMode.description}</p>
              )}
            </div>
          )}

          <DialogFooter>
            {syncResult ? (
              <Button onClick={() => setSyncOpen(false)}>Fechar</Button>
            ) : (
              <>
                <Button
                  variant="outline"
                  onClick={() => setSyncOpen(false)}
                  disabled={syncBusy}
                >
                  Cancelar
                </Button>
                <Button
                  onClick={() => void runSync()}
                  disabled={syncBusy || !!dash?.dirty || !syncMode}
                >
                  {syncBusy ? <Loader2 className="animate-spin" /> : <ArrowDownUp />}
                  Executar sync
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Commit dialog */}
      <Dialog open={commitOpen} onOpenChange={setCommitOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <GitCommit className="size-4" />
              Revisar commit
            </DialogTitle>
          </DialogHeader>
          {commitBusy && !commitPreview ? (
            <div className="flex items-center justify-center gap-2 py-8 text-sm text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Gerando mensagem com IA…
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="commit-msg">Mensagem</Label>
                <Textarea
                  id="commit-msg"
                  value={commitMessage}
                  onChange={(e) => setCommitMessage(e.target.value)}
                  className="min-h-40 font-mono text-xs"
                />
              </div>
              {commitPreview?.notes && commitPreview.notes.length > 0 && (
                <Alert>
                  <AlertDescription>
                    <ul className="list-disc pl-4 text-xs">
                      {commitPreview.notes.map((n, i) => (
                        <li key={i}>{n}</li>
                      ))}
                    </ul>
                  </AlertDescription>
                </Alert>
              )}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setCommitOpen(false)} disabled={commitBusy}>
              Cancelar
            </Button>
            <Button onClick={() => void confirmCommit()} disabled={commitBusy || !commitMessage.trim()}>
              {commitBusy ? <Loader2 className="animate-spin" /> : <GitCommit />}
              {commitConfirmLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* New branch before commit */}
      <Dialog open={newBranchOpen} onOpenChange={setNewBranchOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <GitBranch className="size-4" />
              Nova branch
            </DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <p className="text-sm text-muted-foreground">
              Criar branch a partir de{" "}
              <span className="font-mono text-foreground">{newBranchFrom || "main"}</span> e
              seguir para o commit.
            </p>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="new-branch-name">Nome</Label>
              <Input
                id="new-branch-name"
                value={newBranchName}
                onChange={(e) => setNewBranchName(e.target.value)}
                placeholder="feature/minha-mudanca"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === "Enter" && newBranchName.trim()) void confirmNewBranch()
                }}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setNewBranchOpen(false)}
              disabled={newBranchBusy}
            >
              Cancelar
            </Button>
            <Button
              onClick={() => void confirmNewBranch()}
              disabled={newBranchBusy || !newBranchName.trim()}
            >
              {newBranchBusy ? <Loader2 className="animate-spin" /> : <GitBranch />}
              Criar e continuar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* PR dialog */}
      <Dialog open={prOpen} onOpenChange={setPrOpen}>
        <DialogContent className="flex h-[min(85vh,720px)] flex-col gap-4 overflow-hidden sm:max-w-3xl">
          <DialogHeader className="shrink-0">
            <DialogTitle className="flex items-center gap-2">
              <GitPullRequest className="size-4" />
              Revisar Pull Request
            </DialogTitle>
          </DialogHeader>
          {prBusy && !prPreview ? (
            <div className="flex flex-1 items-center justify-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              Gerando PR com IA…
            </div>
          ) : (
            <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
              <div className="flex shrink-0 flex-col gap-1.5">
                <Label htmlFor="pr-title">Título</Label>
                <Input
                  id="pr-title"
                  value={prTitle}
                  onChange={(e) => setPrTitle(e.target.value)}
                />
                {prPreview?.base && (
                  <p className="text-xs text-muted-foreground">base: {prPreview.base}</p>
                )}
              </div>
              <div className="flex min-h-0 flex-1 flex-col gap-1.5">
                <Label htmlFor="pr-body" className="shrink-0">
                  Descrição
                </Label>
                <div className="min-h-0 flex-1">
                  <Textarea
                    id="pr-body"
                    value={prBody}
                    onChange={(e) => setPrBody(e.target.value)}
                    className="h-full min-h-0 resize-none field-sizing-fixed overflow-y-auto text-xs"
                  />
                </div>
              </div>
            </div>
          )}
          <DialogFooter className="shrink-0 sm:flex-row sm:items-center sm:justify-end">
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <span className="inline-flex items-center gap-2 text-sm text-foreground" />
                }
              >
                <Switch
                  id="pr-draft"
                  checked={prDraft}
                  onCheckedChange={(checked) => setPrDraft(checked)}
                  disabled={prBusy}
                />
                Draft
              </TooltipTrigger>
              <TooltipContent side="top" className="max-w-xs text-left leading-relaxed">
                PR em rascunho (draft): fica oculto da lista padrão de reviews até você marcar
                como pronto. Use para trabalho em progresso sem pedir review ainda.
              </TooltipContent>
            </Tooltip>
            <Button variant="outline" onClick={() => setPrOpen(false)} disabled={prBusy}>
              Cancelar
            </Button>
            <Button onClick={() => void confirmPR()} disabled={prBusy || !prTitle.trim()}>
              {prBusy ? <Loader2 className="animate-spin" /> : <GitPullRequest />}
              Criar PR
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Onboarding dialog */}
      <Dialog open={onboardingOpen} onOpenChange={setOnboardingOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Configurar IA</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            {onboarding?.issues && onboarding.issues.length > 0 && (
              <Alert>
                <AlertDescription>
                  <ul className="list-disc pl-4 text-xs">
                    {onboarding.issues.map((iss) => (
                      <li key={iss.id}>
                        <strong>{iss.title}</strong> — {iss.hint}
                      </li>
                    ))}
                  </ul>
                </AlertDescription>
              </Alert>
            )}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ob-provider">Provider</Label>
              <Input
                id="ob-provider"
                value={obProvider}
                onChange={(e) => setObProvider(e.target.value)}
                placeholder="openai | openrouter | gemini"
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ob-key">API Key</Label>
              <Input
                id="ob-key"
                type="password"
                value={obApiKey}
                onChange={(e) => setObApiKey(e.target.value)}
                placeholder={onboarding?.apiKeyMasked || "sk-…"}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ob-model">Modelo</Label>
              <Input
                id="ob-model"
                value={obModel}
                onChange={(e) => setObModel(e.target.value)}
                placeholder="gpt-4o-mini"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOnboardingOpen(false)} disabled={obBusy}>
              Cancelar
            </Button>
            <Button onClick={() => void saveOnboarding()} disabled={obBusy}>
              {obBusy ? <Loader2 className="animate-spin" /> : null}
              Salvar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Usage dialog */}
      <Dialog open={usageOpen} onOpenChange={setUsageOpen}>
        <DialogContent className="flex max-h-[90vh] flex-col gap-0 overflow-hidden p-0 sm:max-w-4xl">
          <DialogHeader className="border-b px-6 py-4">
            <DialogTitle className="flex items-center gap-2">
              <ChartColumn className="size-4" />
              Uso de tokens
            </DialogTitle>
          </DialogHeader>
          <ScrollArea className="min-h-0 flex-1 px-6 py-4">
            <UsageChartPanel open={usageOpen} />
          </ScrollArea>
        </DialogContent>
      </Dialog>

      {/* Settings dialog */}
      <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Settings className="size-4" />
              Configurações
            </DialogTitle>
          </DialogHeader>

          <Tabs defaultValue="geral" className="min-h-0 w-full gap-3">
            <TabsList className="w-full">
              <TabsTrigger value="geral" className="flex-1">
                Geral
              </TabsTrigger>
              <TabsTrigger value="ia" className="flex-1">
                IA
              </TabsTrigger>
              <TabsTrigger value="atualizacoes" className="flex-1">
                Atualizações
              </TabsTrigger>
              <TabsTrigger value="sobre" className="flex-1">
                Sobre
              </TabsTrigger>
            </TabsList>

            <TabsContent value="geral" className="flex flex-col gap-4 pt-1">
              <section className="flex flex-col gap-2">
                <h3 className="text-xs font-medium text-muted-foreground uppercase">
                  Aparência
                </h3>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="theme-mode">Tema</Label>
                  <Select
                    value={theme}
                    onValueChange={(v) => {
                      if (v === "light" || v === "dark" || v === "system") {
                        setTheme(v)
                      }
                    }}
                  >
                    <SelectTrigger id="theme-mode" className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="w-(--anchor-width)">
                      <SelectItem value="system">Sistema</SelectItem>
                      <SelectItem value="light">Claro</SelectItem>
                      <SelectItem value="dark">Escuro</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </section>

              <Separator />

              <section className="flex flex-col gap-2">
                <h3 className="text-xs font-medium text-muted-foreground uppercase">
                  Validação
                </h3>
                <Label className="cursor-pointer">
                  <Checkbox
                    checked={prefs?.validateCommit ?? false}
                    onCheckedChange={(c) => void setValidateCommit(c === true)}
                  />
                  Validar antes do commit
                </Label>
                <Label className="cursor-pointer">
                  <Checkbox
                    checked={prefs?.validatePR ?? false}
                    onCheckedChange={(c) => void setValidatePR(c === true)}
                  />
                  Validar antes do PR
                </Label>
              </section>

              {prefs?.pinned && prefs.pinned.length > 0 && (
                <>
                  <Separator />
                  <section className="flex flex-col gap-2">
                    <h3 className="text-xs font-medium text-muted-foreground uppercase">
                      Apelidos dos projetos fixados
                    </h3>
                    <div className="flex flex-col gap-2">
                      {prefs.pinned.map((pp) => (
                        <div key={pp.path} className="flex items-center gap-2">
                          <span
                            className="w-40 shrink-0 truncate font-mono text-xs text-muted-foreground"
                            title={pp.path}
                          >
                            {pp.path}
                          </span>
                          <Input
                            value={aliasDrafts[pp.path] ?? pp.alias ?? ""}
                            onChange={(e) =>
                              setAliasDrafts((prev) => ({
                                ...prev,
                                [pp.path]: e.target.value,
                              }))
                            }
                            onBlur={() => void saveAlias(pp.path)}
                            placeholder="apelido"
                            className="h-7"
                          />
                        </div>
                      ))}
                    </div>
                  </section>
                </>
              )}

              {prefsPath && (
                <>
                  <Separator />
                  <p className="font-mono text-[11px] text-muted-foreground">
                    Config: {prefsPath}
                  </p>
                </>
              )}
            </TabsContent>

            <TabsContent value="ia" className="flex max-h-[min(70vh,32rem)] flex-col gap-4 overflow-y-auto pt-1">
              <section className="flex flex-col gap-2">
                <h3 className="text-xs font-medium text-muted-foreground uppercase">
                  Conta
                </h3>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ai-provider">Provider</Label>
                  <Select
                    value={aiProvider}
                    onValueChange={(v) => setAiProvider(String(v ?? "openrouter"))}
                  >
                    <SelectTrigger id="ai-provider" className="w-full">
                      <SelectValue>{aiProvider}</SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="openrouter">openrouter</SelectItem>
                      <SelectItem value="openai">openai</SelectItem>
                      <SelectItem value="gemini">gemini</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ai-key">API Key</Label>
                  <Input
                    id="ai-key"
                    type="password"
                    value={aiApiKey}
                    onChange={(e) => setAiApiKey(e.target.value)}
                    placeholder={aiKeyMasked || "sk-… / AIza…"}
                  />
                  <p className="text-[11px] text-muted-foreground">
                    Deixe em branco para manter a chave atual
                    {aiKeyMasked ? ` (${aiKeyMasked})` : ""}.
                  </p>
                </div>
              </section>

              <Separator />

              <section className="flex flex-col gap-2">
                <h3 className="text-xs font-medium text-muted-foreground uppercase">
                  Chat IA
                </h3>
                <p className="text-[11px] text-muted-foreground">
                  Modelos usados no painel de chat do projeto.
                </p>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ai-chat-model">Modelo primário</Label>
                  <Input
                    id="ai-chat-model"
                    value={aiChatModel}
                    onChange={(e) => setAiChatModel(e.target.value)}
                    list="ai-model-suggestions"
                    placeholder="ex.: gemini-2.5-flash-lite"
                    className="font-mono text-xs"
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ai-chat-fallback">Fallback</Label>
                  <Input
                    id="ai-chat-fallback"
                    value={aiChatFallback}
                    onChange={(e) => setAiChatFallback(e.target.value)}
                    list="ai-model-suggestions"
                    placeholder="opcional"
                    className="font-mono text-xs"
                  />
                </div>
              </section>

              <Separator />

              <section className="flex flex-col gap-2">
                <h3 className="text-xs font-medium text-muted-foreground uppercase">
                  Git IA
                </h3>
                <p className="text-[11px] text-muted-foreground">
                  Modelos usados em commits e Pull Requests.
                </p>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ai-git-model">Modelo primário</Label>
                  <Input
                    id="ai-git-model"
                    value={aiGitModel}
                    onChange={(e) => setAiGitModel(e.target.value)}
                    list="ai-model-suggestions"
                    placeholder="ex.: gemini-2.5-flash-lite"
                    className="font-mono text-xs"
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ai-git-fallback">Fallback</Label>
                  <Input
                    id="ai-git-fallback"
                    value={aiGitFallback}
                    onChange={(e) => setAiGitFallback(e.target.value)}
                    list="ai-model-suggestions"
                    placeholder="opcional"
                    className="font-mono text-xs"
                  />
                </div>
              </section>

              <datalist id="ai-model-suggestions">
                {aiSuggestions.map((m) => (
                  <option key={m} value={m} />
                ))}
              </datalist>

              {aiConfigPath && (
                <p className="font-mono text-[11px] text-muted-foreground">
                  Config: {aiConfigPath}
                </p>
              )}

              <Button onClick={() => void saveAISettings()} disabled={aiBusy}>
                {aiBusy ? <Loader2 className="animate-spin" /> : null}
                Salvar IA
              </Button>
            </TabsContent>

            <TabsContent value="atualizacoes" className="flex flex-col gap-3 pt-1">
              <p className="text-sm text-muted-foreground">
                Verifique se há uma versão mais recente do openbench no GitHub Releases.
              </p>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void checkForUpdates()}
                  disabled={updateBusy}
                >
                  {updateBusy ? <Loader2 className="animate-spin" /> : <Download />}
                  Verificar atualizações
                </Button>
                {updateResult?.available && (
                  <Button size="sm" onClick={() => void installUpdate()} disabled={updateBusy}>
                    Instalar {updateResult.latestVersion} e reiniciar
                  </Button>
                )}
              </div>
              {updateResult && (
                <p className="text-xs text-muted-foreground">
                  {updateResult.message ||
                    (updateResult.available
                      ? `Nova versão disponível: ${updateResult.latestVersion}`
                      : `Você está atualizado (${updateResult.currentVersion}).`)}
                </p>
              )}
            </TabsContent>

            <TabsContent value="sobre" className="flex flex-col gap-4 pt-1">
              <div className="flex flex-col gap-1">
                <h3 className="text-base font-medium">openbench</h3>
                <p className="text-sm text-muted-foreground">
                  App desktop para commits com IA, Pull Requests e fluxo Docker Compose —
                  reutilizando o core Go do openbench no seu Mac.
                </p>
              </div>

              <Separator />

              <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
                <dt className="text-muted-foreground">Versão</dt>
                <dd className="font-mono text-xs">{version || "—"}</dd>

                <dt className="text-muted-foreground">Criador</dt>
                <dd>Laércio Crestani</dd>

                <dt className="text-muted-foreground">Licença</dt>
                <dd>MIT</dd>

                <dt className="text-muted-foreground">Identificador</dt>
                <dd className="font-mono text-xs">com.laerciocrestani.openbench</dd>

                <dt className="text-muted-foreground">Copyright</dt>
                <dd className="text-xs">(c) 2026 openbench</dd>
              </dl>

              <a
                href="https://github.com/laerciocrestani/openbench"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 text-sm text-sky-500 hover:underline"
              >
                <ExternalLink className="size-3.5" />
                github.com/laerciocrestani/openbench
              </a>
            </TabsContent>
          </Tabs>

          <DialogFooter>
            <Button variant="outline" onClick={() => setSettingsOpen(false)}>
              Fechar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <DockerEnvironmentSheet
        open={dockerEnvOpen}
        onOpenChange={setDockerEnvOpen}
        docker={dash?.docker}
        busy={busy}
        onOpenDockerShell={openDockerShell}
        onError={(msg) => setError(msg)}
        onStatus={() => setError(null)}
      />

      <Dialog open={recreateOpen} onOpenChange={setRecreateOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Recreate serviço</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-3 py-1">
            <p className="text-sm text-muted-foreground">
              Escolha o serviço do Compose para force-recreate (`up -d --force-recreate --no-deps`).
            </p>
            {(dash?.docker?.services?.length ?? 0) === 0 ? (
              <Alert>
                <AlertDescription>
                  Nenhum serviço listado. Rode Up antes ou verifique o compose neste projeto.
                </AlertDescription>
              </Alert>
            ) : (
              <div className="flex flex-col gap-2">
                <Label htmlFor="recreate-service">Serviço</Label>
                <Select
                  value={recreateService}
                  onValueChange={(v) => setRecreateService(String(v ?? ""))}
                >
                  <SelectTrigger id="recreate-service" className="w-full">
                    <SelectValue placeholder="Selecione um serviço" />
                  </SelectTrigger>
                  <SelectContent className="w-(--anchor-width)">
                    {(dash?.docker?.services ?? []).map((svc) => (
                      <SelectItem key={svc.name} value={svc.name}>
                        <span className="font-mono">{svc.name}</span>
                        <span className="text-muted-foreground"> · {svc.state || "unknown"}</span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRecreateOpen(false)} disabled={busy}>
              Cancelar
            </Button>
            <Button
              onClick={() => void confirmRecreate()}
              disabled={busy || !recreateService.trim()}
            >
              {busy ? <Loader2 className="animate-spin" /> : <RefreshCw />}
              Recreate
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      </SidebarInset>

      <Sidebar side="right" collapsible="offcanvas" className="border-l">
        <SidebarHeader className="flex flex-row items-center gap-2 border-b px-3 py-2">
          <span className="text-sm font-medium">{dash ? "Terminal + Chat" : "Terminal"}</span>
          <span className="text-[11px] text-muted-foreground">
            {dash ? "shell · IA" : "shell"}
          </span>
        </SidebarHeader>
        <SidebarContent className="overflow-hidden p-0">
          <TerminalChatSplit
            showChat={!!dash}
            terminal={
              <TerminalPanel
                projectPath={dash?.path ?? null}
                visible={terminalOpen}
                session={termSession}
                onResetToHost={() => setTermSession({ kind: "host" })}
              />
            }
            chat={
              dash ? (
                <ProjectChatPanel projectPath={dash.path} visible={terminalOpen} />
              ) : null
            }
          />
        </SidebarContent>
        <SidebarWidthRail widthPx={widthPx} onCommitWidth={commitWidth} />
      </Sidebar>
    </SidebarProvider>
  )
}

export default App
