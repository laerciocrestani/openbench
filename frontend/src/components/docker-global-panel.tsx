import { useCallback, useEffect, useState } from "react"

import { AppService } from "../../bindings/github.com/laerciocrestani/openbench"
import type {
  GlobalDockerContainer,
  GlobalDockerProject,
  GlobalDockerView,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  ChevronRight,
  CircleAlert,
  Container,
  EllipsisVertical,
  Loader2,
  Play,
  RefreshCw,
  Square,
  Trash2,
} from "lucide-react"

function errText(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === "string") return e
  try {
    return JSON.stringify(e)
  } catch {
    return String(e)
  }
}

function stateVariant(state: string): "default" | "secondary" | "outline" | "destructive" {
  const s = state.toLowerCase()
  if (s === "running") return "default"
  if (s === "exited" || s === "dead") return "secondary"
  if (s === "restarting" || s === "paused") return "outline"
  return "secondary"
}

function projectTitle(proj: GlobalDockerProject): string {
  if (proj.name && proj.name !== "Standalone") return proj.name
  if (proj.workingDir) {
    const parts = proj.workingDir.replace(/[/\\]+$/, "").split(/[/\\]/)
    return parts[parts.length - 1] || proj.name
  }
  return proj.name || "Containers"
}

function projectKey(proj: GlobalDockerProject): string {
  return proj.composeFile || proj.workingDir || proj.name || "project"
}

function isRunning(state: string): boolean {
  return state.toLowerCase() === "running"
}

