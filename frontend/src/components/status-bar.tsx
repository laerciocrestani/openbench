import { Browser } from "@wailsio/runtime"
import {
  AlertTriangle,
  Container,
  ExternalLink,
  GitBranch,
  Loader2,
  Play,
  RefreshCw,
  Sparkles,
  Square,
  Workflow,
} from "lucide-react"

import type { Dashboard } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

function StatusBadge({ label, dirty }: { label: string; dirty: boolean }) {
  return (
    <Badge variant={dirty ? "secondary" : "outline"} className="h-5 font-normal">
      {label || (dirty ? "dirty" : "clean")}
    </Badge>
  )
}

function DiffStat({
  insertions = 0,
  deletions = 0,
}: {
  insertions?: number
  deletions?: number
}) {
  if (insertions <= 0 && deletions <= 0) return null
  return (
    <span
      className="inline-flex items-center gap-1 font-mono text-[11px] tabular-nums leading-none"
      title={`+${insertions} −${deletions}`}
    >
      {insertions > 0 && <span className="text-emerald-500">+{insertions}</span>}
      {deletions > 0 && <span className="text-rose-400">−{deletions}</span>}
    </span>
  )
}

function canOpenPRInWeb(dash: Dashboard): boolean {
  return Boolean(dash.openPR?.url) && dash.commitsAheadOfBase > 0 && dash.hasBranchDiff
}

function openPRDisabledReason(dash: Dashboard): string | undefined {
  if (!dash.openPR?.url) return "Nenhum pull request aberto nesta branch"
  if (dash.commitsAheadOfBase <= 0) return "Sem commits à frente da base"
  if (!dash.hasBranchDiff) return "Diff vazio em relação à base"
  return undefined
}

function StatusBarSep() {
  return (
    <span className="mx-1 shrink-0 select-none text-muted-foreground/50" aria-hidden>
      |
    </span>
  )
}

