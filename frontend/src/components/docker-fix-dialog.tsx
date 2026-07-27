import { useEffect, useMemo, useState } from "react"
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
import { Checkbox } from "@/components/ui/checkbox"
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

/** Mirrors desktop.DockerFixPlanView until bindings are regenerated. */
export type DockerFixStepView = {
  id: string
  kind: string
  title: string
  target: string
  risk: string
  command?: string
  status?: string
  detail?: string
  enabled: boolean
}

export type DockerFixFileView = {
  path: string
  reason: string
  bytes: number
  preview: string
  enabled: boolean
}

export type DockerFixPlanView = {
  problem: string
  resolution: string
  steps: DockerFixStepView[]
  files: DockerFixFileView[]
  notes?: string[]
  message?: string
  canFix: boolean
  action?: string
}

function stepIcon(status: string | undefined) {
  switch (status) {
    case "running":
      return <Loader2 className="size-3.5 animate-spin text-sky-600 dark:text-sky-400" />
    case "ok":
      return <CheckCircle2 className="size-3.5 text-emerald-600 dark:text-emerald-400" />
    case "error":
      return <XCircle className="size-3.5 text-destructive" />
    case "skipped":
      return <Circle className="size-3.5 text-muted-foreground" />
    default:
      return <Circle className="size-3.5 text-muted-foreground/70" />
  }
}

function riskBadge(risk: string) {
  if (risk === "destructive") return <Badge variant="destructive">destrutivo</Badge>
  if (risk === "warn") return <Badge variant="secondary">atenção</Badge>
  return null
}

