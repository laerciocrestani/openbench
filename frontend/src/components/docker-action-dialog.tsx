import { useEffect, useRef } from "react"
import {
  CheckCircle2,
  Circle,
  Container,
  Loader2,
  Sparkles,
  XCircle,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { cn } from "@/lib/utils"

export type DockerServiceProgress = {
  name: string
  status: "pending" | "running" | "ok" | "error"
  detail?: string
}

export type DockerActionDialogState = {
  open: boolean
  action: string
  running: boolean
  ok: boolean | null
  message: string
  lines: string[]
  services: DockerServiceProgress[]
}

function statusIcon(status: DockerServiceProgress["status"]) {
  switch (status) {
    case "running":
      return <Loader2 className="size-3.5 animate-spin text-sky-600 dark:text-sky-400" />
    case "ok":
      return <CheckCircle2 className="size-3.5 text-emerald-600 dark:text-emerald-400" />
    case "error":
      return <XCircle className="size-3.5 text-destructive" />
    default:
      return <Circle className="size-3.5 text-muted-foreground/70" />
  }
}

function statusLabel(status: DockerServiceProgress["status"]) {
  switch (status) {
    case "running":
      return "em andamento"
    case "ok":
      return "ok"
    case "error":
      return "erro"
    default:
      return "pendente"
  }
}

export function DockerActionDialog({
  state,
  onOpenChange,
  onFixWithAI,
}: {
  state: DockerActionDialogState
  onOpenChange: (open: boolean) => void
  onFixWithAI?: () => void
}) {
  const logEndRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth", block: "end" })
  }, [state.lines.length])

  const title = state.action ? `docker ${state.action}` : "Docker"
  const failed = !state.running && state.ok === false
  const showFix = failed && !!onFixWithAI

  return (
    <Dialog
      open={state.open}
      onOpenChange={(next) => {
        if (!next && state.running) return
        onOpenChange(next)
      }}
    >
      <DialogContent className="flex max-h-[85vh] w-[min(40rem,calc(100%-2rem))] max-w-none flex-col gap-0 overflow-hidden p-0 sm:max-w-none">
        <DialogHeader className="shrink-0 space-y-1.5 border-b px-4 py-3 text-left">
          <DialogTitle className="flex items-center gap-2 text-base">
            <Container className="size-4 text-muted-foreground" />
            {title}
            {state.running ? (
              <Badge variant="outline" className="ml-1 gap-1 font-normal">
                <Loader2 className="size-3 animate-spin" />
                executando
              </Badge>
            ) : state.ok === true ? (
              <Badge variant="outline" className="ml-1 border-emerald-500/40 font-normal text-emerald-700 dark:text-emerald-400">
                concluído
              </Badge>
            ) : state.ok === false ? (
              <Badge variant="destructive" className="ml-1 font-normal">
                falhou
              </Badge>
            ) : null}
          </DialogTitle>
          <DialogDescription className="text-xs">
            {state.message ||
              (state.running
                ? "Acompanhe o status de cada serviço e o log do Compose."
                : "Resultado da ação Docker.")}
          </DialogDescription>
        </DialogHeader>

        <div
          className={cn(
            "min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-3",
            showFix ? "max-h-[calc(85vh-8.5rem)]" : "max-h-[calc(85vh-5.5rem)]",
          )}
        >
          <div className="flex flex-col gap-3">
            {state.services.length > 0 ? (
              <ul className="space-y-1.5 rounded-lg border p-2">
                {state.services.map((svc) => (
                  <li
                    key={svc.name}
                    className="flex items-start gap-2 rounded-md px-1.5 py-1 text-sm"
                  >
                    <span className="mt-0.5">{statusIcon(svc.status)}</span>
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-mono text-xs font-medium">{svc.name}</span>
                        <span className="text-[11px] text-muted-foreground">
                          {statusLabel(svc.status)}
                        </span>
                      </div>
                      {svc.detail ? (
                        <p
                          className={cn(
                            "mt-0.5 break-words font-mono text-[11px] text-muted-foreground",
                            svc.status === "error" && "text-destructive",
                          )}
                          title={svc.detail}
                        >
                          {svc.detail}
                        </p>
                      ) : null}
                    </div>
                  </li>
                ))}
              </ul>
            ) : null}

            <pre className="whitespace-pre-wrap break-words rounded-lg border bg-muted/30 p-3 font-mono text-[11px] leading-relaxed text-foreground/90">
              {state.lines.length === 0
                ? state.running
                  ? "Aguardando output do Docker Compose…"
                  : "Sem output."
                : state.lines.join("\n")}
              <div ref={logEndRef} />
            </pre>
          </div>
        </div>

        {showFix ? (
          <DialogFooter className="mx-0 mb-0 shrink-0 rounded-none border-t bg-transparent px-4 py-3 sm:justify-end">
            <Button onClick={() => onFixWithAI?.()}>
              <Sparkles className="size-4" />
              Corrigir com IA
            </Button>
          </DialogFooter>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

export function emptyDockerActionDialog(): DockerActionDialogState {
  return {
    open: false,
    action: "",
    running: false,
    ok: null,
    message: "",
    lines: [],
    services: [],
  }
}
