import { useEffect, useRef } from "react"
import {
  CheckCircle2,
  Circle,
  Container,
  Loader2,
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
import { ScrollArea } from "@/components/ui/scroll-area"
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
}: {
  state: DockerActionDialogState
  onOpenChange: (open: boolean) => void
}) {
  const logEndRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth", block: "end" })
  }, [state.lines.length])

  const title = state.action ? `docker ${state.action}` : "Docker"
  const canClose = !state.running

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

        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden px-4 py-3">
          {state.services.length > 0 ? (
            <ul className="shrink-0 space-y-1.5 rounded-lg border p-2">
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
                          "mt-0.5 truncate font-mono text-[11px] text-muted-foreground",
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

          <ScrollArea className="min-h-[12rem] flex-1 rounded-lg border bg-muted/30">
            <pre className="whitespace-pre-wrap break-words p-3 font-mono text-[11px] leading-relaxed text-foreground/90">
              {state.lines.length === 0
                ? state.running
                  ? "Aguardando output do Docker Compose…"
                  : "Sem output."
                : state.lines.join("\n")}
              <div ref={logEndRef} />
            </pre>
          </ScrollArea>
        </div>

        <DialogFooter className="shrink-0 border-t px-4 py-3">
          <Button
            variant="outline"
            disabled={!canClose}
            onClick={() => onOpenChange(false)}
          >
            Fechar
          </Button>
        </DialogFooter>
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
