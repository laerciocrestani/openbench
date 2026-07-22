import { useEffect, useMemo, useState } from "react"
import {
  AlertTriangle,
  CheckCircle2,
  Circle,
  Loader2,
  Stethoscope,
  XCircle,
} from "lucide-react"

import type {
  DoctorFixPlanView,
  DoctorFixStepView,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

function stepIcon(status: string | undefined) {
  switch (status) {
    case "running":
      return <Loader2 className="size-3.5 animate-spin text-sky-600 dark:text-sky-400" />
    case "ok":
      return <CheckCircle2 className="size-3.5 text-emerald-600 dark:text-emerald-400" />
    case "error":
    case "manual":
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

export function DoctorFixDialog({
  open,
  onOpenChange,
  plan,
  loadingPlan,
  running,
  liveSteps,
  onConfirm,
  onReplan,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  plan: DoctorFixPlanView | null
  loadingPlan: boolean
  running: boolean
  liveSteps: DoctorFixStepView[]
  onConfirm: (opts: {
    newBranch: string
    baseAction: string
    confirmDestructive: boolean
  }) => void
  onReplan: (opts: { newBranch: string; baseAction: string }) => void
}) {
  const [newBranch, setNewBranch] = useState("")
  const [baseAction, setBaseAction] = useState("")
  const [confirmDestructive, setConfirmDestructive] = useState(false)
  const [hydrated, setHydrated] = useState(false)

  useEffect(() => {
    if (!open) {
      setHydrated(false)
      return
    }
    if (!plan || hydrated) return
    setNewBranch(plan.suggestedBranch || "")
    setBaseAction(plan.suggestedBaseAction || plan.baseActionOptions?.[0] || "")
    setConfirmDestructive(false)
    setHydrated(true)
  }, [open, plan, hydrated])

  const timeline = useMemo(() => {
    const base = plan?.steps ?? []
    if (liveSteps.length === 0) return base
    const byId = new Map(liveSteps.map((s) => [s.id, s]))
    return base.map((s) => {
      const live = byId.get(s.id)
      if (!live) return s
      return {
        ...s,
        ...live,
        risk: live.risk || s.risk,
        title: live.title || s.title,
        command: live.command || s.command,
      }
    })
  }, [plan, liveSteps])

  const failed = timeline.find((s) => s.status === "error")
  const allOk =
    !running &&
    timeline.length > 0 &&
    timeline.every((s) => s.status === "ok") &&
    !failed
  const canConfirm =
    !!plan?.canAutoFix &&
    !running &&
    !allOk &&
    (!plan.needsBranchName || newBranch.trim().length > 0) &&
    (!plan.needsDestructiveConfirm || confirmDestructive)

  const startFix = () => {
    if (!canConfirm) return
    onConfirm({
      newBranch: newBranch.trim(),
      baseAction,
      confirmDestructive,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] w-[min(48rem,calc(100%-2rem))] max-w-none flex-col gap-0 overflow-hidden p-0 sm:max-w-none">
        <DialogHeader className="shrink-0 space-y-2 border-b px-4 py-3 text-left">
          <DialogTitle className="flex items-center gap-2 text-base">
            <Stethoscope className="size-4 text-muted-foreground" />
            Ajustar com o Doctor
          </DialogTitle>
          <DialogDescription className="text-xs">
            {plan
              ? `${plan.summary || "Plano de correção"} · ${plan.branch || "—"} → base ${plan.base || "—"}`
              : "Montando plano…"}
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden">
          <div className="space-y-4 px-4 py-3">
            {loadingPlan && !plan ? (
              <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
                Calculando passos…
              </div>
            ) : null}

            {plan && !plan.canAutoFix ? (
              <p className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {plan.blockReason || "Ajuste automático indisponível."}
              </p>
            ) : null}

            {plan?.needsBranchName ? (
              <div className="space-y-1.5">
                <Label htmlFor="doctor-fix-branch">Nova feature branch</Label>
                <Input
                  id="doctor-fix-branch"
                  value={newBranch}
                  onChange={(e) => setNewBranch(e.target.value)}
                  onBlur={() => {
                    if (!running) onReplan({ newBranch: newBranch.trim(), baseAction })
                  }}
                  disabled={running}
                  placeholder="feature/minha-alteracao"
                />
              </div>
            ) : null}

            {plan?.needsBaseAction && (plan.baseActionOptions?.length ?? 0) > 0 ? (
              <div className="space-y-1.5">
                <Label>Ação na base ({plan.base})</Label>
                <div className="flex flex-wrap gap-1.5">
                  {plan.baseActionOptions!.map((opt) => (
                    <Button
                      key={opt}
                      size="sm"
                      type="button"
                      variant={baseAction === opt ? "default" : "outline"}
                      disabled={running}
                      onClick={() => {
                        setBaseAction(opt)
                        setConfirmDestructive(false)
                        onReplan({ newBranch: newBranch.trim(), baseAction: opt })
                      }}
                    >
                      {opt}
                    </Button>
                  ))}
                </div>
              </div>
            ) : null}

            {plan?.needsDestructiveConfirm ? (
              <label className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm">
                <Checkbox
                  checked={confirmDestructive}
                  disabled={running}
                  onCheckedChange={(v) => setConfirmDestructive(v === true)}
                  className="mt-0.5"
                />
                <span>
                  Confirmo o reset destrutivo da base <code className="text-xs">{plan.base}</code>{" "}
                  (descarta commits locais da base).
                </span>
              </label>
            ) : null}

            {(plan?.warnings?.length ?? 0) > 0 && (
              <ul className="space-y-1 text-xs text-amber-700 dark:text-amber-400">
                {plan!.warnings!.map((w) => (
                  <li key={w} className="flex gap-1.5">
                    <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                    <span>{w}</span>
                  </li>
                ))}
              </ul>
            )}

            <section className="space-y-2">
              <div className="flex items-center justify-between gap-2">
                <h3 className="text-xs font-medium text-muted-foreground">Timeline de comandos</h3>
                {running ? (
                  <span className="flex items-center gap-1.5 text-xs text-sky-700 dark:text-sky-300">
                    <Loader2 className="size-3 animate-spin" />
                    Executando…
                  </span>
                ) : null}
                {allOk ? (
                  <span className="flex items-center gap-1.5 text-xs text-emerald-700 dark:text-emerald-300">
                    <CheckCircle2 className="size-3" />
                    Concluído
                  </span>
                ) : null}
              </div>
              <ol className="space-y-2">
                {timeline.map((step, idx) => {
                  const isRunning = step.status === "running"
                  return (
                    <li
                      key={step.id || idx}
                      className={cn(
                        "rounded-lg border px-3 py-2 transition-colors",
                        isRunning && "border-sky-500/50 bg-sky-500/10 ring-1 ring-sky-500/20",
                        step.status === "error" && "border-destructive/40 bg-destructive/5",
                        step.status === "ok" && "border-emerald-500/25 bg-emerald-500/5",
                      )}
                    >
                      <div className="flex items-start gap-2">
                        <span className="mt-0.5">{stepIcon(step.status)}</span>
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <p className="text-sm font-medium">
                              {idx + 1}. {step.title}
                            </p>
                            {isRunning ? (
                              <Badge variant="outline" className="text-[10px] text-sky-700 dark:text-sky-300">
                                executando
                              </Badge>
                            ) : null}
                            {step.status === "ok" ? (
                              <Badge variant="outline" className="text-[10px] text-emerald-700 dark:text-emerald-300">
                                ok
                              </Badge>
                            ) : null}
                            {riskBadge(step.risk)}
                          </div>
                          <pre className="mt-1 overflow-x-auto rounded bg-muted/40 px-2 py-1 font-mono text-[11px] text-muted-foreground">
                            {step.command}
                          </pre>
                          {step.detail ? (
                            <p className="mt-1 text-xs text-destructive">{step.detail}</p>
                          ) : null}
                          {step.manualHint ? (
                            <p className="mt-1 text-xs text-amber-700 dark:text-amber-400">
                              Manual: {step.manualHint}
                            </p>
                          ) : null}
                        </div>
                      </div>
                    </li>
                  )
                })}
              </ol>
            </section>

            {failed ? (
              <p className="text-sm text-muted-foreground">
                O Doctor parou neste passo. Siga a orientação manual e rode o Doctor de novo depois.
              </p>
            ) : null}
            {allOk ? (
              <p className="text-sm text-emerald-700 dark:text-emerald-300">
                Ajuste concluído. Pode fechar e continuar na nova branch.
              </p>
            ) : null}
          </div>
        </div>

        <DialogFooter className="mx-0 mb-0 shrink-0 gap-2 rounded-none border-t px-4 py-3 sm:justify-between">
          <Button size="sm" variant="secondary" onClick={() => onOpenChange(false)} disabled={running}>
            {failed || allOk ? "Fechar" : "Cancelar"}
          </Button>
          <Button
            type="button"
            size="sm"
            disabled={!canConfirm}
            onPointerDown={(e) => {
              // Avoid input-blur → replan → disabled button swallowing the click.
              e.preventDefault()
              startFix()
            }}
          >
            {running ? <Loader2 className="animate-spin" /> : null}
            {running ? "Executando…" : allOk ? "Concluído" : "Confirmar e executar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