export function DockerGlobalPanel({
  active,
}: {
  active: boolean
}) {
  const [view, setView] = useState<GlobalDockerView | null>(null)
  const [loading, setLoading] = useState(false)
  const [busyKey, setBusyKey] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const next = await AppService.ListGlobalDocker()
      setView(next)
    } catch (e) {
      setError(errText(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!active) return
    void refresh()
    const id = window.setInterval(() => void refresh(), 15_000)
    return () => window.clearInterval(id)
  }, [active, refresh])

  const runAction = async (key: string, fn: () => Promise<{ docker?: GlobalDockerView } | null>) => {
    setBusyKey(key)
    try {
      const res = await fn()
      if (res?.docker) setView(res.docker)
      else await refresh()
    } catch (e) {
      setError(errText(e))
      await refresh()
    } finally {
      setBusyKey(null)
    }
  }

  const withDocker = async (fn: () => Promise<unknown>) => {
    await fn()
    return { docker: await AppService.ListGlobalDocker() }
  }

  const projects = view?.projects ?? []
  const busy = busyKey !== null

  return (
    <div className="flex h-full min-h-0 flex-col rounded-xl border bg-card">
      <div className="flex shrink-0 items-center gap-2 border-b px-4 py-3">
        <Container className="size-4 text-muted-foreground" />
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-medium">Docker</h2>
          <p className="truncate text-xs text-muted-foreground">
            {view?.summary
              ? `${view.running}/${view.total} · ${view.summary}`
              : loading
                ? "Carregando…"
                : "Containers do daemon"}
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void refresh()}
          disabled={loading || busy}
          title="Atualizar lista"
        >
          {loading ? <Loader2 className="animate-spin" /> : <RefreshCw />}
          Refresh
        </Button>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-3 p-3">
          {!view?.available && (
            <Alert>
              <AlertDescription>
                {view?.error || "Docker CLI não encontrado no PATH."}
              </AlertDescription>
            </Alert>
          )}

          {view?.available && !view.daemonRunning && (
            <Alert>
              <AlertDescription>
                {view.error || "Docker daemon não está rodando."}
              </AlertDescription>
            </Alert>
          )}

          {view?.available && view.daemonRunning && projects.length === 0 && !loading && (
            <p className="py-8 text-center text-sm text-muted-foreground">
              Nenhum container no daemon.
            </p>
          )}

          {projects.map((proj) => {
            const key = projectKey(proj)
            const containers = proj.containers ?? []
            const compose = proj.composeFile ?? ""

            return (
              <ProjectBlock
                key={`${proj.name}:${key}`}
                proj={proj}
                busyKey={busyKey}
                onProjectStart={() =>
                  void runAction(`start:${key}`, () =>
                    withDocker(async () => {
                      for (const c of containers) {
                        if (!isRunning(c.state)) await AppService.GlobalDockerStart(c.id)
                      }
                    }),
                  )
                }
                onProjectStop={() =>
                  void runAction(`stop:${key}`, () =>
                    withDocker(async () => {
                      for (const c of containers) {
                        if (isRunning(c.state)) await AppService.GlobalDockerStop(c.id)
                      }
                    }),
                  )
                }
                onProjectUp={() => {
                  if (!compose) return
                  void runAction(`up:${key}`, () => AppService.GlobalDockerUp(compose, false))
                }}
                onProjectDown={() => {
                  if (!compose) return
                  void runAction(`down:${key}`, () => AppService.GlobalDockerDown(compose))
                }}
                onProjectRecreate={() =>
                  void runAction(`recreate:${key}`, () =>
                    withDocker(async () => {
                      for (const c of containers) {
                        if (c.canCompose && c.service) {
                          await AppService.GlobalDockerRecreate(c.id)
                        }
                      }
                    }),
                  )
                }
                onStart={(c) =>
                  void runAction(`start:${c.id}`, () => AppService.GlobalDockerStart(c.id))
                }
                onStop={(c) =>
                  void runAction(`stop:${c.id}`, () => AppService.GlobalDockerStop(c.id))
                }
                onUp={(c) => {
                  const file = c.composeFile || compose
                  if (!file) return
                  void runAction(`up:${c.id}`, () => AppService.GlobalDockerUp(file, false))
                }}
                onDown={(c) => {
                  const file = c.composeFile || compose
                  if (!file) return
                  void runAction(`down:${c.id}`, () => AppService.GlobalDockerDown(file))
                }}
                onRecreate={(c) =>
                  void runAction(`recreate:${c.id}`, () => AppService.GlobalDockerRecreate(c.id))
                }
              />
            )
          })}
        </div>
      </ScrollArea>

      <AlertDialog
        open={error !== null}
        onOpenChange={(open) => {
          if (!open) setError(null)
        }}
      >
        <AlertDialogContent className="flex max-h-[min(90vh,42rem)] max-w-[calc(100%-2rem)] flex-col gap-3 data-[size=default]:max-w-[calc(100%-2rem)] data-[size=default]:sm:max-w-2xl">
          <AlertDialogHeader className="shrink-0 !place-items-center !text-center sm:!place-items-center sm:!text-center">
            <AlertDialogMedia className="bg-destructive/10 text-destructive dark:bg-destructive/20">
              <CircleAlert />
            </AlertDialogMedia>
            <AlertDialogTitle className="w-full text-center sm:col-start-1!">
              Erro Docker
            </AlertDialogTitle>
            <AlertDialogDescription className="sr-only">
              Ocorreu um erro ao executar a ação Docker.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto rounded-lg border bg-muted/40 p-3">
            <pre className="whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-foreground">
              {error}
            </pre>
          </div>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function ProjectBlock({
  proj,
  busyKey,
  onProjectStart,
  onProjectStop,
  onProjectUp,
  onProjectDown,
  onProjectRecreate,
  onUp,
  onDown,
  onStart,
  onStop,
  onRecreate,
}: {
  proj: GlobalDockerProject
  busyKey: string | null
  onProjectStart: () => void
  onProjectStop: () => void
  onProjectUp: () => void
  onProjectDown: () => void
  onProjectRecreate: () => void
  onUp: (c: GlobalDockerContainer) => void
  onDown: (c: GlobalDockerContainer) => void
  onStart: (c: GlobalDockerContainer) => void
  onStop: (c: GlobalDockerContainer) => void
  onRecreate: (c: GlobalDockerContainer) => void
}) {
  const title = projectTitle(proj)
  const key = projectKey(proj)
  const composeFile = proj.composeFile ?? ""
  const containers = proj.containers ?? []
  const hasRunning = proj.running > 0
  const [open, setOpen] = useState(hasRunning)
  const [userToggled, setUserToggled] = useState(false)

  useEffect(() => {
    if (userToggled) return
    setOpen(hasRunning)
  }, [hasRunning, userToggled])

  const anyBusy = busyKey !== null
  const projectBusy =
    busyKey === `start:${key}` ||
    busyKey === `stop:${key}` ||
    busyKey === `up:${key}` ||
    busyKey === `down:${key}` ||
    busyKey === `recreate:${key}`
  const allRunning = containers.length > 0 && containers.every((c) => isRunning(c.state))
  const noneRunning = containers.every((c) => !isRunning(c.state))
  const canCompose = !!(proj.canCompose && composeFile)
  const canRecreate = containers.some((c) => c.canCompose && c.service)

  return (
    <div className="overflow-hidden rounded-lg border">
      <div
        className={`flex h-9 items-center gap-1 bg-muted/40 px-1.5 ${open ? "border-b" : ""}`}
      >
        <button
          type="button"
          className="flex min-w-0 flex-1 items-center gap-1 rounded-md px-1 py-1 text-left hover:bg-muted/80"
          onClick={() => {
            setUserToggled(true)
            setOpen((v) => !v)
          }}
          aria-expanded={open}
          title={open ? "Recolher containers" : "Expandir containers"}
        >
          <ChevronRight
            className={`size-3.5 shrink-0 text-muted-foreground transition-transform ${open ? "rotate-90" : ""}`}
          />
          {hasRunning && (
            <span
              className="size-1.5 shrink-0 rounded-full bg-emerald-500"
              title="Há containers running"
              aria-hidden
            />
          )}
          <span
            className="min-w-0 flex-1 truncate text-xs font-medium"
            title={proj.workingDir || title}
          >
            {title}
            <span className="ml-2 font-normal text-muted-foreground">
              {proj.running}/{proj.total}
              {composeFile ? ` · ${composeFile.split(/[/\\]/).pop()}` : ""}
            </span>
          </span>
        </button>
        <div
          className="flex shrink-0 items-center"
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        >
          <ActionButtons
            busy={anyBusy}
            spinning={projectBusy}
            spinKind={
              busyKey === `start:${key}`
                ? "start"
                : busyKey === `stop:${key}`
                  ? "stop"
                  : busyKey === `up:${key}` ||
                      busyKey === `down:${key}` ||
                      busyKey === `recreate:${key}`
                    ? "more"
                    : null
            }
            canStart={!allRunning}
            canStop={!noneRunning}
            canUp={canCompose}
            canDown={canCompose}
            canRecreate={canRecreate}
            onStart={onProjectStart}
            onStop={onProjectStop}
            onUp={onProjectUp}
            onDown={onProjectDown}
            onRecreate={onProjectRecreate}
          />
        </div>
      </div>

      {open && (
        <ul className="divide-y">
          {containers.map((c) => (
            <ContainerRow
              key={c.id}
              container={c}
              busyKey={busyKey}
              onStart={() => onStart(c)}
              onStop={() => onStop(c)}
              onUp={() => onUp(c)}
              onDown={() => onDown(c)}
              onRecreate={() => onRecreate(c)}
            />
          ))}
          {containers.length === 0 && (
            <li className="px-3 py-2 text-xs text-muted-foreground">Nenhum container.</li>
          )}
        </ul>
      )}
    </div>
  )
}