export function DockerFixDialog({
  open,
  onOpenChange,
  plan,
  loadingPlan,
  planError,
  running,
  liveSteps,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  plan: DockerFixPlanView | null
  loadingPlan: boolean
  planError?: string | null
  running: boolean
  liveSteps: DockerFixStepView[]
  onConfirm: (opts: { enabledStepIDs: string[]; enabledFilePaths: string[] }) => void
}) {
  const [enabledSteps, setEnabledSteps] = useState<Record<string, boolean>>({})
  const [enabledFiles, setEnabledFiles] = useState<Record<string, boolean>>({})
  const [hydrated, setHydrated] = useState(false)

  useEffect(() => {
    if (!open) {
      setHydrated(false)
      return
    }
    if (!plan || hydrated) return
    const steps: Record<string, boolean> = {}
    for (const s of plan.steps ?? []) {
      steps[s.id] = s.enabled !== false
    }
    const files: Record<string, boolean> = {}
    for (const f of plan.files ?? []) {
      files[f.path] = f.enabled !== false
    }
    setEnabledSteps(steps)
    setEnabledFiles(files)
    setHydrated(true)
  }, [open, plan, hydrated])

  const timeline = useMemo(() => {
    const base = plan?.steps ?? []
    if (liveSteps.length === 0) return base
    const byId = new Map(liveSteps.map((s) => [s.id, s]))
    return base.map((s) => {
      const live = byId.get(s.id)
      if (!live) return s
      return { ...s, ...live, risk: live.risk || s.risk, title: live.title || s.title }
    })
  }, [plan, liveSteps])

  const failed = timeline.find((s) => s.status === "error")
  const allOk =
    !running &&
    liveSteps.length > 0 &&
    timeline.filter((s) => enabledSteps[s.id]).every((s) => s.status === "ok") &&
    !failed

  const enabledStepIDs = Object.entries(enabledSteps)
    .filter(([, on]) => on)
    .map(([id]) => id)
  const enabledFilePaths = Object.entries(enabledFiles)
    .filter(([, on]) => on)
    .map(([path]) => path)

  const hasSelection = enabledStepIDs.length > 0 || enabledFilePaths.length > 0
  const canConfirm =
    !!plan?.canFix && !running && !allOk && !loadingPlan && hasSelection && !planError

  const errMsg = planError || (!plan?.canFix && plan?.message ? plan.message : null)

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && running) return
        onOpenChange(next)
      }}
    >
      <DialogContent className="flex max-h-[85vh] w-[min(40rem,calc(100%-2rem))] max-w-none flex-col gap-0 overflow-hidden p-0 sm:max-w-none">
        <DialogHeader className="shrink-0 space-y-1.5 border-b px-4 py-3 text-left">
          <DialogTitle className="flex items-center gap-2 text-base">
            <Sparkles className="size-4 text-muted-foreground" />
            Corrigir com IA
            {loadingPlan ? (
              <Badge variant="outline" className="ml-1 gap-1 font-normal">
                <Loader2 className="size-3 animate-spin" />
                analisando
              </Badge>
            ) : running ? (
              <Badge variant="outline" className="ml-1 gap-1 font-normal">
                <Loader2 className="size-3 animate-spin" />
                executando
              </Badge>
            ) : allOk ? (
              <Badge
                variant="outline"
                className="ml-1 border-emerald-500/40 font-normal text-emerald-700 dark:text-emerald-400"
              >
                concluído
              </Badge>
            ) : failed ? (
              <Badge variant="destructive" className="ml-1 font-normal">
                falhou
              </Badge>
            ) : null}
          </DialogTitle>
          <DialogDescription className="text-xs">
            Problema e resolução sugeridos a partir do erro Docker. Desmarque o que não quiser
            executar.
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="min-h-0 flex-1">
          <div className="flex flex-col gap-4 px-4 py-3">
            {loadingPlan ? (
              <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
                A IA está analisando o erro e montando o plano…
              </div>
            ) : errMsg && !plan?.canFix ? (
              <p className="rounded-lg border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {errMsg}
              </p>
            ) : plan ? (
              <>
                <section className="space-y-1.5">
                  <h3 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    Problema
                  </h3>
                  <p className="text-sm leading-relaxed">{plan.problem}</p>
                </section>
                <section className="space-y-1.5">
                  <h3 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    Resolução
                  </h3>
                  <p className="text-sm leading-relaxed">{plan.resolution}</p>
                </section>

                {(plan.steps?.length ?? 0) > 0 ? (
                  <section className="space-y-2">
                    <h3 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      Passos
                    </h3>
                    <ul className="space-y-1.5 rounded-lg border p-2">
                      {timeline.map((s) => (
                        <li
                          key={s.id}
                          className="flex items-start gap-2 rounded-md px-1.5 py-1 text-sm"
                        >
                          {!running && !allOk ? (
                            <Checkbox
                              className="mt-0.5"
                              checked={!!enabledSteps[s.id]}
                              disabled={running}
                              onCheckedChange={(v) =>
                                setEnabledSteps((prev) => ({ ...prev, [s.id]: !!v }))
                              }
                            />
                          ) : (
                            <span className="mt-0.5">{stepIcon(s.status)}</span>
                          )}
                          <div className="min-w-0 flex-1">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="font-medium">{s.title}</span>
                              {riskBadge(s.risk)}
                            </div>
                            {s.command ? (
                              <p className="mt-0.5 font-mono text-[11px] text-muted-foreground">
                                {s.command}
                              </p>
                            ) : null}
                            {s.detail ? (
                              <p
                                className={cn(
                                  "mt-0.5 truncate font-mono text-[11px] text-muted-foreground",
                                  s.status === "error" && "text-destructive",
                                )}
                                title={s.detail}
                              >
                                {s.detail}
                              </p>
                            ) : null}
                          </div>
                        </li>
                      ))}
                    </ul>
                  </section>
                ) : null}

                {(plan.files?.length ?? 0) > 0 ? (
                  <section className="space-y-2">
                    <h3 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      Arquivos
                    </h3>
                    <ul className="space-y-2 rounded-lg border p-2">
                      {plan.files.map((f) => (
                        <li key={f.path} className="space-y-1 px-1.5 py-1 text-sm">
                          <div className="flex items-start gap-2">
                            <Checkbox
                              className="mt-0.5"
                              checked={!!enabledFiles[f.path]}
                              disabled={running || allOk}
                              onCheckedChange={(v) =>
                                setEnabledFiles((prev) => ({ ...prev, [f.path]: !!v }))
                              }
                            />
                            <div className="min-w-0 flex-1">
                              <div className="flex flex-wrap items-center gap-2">
                                <span className="font-mono text-xs font-medium">{f.path}</span>
                                <span className="text-[11px] text-muted-foreground">
                                  {f.bytes} bytes
                                </span>
                              </div>
                              <p className="mt-0.5 text-xs text-muted-foreground">{f.reason}</p>
                              {f.preview ? (
                                <pre className="mt-1 max-h-24 overflow-auto rounded-md bg-muted/40 p-2 font-mono text-[10px] leading-relaxed">
                                  {f.preview}
                                </pre>
                              ) : null}
                            </div>
                          </div>
                        </li>
                      ))}
                    </ul>
                  </section>
                ) : null}

                {(plan.notes?.length ?? 0) > 0 ? (
                  <section className="space-y-1.5">
                    <h3 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      Observações
                    </h3>
                    <ul className="list-inside list-disc text-xs text-muted-foreground">
                      {plan.notes!.map((n, i) => (
                        <li key={i}>{n}</li>
                      ))}
                    </ul>
                  </section>
                ) : null}
              </>
            ) : null}
          </div>
        </ScrollArea>

        <DialogFooter className="mx-0 mb-0 shrink-0 rounded-none border-t bg-transparent px-4 py-3 sm:justify-end">
          {canConfirm ? (
            <Button
              onClick={() =>
                onConfirm({
                  enabledStepIDs,
                  enabledFilePaths,
                })
              }
            >
              <Container className="size-4" />
              Executar correção
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