export function StatusBar({
  dash,
  busy,
  prManageBusy,
  gitLoading,
  filesLoading,
  ciLoading,
  dockerVisible,
  dockerLoading,
  onOpenBranches,
  onMarkPRReady,
  onOpenSettings,
  onOpenDockerEnv,
  onDockerUp,
  onDockerUpBuild,
  onDockerStart,
  onDockerStop,
  onDockerRecreate,
  onDockerDown,
  className,
}: {
  dash: Dashboard
  busy: boolean
  prManageBusy: boolean
  gitLoading: boolean
  filesLoading: boolean
  ciLoading: boolean
  dockerVisible: boolean
  dockerLoading: boolean
  onOpenBranches: () => void
  onMarkPRReady: () => void
  onOpenSettings: () => void
  onOpenDockerEnv: () => void
  onDockerUp: () => void
  onDockerUpBuild: () => void
  onDockerStart: () => void
  onDockerStop: () => void
  onDockerRecreate: () => void
  onDockerDown: () => void
  className?: string
}) {
  const pr = dash.openPR
  const openPREnabled = canOpenPRInWeb(dash)
  const openPRTitle = openPRDisabledReason(dash)
  const insertions =
    dash.contextIndex?.insertions ??
    (dash.changedFiles ?? []).reduce((a, f) => a + (f.insertions || 0), 0)
  const deletions =
    dash.contextIndex?.deletions ??
    (dash.changedFiles ?? []).reduce((a, f) => a + (f.deletions || 0), 0)

  const hasCompose = Boolean(dash.docker.composeFile?.trim())
  const dockerMissing = hasCompose && !dash.docker.available
  const dockerDaemonOff =
    hasCompose && dash.docker.available && !dash.docker.daemonRunning
  const dockerDown =
    hasCompose && dash.docker.daemonRunning && dash.docker.total === 0
  const dockerComposeStopped =
    hasCompose &&
    dash.docker.daemonRunning &&
    dash.docker.total > 0 &&
    dash.docker.running === 0
  const dockerRunning =
    hasCompose && dash.docker.daemonRunning && dash.docker.running > 0
  const dockerNeedsAttention =
    dockerMissing || dockerDaemonOff || dockerDown || dockerComposeStopped
  const dockerActionsEnabled =
    hasCompose && dash.docker.available && dash.docker.daemonRunning

  const dockerTitle = !dockerVisible
    ? "Docker Compose não detectado"
    : dockerLoading
      ? "Consultando Docker…"
      : dockerMissing
        ? "Docker CLI não encontrado"
        : dockerDaemonOff
          ? "Docker daemon parado"
          : dockerDown
            ? "Ambiente Down — clique para containers"
            : dockerComposeStopped
              ? "Containers parados — clique para containers"
              : dash.docker.summary || "Abrir containers, shell e presets"

  return (
    <footer
      data-slot="status-bar"
      className={cn(
        "flex h-8 w-full shrink-0 items-center justify-end gap-0 overflow-x-auto border-t bg-muted/40 px-2 text-[11px] text-muted-foreground",
        className,
      )}
    >
      {/* Branch */}
      <button
        type="button"
        onClick={onOpenBranches}
        className="inline-flex h-6 max-w-[min(18rem,40vw)] shrink-0 items-center gap-1.5 rounded-md px-1.5 hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        title="Trocar branch"
      >
        <GitBranch className="size-3.5 shrink-0" />
        <span className="truncate font-mono text-foreground">
          {dash.branch}
          {dash.detached ? " (detached)" : ""}
        </span>
        {dash.ahead > 0 && (
          <Badge variant="outline" className="h-4 px-1 font-mono text-[10px]">
            ↑{dash.ahead}
          </Badge>
        )}
        {dash.behind > 0 && (
          <Badge variant="outline" className="h-4 px-1 font-mono text-[10px]">
            ↓{dash.behind}
          </Badge>
        )}
        {dash.baseBehind > 0 && (
          <Badge variant="outline" className="h-4 px-1 font-mono text-[10px]">
            {dash.baseBranch || "base"} ↓{dash.baseBehind}
          </Badge>
        )}
        {dash.commitsAheadOfBase > 0 && (
          <Badge variant="outline" className="h-4 px-1 font-mono text-[10px]">
            {dash.commitsAheadOfBase} vs base
          </Badge>
        )}
      </button>

      <StatusBarSep />

      {/* Status */}
      <div className="flex min-w-0 shrink-0 items-center gap-1.5">
        {gitLoading ? (
          <Badge variant="outline" className="h-5 gap-1 font-normal">
            <Loader2 className="size-3 animate-spin" />
            git
          </Badge>
        ) : (
          <StatusBadge label={dash.statusLabel} dirty={dash.dirty} />
        )}
        <DiffStat insertions={insertions} deletions={deletions} />
        {filesLoading ? (
          <Badge variant="outline" className="h-5 gap-1 font-normal">
            <Loader2 className="size-3 animate-spin" />
            diffs
          </Badge>
        ) : null}
        <Badge variant="outline" className="h-5 font-normal">
          staged {dash.staged}
        </Badge>
        <Badge variant="outline" className="h-5 font-normal">
          mod {dash.modified}
        </Badge>
        <Badge variant="outline" className="h-5 font-normal">
          untracked {dash.untracked}
        </Badge>
        {ciLoading && !dash.ciLabel ? (
          <Badge variant="outline" className="h-5 gap-1 font-normal">
            <Loader2 className="size-3 animate-spin" />
            CI
          </Badge>
        ) : null}
        {dash.ciLabel ? (
          <Badge
            variant={
              dash.ciState === "fail"
                ? "destructive"
                : dash.ciState === "pass"
                  ? "secondary"
                  : "outline"
            }
            title={[dash.ciHost, dash.ciFromCache ? "cache offline" : ""]
              .filter(Boolean)
              .join(" · ")}
            className="h-5 gap-1"
          >
            <Workflow className="size-3" />
            {dash.ciLabel}
            {dash.ciFromCache ? " · off" : ""}
            {ciLoading ? <Loader2 className="size-3 animate-spin opacity-60" /> : null}
          </Badge>
        ) : null}
        {pr ? (
          <>
            <Badge variant={pr.isDraft ? "outline" : "default"} className="h-5">
              PR #{pr.number}
              {pr.isDraft ? " · draft" : ""}
            </Badge>
            {pr.checksSummary ? (
              <Badge
                variant={
                  (pr.checksFail ?? 0) > 0
                    ? "destructive"
                    : (pr.checksPending ?? 0) > 0
                      ? "outline"
                      : "secondary"
                }
                className="h-5"
                title={pr.title}
              >
                checks: {pr.checksSummary}
              </Badge>
            ) : null}
            <Button
              size="xs"
              variant="ghost"
              className="h-6 gap-1 px-1.5 text-[11px]"
              disabled={!openPREnabled}
              title={openPRTitle ?? pr.title}
              onClick={() => {
                const url = pr.url
                if (!url) return
                void Browser.OpenURL(url).catch(() => {
                  window.open(url, "_blank", "noopener,noreferrer")
                })
              }}
            >
              <ExternalLink className="size-3" />
              Open PR
            </Button>
            {pr.isDraft ? (
              <Button
                size="xs"
                variant="secondary"
                className="h-6 text-[11px]"
                disabled={busy || prManageBusy}
                onClick={onMarkPRReady}
              >
                {prManageBusy ? <Loader2 className="size-3 animate-spin" /> : null}
                Ready for review
              </Button>
            ) : null}
          </>
        ) : null}
      </div>

      <StatusBarSep />

      {/* Docker */}
      <div className="flex min-w-0 shrink-0 items-center gap-1">
        <button
          type="button"
          onClick={() => {
            if (dockerVisible && !dockerLoading) onOpenDockerEnv()
          }}
          disabled={!dockerVisible || dockerLoading}
          className="inline-flex h-6 shrink-0 items-center gap-1.5 rounded-md px-1.5 hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-60"
          title={dockerTitle}
        >
          <Container className="size-3.5 shrink-0" />
          {!dockerVisible ? (
            <Badge variant="outline" className="h-5 font-normal">
              indisponível
            </Badge>
          ) : dockerLoading ? (
            <Badge variant="outline" className="h-5 gap-1 font-normal">
              <Loader2 className="size-3 animate-spin" />
              docker
            </Badge>
          ) : (
            <>
              <Badge
                variant="outline"
                className={cn(
                  "h-5 font-normal",
                  dockerNeedsAttention &&
                    "border-amber-500/50 text-amber-700 dark:text-amber-400",
                )}
              >
                {dash.docker.running}/{dash.docker.total}
              </Badge>
              {dockerNeedsAttention ? (
                <Badge
                  variant="outline"
                  className="h-5 gap-1 border-amber-500/50 font-normal text-amber-700 dark:text-amber-400"
                >
                  <AlertTriangle className="size-3" />
                  parado
                </Badge>
              ) : null}
            </>
          )}
        </button>
        {dockerActionsEnabled ? (
          <div className="flex items-center gap-0.5">
            {dockerDown ? (
              <>
                <Button
                  size="xs"
                  variant="ghost"
                  className="h-6 px-1.5 text-[11px]"
                  onClick={onDockerUp}
                  disabled={busy}
                >
                  <Play className="size-3" />
                  Up
                </Button>
                <Button
                  size="xs"
                  variant="ghost"
                  className="h-6 px-1.5 text-[11px]"
                  onClick={onDockerUpBuild}
                  disabled={busy}
                >
                  Up --build
                </Button>
              </>
            ) : null}
            {dockerComposeStopped ? (
              <Button
                size="xs"
                variant="ghost"
                className="h-6 px-1.5 text-[11px]"
                onClick={onDockerStart}
                disabled={busy}
              >
                <Play className="size-3" />
                Start
              </Button>
            ) : null}
            {dockerRunning ? (
              <>
                <Button
                  size="xs"
                  variant="ghost"
                  className="h-6 px-1.5 text-[11px]"
                  onClick={onDockerStop}
                  disabled={busy}
                >
                  <Square className="size-3" />
                  Stop
                </Button>
                <Button
                  size="xs"
                  variant="ghost"
                  className="h-6 px-1.5 text-[11px]"
                  onClick={onDockerRecreate}
                  disabled={busy || !(dash.docker.services?.length)}
                >
                  <RefreshCw className="size-3" />
                  Recreate
                </Button>
                <Button
                  size="xs"
                  variant="ghost"
                  className="h-6 px-1.5 text-[11px] text-destructive hover:text-destructive"
                  onClick={onDockerDown}
                  disabled={busy}
                >
                  Down
                </Button>
              </>
            ) : null}
          </div>
        ) : null}
      </div>

      <StatusBarSep />

      {/* IA */}
      <button
        type="button"
        onClick={onOpenSettings}
        className="inline-flex h-6 shrink-0 items-center gap-1.5 rounded-md px-1.5 hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        title={dash.aiReady ? "Configurações de IA" : "Configurar IA"}
      >
        <Sparkles className="size-3.5 shrink-0" />
        <Badge
          variant={dash.aiReady ? "default" : "destructive"}
          className="h-5 font-normal"
        >
          {dash.aiReady ? "IA pronta" : "IA: config"}
        </Badge>
        <span className="hidden truncate sm:inline">
          {dash.provider || "—"} · {dash.model || "—"}
        </span>
      </button>
    </footer>
  )
}