function ContainerRow({
  container: c,
  busyKey,
  onStart,
  onStop,
  onUp,
  onDown,
  onRecreate,
}: {
  container: GlobalDockerContainer
  busyKey: string | null
  onStart: () => void
  onStop: () => void
  onUp: () => void
  onDown: () => void
  onRecreate: () => void
}) {
  const rowBusy =
    busyKey === `start:${c.id}` ||
    busyKey === `stop:${c.id}` ||
    busyKey === `recreate:${c.id}` ||
    busyKey === `up:${c.id}` ||
    busyKey === `down:${c.id}`
  const running = isRunning(c.state)
  const canCompose = !!(c.canCompose && (c.composeFile || c.service))
  const canRecreate = !!(c.canCompose && c.service)
  const meta = [c.ports, c.image].filter(Boolean).join(" · ") || c.status || c.id.slice(0, 12)
  const anyBusy = busyKey !== null

  return (
    <li className="flex h-9 items-center gap-2 px-2.5">
      <span className="w-[7.5rem] shrink-0 truncate font-mono text-xs" title={c.name}>
        {c.service || c.name}
      </span>
      <Badge variant={stateVariant(c.state)} className="h-5 shrink-0 px-1.5 text-[10px]">
        {c.state || "unknown"}
      </Badge>
      <span className="min-w-0 flex-1 truncate text-[11px] text-muted-foreground" title={meta}>
        {meta}
      </span>
      <ActionButtons
        busy={anyBusy}
        spinning={rowBusy}
        spinKind={
          busyKey === `start:${c.id}`
            ? "start"
            : busyKey === `stop:${c.id}`
              ? "stop"
              : busyKey === `up:${c.id}` || busyKey === `down:${c.id}` || busyKey === `recreate:${c.id}`
                ? "more"
                : null
        }
        canStart={!running}
        canStop={running}
        canUp={canCompose}
        canDown={canCompose}
        canRecreate={canRecreate}
        onStart={onStart}
        onStop={onStop}
        onUp={onUp}
        onDown={onDown}
        onRecreate={onRecreate}
      />
    </li>
  )
}

function ActionButtons({
  busy,
  spinning,
  spinKind,
  canStart,
  canStop,
  canUp,
  canDown,
  canRecreate,
  onStart,
  onStop,
  onUp,
  onDown,
  onRecreate,
}: {
  busy: boolean
  spinning: boolean
  spinKind: "start" | "stop" | "more" | null
  canStart: boolean
  canStop: boolean
  canUp: boolean
  canDown: boolean
  canRecreate: boolean
  onStart: () => void
  onStop: () => void
  onUp: () => void
  onDown: () => void
  onRecreate: () => void
}) {
  return (
    <div className="flex shrink-0 items-center gap-0.5">
      <Button
        variant="ghost"
        size="icon-xs"
        title="Start"
        disabled={busy || !canStart}
        onClick={onStart}
      >
        {spinning && spinKind === "start" ? <Loader2 className="animate-spin" /> : <Play />}
      </Button>
      <Button
        variant="ghost"
        size="icon-xs"
        title="Stop"
        disabled={busy || !canStop}
        onClick={onStop}
      >
        {spinning && spinKind === "stop" ? <Loader2 className="animate-spin" /> : <Square />}
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="icon-xs"
              disabled={busy}
              aria-label="Mais ações"
              title="Mais ações"
            />
          }
        >
          {spinning && spinKind === "more" ? (
            <Loader2 className="animate-spin" />
          ) : (
            <EllipsisVertical />
          )}
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="min-w-40">
          <DropdownMenuItem disabled={!canUp || busy} onClick={onUp}>
            <Play />
            Up
          </DropdownMenuItem>
          <DropdownMenuItem disabled={!canDown || busy} onClick={onDown}>
            <Trash2 />
            Down
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem disabled={!canRecreate || busy} onClick={onRecreate}>
            <RefreshCw />
            Recreate
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
