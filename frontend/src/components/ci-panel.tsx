import { useCallback, useEffect, useMemo, useState } from "react"

import { AppService } from "../../bindings/github.com/laerciocrestani/openbench"
import type {
  CIDispatchInputView,
  CIDispatchPreviewView,
  CIFixPreviewView,
  CILogView,
  CIRerunPreviewView,
  CIRunDetailView,
  CIRunView,
  CIStatusView,
  CIUsageView,
  CIWorkflowView,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { cn } from "@/lib/utils"
import {
  CircleAlert,
  ExternalLink,
  FileText,
  Loader2,
  Play,
  RefreshCw,
  RotateCcw,
  Workflow,
  Wrench,
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

function shortSha(sha: string | undefined): string {
  if (!sha) return "—"
  return sha.length > 7 ? sha.slice(0, 7) : sha
}

function conclusionVariant(
  status: string,
  conclusion: string,
  failed: boolean,
): "default" | "secondary" | "outline" | "destructive" {
  if (failed) return "destructive"
  const c = (conclusion || status || "").toLowerCase()
  if (c === "success" || c === "completed") return "default"
  if (
    c === "in_progress" ||
    c === "queued" ||
    c === "pending" ||
    c === "waiting"
  ) {
    return "outline"
  }
  return "secondary"
}

function labelOf(run: CIRunView): string {
  return run.conclusion || run.status || "—"
}

function ActionsUsageBanner({
  usage,
}: {
  usage: CIUsageView | null | undefined
}) {
  if (!usage) return null
  const mins =
    usage.runMinutes ?? usage.windowMinutes ?? usage.orgUsedMinutes ?? null
  return (
    <Alert
      className={cn(
        "shrink-0",
        usage.state === "unavailable" && "border-destructive/40",
      )}
    >
      <CircleAlert className="size-4" />
      <AlertDescription className="text-xs leading-relaxed">
        <span className="font-medium">Actions · {usage.state}</span>
        {mins != null && (
          <span className="ml-2 font-mono tabular-nums">~{mins} min</span>
        )}
        {usage.orgUsedMinutes != null && usage.orgIncludedMinutes != null && (
          <span className="ml-2 font-mono tabular-nums">
            org {usage.orgUsedMinutes}/{usage.orgIncludedMinutes}
            {usage.orgRemainingMinutes != null &&
              ` · resta ${usage.orgRemainingMinutes}`}
          </span>
        )}
        {usage.message ? (
          <span className="mt-0.5 block text-muted-foreground">
            {usage.message}
          </span>
        ) : null}
      </AlertDescription>
    </Alert>
  )
}

function DispatchInputField({
  input,
  value,
  onChange,
  disabled,
}: {
  input: CIDispatchInputView
  value: string
  onChange: (value: string) => void
  disabled?: boolean
}) {
  const label = input.description?.trim() || input.id
  const type = (input.type || "string").toLowerCase()

  if (type === "boolean") {
    const checked = value === "true"
    return (
      <div className="flex items-center gap-2">
        <Checkbox
          id={`dispatch-${input.id}`}
          checked={checked}
          disabled={disabled}
          onCheckedChange={(v) => onChange(v === true ? "true" : "false")}
        />
        <Label htmlFor={`dispatch-${input.id}`} className="text-sm">
          {label}
          {input.required ? " *" : ""}
        </Label>
      </div>
    )
  }

  if (type === "choice" && (input.options?.length ?? 0) > 0) {
    return (
      <div className="flex flex-col gap-1.5">
        <Label htmlFor={`dispatch-${input.id}`} className="text-sm">
          {label}
          {input.required ? " *" : ""}
        </Label>
        <Select
          value={value || undefined}
          onValueChange={(v) => onChange(String(v ?? ""))}
          disabled={disabled}
        >
          <SelectTrigger id={`dispatch-${input.id}`} className="w-full">
            <SelectValue placeholder="Selecione…" />
          </SelectTrigger>
          <SelectContent>
            {input.options!.map((opt) => (
              <SelectItem key={opt} value={opt}>
                {opt}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className="font-mono text-[10px] text-muted-foreground">{input.id}</span>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={`dispatch-${input.id}`} className="text-sm">
        {label}
        {input.required ? " *" : ""}
      </Label>
      <Input
        id={`dispatch-${input.id}`}
        value={value}
        disabled={disabled}
        type={type === "number" ? "number" : "text"}
        placeholder={input.default || input.id}
        onChange={(e) => onChange(e.target.value)}
      />
      <span className="font-mono text-[10px] text-muted-foreground">{input.id}</span>
    </div>
  )
}

export function CIPanel({
  open,
  onOpenChange,
  projectPath,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectPath: string
}) {
  const [failedOnly, setFailedOnly] = useState(false)
  const [status, setStatus] = useState<CIStatusView | null>(null)
  const [detail, setDetail] = useState<CIRunDetailView | null>(null)
  const [log, setLog] = useState<CILogView | null>(null)
  const [workflows, setWorkflows] = useState<CIWorkflowView[]>([])
  const [loading, setLoading] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [logLoading, setLogLoading] = useState(false)
  const [mutateBusy, setMutateBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [rerunPreview, setRerunPreview] = useState<CIRerunPreviewView | null>(
    null,
  )
  const [dispatchPreview, setDispatchPreview] =
    useState<CIDispatchPreviewView | null>(null)
  const [dispatchFields, setDispatchFields] = useState<Record<string, string>>(
    {},
  )
  const [fixPreview, setFixPreview] = useState<CIFixPreviewView | null>(null)
  const [fixMessage, setFixMessage] = useState("")

  const dispatchInputs = dispatchPreview?.inputs ?? []
  const dispatchMissingRequired = useMemo(() => {
    return dispatchInputs
      .filter((inp) => inp.required && !(dispatchFields[inp.id] ?? "").trim())
      .map((inp) => inp.id)
  }, [dispatchInputs, dispatchFields])

  const setDispatchField = (id: string, value: string) => {
    setDispatchFields((prev) => ({ ...prev, [id]: value }))
  }

  const fieldsToPairs = (fields: Record<string, string>): string[] =>
    Object.entries(fields)
      .filter(([k, v]) => k.trim() && v.trim() !== "")
      .map(([k, v]) => `${k}=${v}`)

  const refresh = useCallback(async () => {
    if (!projectPath) return
    setLoading(true)
    setError(null)
    try {
      const [next, wfs] = await Promise.all([
        AppService.CIStatus(failedOnly, 20),
        AppService.ListCIWorkflows().catch(() => [] as CIWorkflowView[]),
      ])
      setStatus(next ?? null)
      setWorkflows(wfs ?? [])
      setDetail(null)
      setLog(null)
    } catch (e) {
      setError(errText(e))
      setStatus(null)
    } finally {
      setLoading(false)
    }
  }, [failedOnly, projectPath])

  useEffect(() => {
    if (!open || !projectPath) return
    void refresh()
  }, [open, projectPath, refresh])

  const openRun = async (id: number) => {
    setDetailLoading(true)
    setError(null)
    setLog(null)
    try {
      const next = await AppService.CIRunDetail(id)
      setDetail(next ?? null)
    } catch (e) {
      setError(errText(e))
    } finally {
      setDetailLoading(false)
    }
  }

  const loadLog = async (runId: number, jobId: number, onlyFailed: boolean) => {
    setLogLoading(true)
    setError(null)
    try {
      const next = await AppService.CILog(runId, jobId, onlyFailed)
      setLog(next ?? null)
    } catch (e) {
      setError(errText(e))
    } finally {
      setLogLoading(false)
    }
  }

  const askRerun = async (runId: number, jobId: number, onlyFailed: boolean) => {
    setMutateBusy(true)
    setError(null)
    try {
      const prev = await AppService.PreviewCIRerun(runId, jobId, onlyFailed)
      setRerunPreview(prev ?? null)
    } catch (e) {
      setError(errText(e))
    } finally {
      setMutateBusy(false)
    }
  }

  const confirmRerun = async () => {
    if (!rerunPreview) return
    setMutateBusy(true)
    setError(null)
    try {
      await AppService.ConfirmCIRerun(
        rerunPreview.runId,
        rerunPreview.jobId ?? 0,
        rerunPreview.failedOnly,
      )
      setRerunPreview(null)
      await openRun(rerunPreview.runId)
      await refresh()
    } catch (e) {
      setError(errText(e))
    } finally {
      setMutateBusy(false)
    }
  }

  const askDispatch = async (workflow: string) => {
    setMutateBusy(true)
    setError(null)
    try {
      const prev = await AppService.PreviewCIDispatch(workflow, "", [])
      setDispatchPreview(prev ?? null)
      const seed: Record<string, string> = {}
      if (prev?.fields) {
        for (const [k, v] of Object.entries(prev.fields)) {
          if (k && v != null) seed[k] = String(v)
        }
      }
      for (const inp of prev?.inputs ?? []) {
        if (seed[inp.id] != null && seed[inp.id] !== "") continue
        if (inp.default) seed[inp.id] = inp.default
        else if (inp.type === "choice" && inp.options?.[0]) seed[inp.id] = inp.options[0]
        else if (inp.type === "boolean") seed[inp.id] = "false"
      }
      setDispatchFields(seed)
    } catch (e) {
      setError(errText(e))
    } finally {
      setMutateBusy(false)
    }
  }

  const confirmDispatch = async () => {
    if (!dispatchPreview) return
    if (dispatchMissingRequired.length > 0) {
      setError(
        `Informe o(s) input(s) obrigatório(s): ${dispatchMissingRequired.join(", ")}`,
      )
      return
    }
    setMutateBusy(true)
    setError(null)
    try {
      await AppService.ConfirmCIDispatch(
        dispatchPreview.workflow,
        dispatchPreview.ref ?? "",
        fieldsToPairs(dispatchFields),
      )
      setDispatchPreview(null)
      setDispatchFields({})
      await refresh()
    } catch (e) {
      setError(errText(e))
    } finally {
      setMutateBusy(false)
    }
  }


  const askFix = async (runId: number, jobId: number) => {
    setMutateBusy(true)
    setError(null)
    try {
      const prev = await AppService.PreviewCIFix(runId, jobId)
      setFixPreview(prev ?? null)
      setFixMessage(prev?.commitMessage ?? "")
    } catch (e) {
      setError(errText(e))
    } finally {
      setMutateBusy(false)
    }
  }

  const confirmFix = async () => {
    if (!fixPreview) return
    setMutateBusy(true)
    setError(null)
    try {
      await AppService.ConfirmCIFix(fixMessage || fixPreview.commitMessage, true)
      setFixPreview(null)
      await refresh()
      if (fixPreview.runId) await openRun(fixPreview.runId)
    } catch (e) {
      setError(errText(e))
    } finally {
      setMutateBusy(false)
    }
  }

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent
          side="right"
          className="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl"
        >
          <SheetHeader className="shrink-0 border-b px-4 py-3">
            <SheetTitle className="flex items-center gap-2 text-base">
              <Workflow className="size-4" />
              CI / GitHub Actions
            </SheetTitle>
            <p className="text-xs text-muted-foreground">
              {status?.branch
                ? `branch ${status.branch} · HEAD ${shortSha(status.headSha)}`
                : "Observar e reagir aos runs do projeto aberto"}
            </p>
          </SheetHeader>

          <div className="flex shrink-0 flex-wrap items-center gap-3 border-b px-4 py-2">
            <div className="flex items-center gap-2">
              <Checkbox
                id="ci-failed-only"
                checked={failedOnly}
                onCheckedChange={(v) => setFailedOnly(v === true)}
              />
              <Label htmlFor="ci-failed-only" className="text-xs font-normal">
                Só falhos
              </Label>
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={() => void refresh()}
              disabled={loading || mutateBusy}
            >
              {loading ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <RefreshCw className="size-3.5" />
              )}
              Atualizar
            </Button>
            {detail && (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  setDetail(null)
                  setLog(null)
                }}
                className="ml-auto"
              >
                ← Lista
              </Button>
            )}
          </div>

          <div className="shrink-0 space-y-2 px-4 py-2">
            <ActionsUsageBanner
              usage={
                rerunPreview?.usage ??
                dispatchPreview?.usage ??
                detail?.usage ??
                status?.usage
              }
            />
            {error && (
              <Alert variant="destructive">
                <AlertDescription className="text-xs">{error}</AlertDescription>
              </Alert>
            )}
          </div>

          <ScrollArea className="min-h-0 flex-1 px-4 pb-4">
            {detailLoading && (
              <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
                Carregando run…
              </div>
            )}

            {!detailLoading && detail && (
              <div className="space-y-4">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge
                    variant={conclusionVariant(
                      detail.run.status,
                      detail.run.conclusion,
                      detail.run.failed,
                    )}
                  >
                    {labelOf(detail.run)}
                  </Badge>
                  <span className="text-sm font-medium">{detail.run.name}</span>
                  {detail.run.url && (
                    <a
                      href={detail.run.url}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:underline"
                    >
                      GitHub <ExternalLink className="size-3" />
                    </a>
                  )}
                </div>
                <p className="text-xs text-muted-foreground">
                  {detail.run.workflowName} · {detail.run.event} ·{" "}
                  {shortSha(detail.run.headSha)}
                </p>
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={logLoading || mutateBusy}
                    onClick={() => void loadLog(detail.run.id, 0, true)}
                  >
                    {logLoading ? (
                      <Loader2 className="size-3.5 animate-spin" />
                    ) : (
                      <FileText className="size-3.5" />
                    )}
                    Logs falhos
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={logLoading || mutateBusy}
                    onClick={() => void loadLog(detail.run.id, 0, false)}
                  >
                    Log completo
                  </Button>
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={mutateBusy}
                    onClick={() => void askRerun(detail.run.id, 0, false)}
                  >
                    <RotateCcw className="size-3.5" />
                    Re-run
                  </Button>
                  {detail.run.failed && (
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={mutateBusy}
                      onClick={() => void askRerun(detail.run.id, 0, true)}
                    >
                      <RotateCcw className="size-3.5" />
                      Re-run falhos
                    </Button>
                  )}
                  {detail.run.failed && (
                    <Button
                      size="sm"
                      variant="default"
                      disabled={mutateBusy}
                      onClick={() => void askFix(detail.run.id, 0)}
                    >
                      <Wrench className="size-3.5" />
                      Corrigir com IA
                    </Button>
                  )}
                  {log && (
                    <Button size="sm" variant="ghost" onClick={() => setLog(null)}>
                      Fechar log
                    </Button>
                  )}
                </div>

                {log && (
                  <div className="space-y-2 rounded-lg border p-3">
                    <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                      <span className="font-medium text-foreground">
                        Log redigido
                      </span>
                      <span>
                        {log.bytes} bytes
                        {log.truncated ? " · truncado" : ""}
                        {log.jobId ? ` · job ${log.jobId}` : ""}
                        {log.failedOnly ? " · só falhos" : ""}
                      </span>
                    </div>
                    {log.message && (
                      <p className="text-xs text-muted-foreground">
                        {log.message}
                      </p>
                    )}
                    <pre className="max-h-[50vh] overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted/50 p-2 font-mono text-[11px] leading-relaxed">
                      {log.redactedText || "(vazio)"}
                    </pre>
                  </div>
                )}

                {(detail.jobs ?? []).map((job) => (
                  <div key={job.id} className="rounded-lg border p-3">
                    <div className="mb-2 flex flex-wrap items-center gap-2">
                      <Badge
                        variant={conclusionVariant(
                          job.status,
                          job.conclusion,
                          job.failed,
                        )}
                      >
                        {job.conclusion || job.status}
                      </Badge>
                      <span className="text-sm font-medium">{job.name}</span>
                      <div className="ml-auto flex gap-1">
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-7 px-2 text-xs"
                          disabled={logLoading || mutateBusy}
                          onClick={() =>
                            void loadLog(detail.run.id, job.id, job.failed)
                          }
                        >
                          <FileText className="size-3" />
                          Log
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-7 px-2 text-xs"
                          disabled={mutateBusy}
                          onClick={() =>
                            void askRerun(detail.run.id, job.id, false)
                          }
                        >
                          <RotateCcw className="size-3" />
                          Re-run
                        </Button>
                      </div>
                    </div>
                    <ul className="space-y-1 text-xs">
                      {(job.steps ?? []).map((step) => (
                        <li
                          key={`${job.id}-${step.number}`}
                          className={cn(
                            "flex justify-between gap-2 font-mono",
                            step.failed && "text-destructive",
                          )}
                        >
                          <span>
                            {step.number}. {step.name}
                          </span>
                          <span className="text-muted-foreground">
                            {step.conclusion || step.status}
                          </span>
                        </li>
                      ))}
                    </ul>
                  </div>
                ))}
                {(detail.jobs ?? []).length === 0 && (
                  <p className="text-sm text-muted-foreground">Run sem jobs.</p>
                )}
              </div>
            )}

            {!detailLoading && !detail && (
              <>
                {loading && !status ? (
                  <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
                    <Loader2 className="size-4 animate-spin" />
                    Listando runs…
                  </div>
                ) : (status?.runs ?? []).length === 0 ? (
                  <p className="py-8 text-sm text-muted-foreground">
                    Nenhum workflow run encontrado
                    {failedOnly ? " com falha" : ""} nesta branch.
                  </p>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Status</TableHead>
                        <TableHead>Run</TableHead>
                        <TableHead>Event</TableHead>
                        <TableHead>SHA</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {(status?.runs ?? []).map((run) => (
                        <TableRow
                          key={run.id}
                          className="cursor-pointer"
                          onClick={() => void openRun(run.id)}
                        >
                          <TableCell>
                            <Badge
                              variant={conclusionVariant(
                                run.status,
                                run.conclusion,
                                run.failed,
                              )}
                            >
                              {labelOf(run)}
                            </Badge>
                          </TableCell>
                          <TableCell className="max-w-[12rem] truncate text-sm">
                            {run.name || run.workflowName}
                          </TableCell>
                          <TableCell className="text-xs text-muted-foreground">
                            {run.event}
                          </TableCell>
                          <TableCell className="font-mono text-xs">
                            {shortSha(run.headSha)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}

                {workflows.length > 0 && (
                  <div className="mt-6 space-y-2">
                    <h3 className="text-sm font-medium">Workflows · dispatch</h3>
                    <p className="text-xs text-muted-foreground">
                      Disparo manual consome minutos de Actions — confirmação
                      obrigatória.
                    </p>
                    <ul className="space-y-2">
                      {workflows.map((wf) => (
                        <li
                          key={wf.id}
                          className="flex items-center gap-2 rounded-lg border px-3 py-2"
                        >
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-sm font-medium">
                              {wf.name}
                            </p>
                            <p className="truncate font-mono text-[11px] text-muted-foreground">
                              {wf.path || wf.state}
                            </p>
                          </div>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={mutateBusy || wf.state === "disabled"}
                            onClick={() =>
                              void askDispatch(wf.path || wf.name)
                            }
                          >
                            <Play className="size-3.5" />
                            Dispatch
                          </Button>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </>
            )}
          </ScrollArea>
        </SheetContent>
      </Sheet>

      <AlertDialog
        open={!!rerunPreview}
        onOpenChange={(v) => {
          if (!v) setRerunPreview(null)
        }}
      >
        <AlertDialogContent className="sm:max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle>Confirmar re-run</AlertDialogTitle>
            <AlertDialogDescription className="space-y-2 text-left">
              <span className="block font-medium text-foreground">
                {rerunPreview?.costWarning}
              </span>
              <span className="block text-xs">
                Run #{rerunPreview?.runId} · {rerunPreview?.runName} ·{" "}
                {rerunPreview?.headBranch}
                {rerunPreview?.jobId ? ` · job ${rerunPreview.jobId}` : ""}
                {rerunPreview?.failedOnly ? " · só falhos" : ""}
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          {rerunPreview?.usage && (
            <ActionsUsageBanner usage={rerunPreview.usage} />
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={mutateBusy}>Cancelar</AlertDialogCancel>
            <AlertDialogAction
              disabled={mutateBusy}
              onClick={(e) => {
                e.preventDefault()
                void confirmRerun()
              }}
            >
              {mutateBusy ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <RotateCcw className="size-3.5" />
              )}
              Confirmar re-run
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={!!dispatchPreview}
        onOpenChange={(v) => {
          if (!v) {
            setDispatchPreview(null)
            setDispatchFields({})
          }
        }}
      >
        <AlertDialogContent className="sm:max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle>Confirmar dispatch</AlertDialogTitle>
            <AlertDialogDescription className="space-y-2 text-left">
              <span className="block font-medium text-foreground">
                {dispatchPreview?.costWarning}
              </span>
              <span className="block text-xs">
                Workflow: {dispatchPreview?.workflow}
                {!dispatchPreview?.canDispatch
                  ? " · workflow_dispatch pode não estar habilitado"
                  : ""}
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          {dispatchInputs.length > 0 ? (
            <div className="flex flex-col gap-3 py-1">
              {dispatchInputs.map((inp) => (
                <DispatchInputField
                  key={inp.id}
                  input={inp}
                  value={dispatchFields[inp.id] ?? ""}
                  onChange={(v) => setDispatchField(inp.id, v)}
                  disabled={mutateBusy}
                />
              ))}
            </div>
          ) : null}
          {dispatchPreview?.usage && (
            <ActionsUsageBanner usage={dispatchPreview.usage} />
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={mutateBusy}>Cancelar</AlertDialogCancel>
            <AlertDialogAction
              disabled={mutateBusy || dispatchMissingRequired.length > 0}
              onClick={(e) => {
                e.preventDefault()
                void confirmDispatch()
              }}
            >
              {mutateBusy ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Play className="size-3.5" />
              )}
              Confirmar dispatch
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={!!fixPreview}
        onOpenChange={(v) => {
          if (!v) setFixPreview(null)
        }}
      >
        <AlertDialogContent className="sm:max-w-lg">
          <AlertDialogHeader>
            <AlertDialogTitle>Corrigir CI com IA</AlertDialogTitle>
            <AlertDialogDescription className="space-y-2 text-left">
              <span className="block text-sm text-foreground">
                {fixPreview?.summary}
              </span>
              {fixPreview?.defaultBranchWarning ? (
                <span className="block text-destructive">
                  {fixPreview.defaultBranchWarning}
                </span>
              ) : null}
              <span className="block text-xs text-muted-foreground">
                {(fixPreview?.files ?? []).length} arquivo(s) · commit+push após
                confirmar
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-2 text-xs">
            <ul className="max-h-32 space-y-1 overflow-auto rounded-md border p-2 font-mono">
              {(fixPreview?.files ?? []).map((f) => (
                <li key={f.path}>
                  {f.path} ({f.bytes} B)
                </li>
              ))}
            </ul>
            <textarea
              className="min-h-20 w-full rounded-md border bg-background p-2 font-mono text-xs"
              value={fixMessage}
              onChange={(e) => setFixMessage(e.target.value)}
            />
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={mutateBusy}>Cancelar</AlertDialogCancel>
            <AlertDialogAction
              disabled={
                mutateBusy ||
                !fixPreview?.files?.length ||
                !fixMessage.trim()
              }
              onClick={(e) => {
                e.preventDefault()
                void confirmFix()
              }}
            >
              {mutateBusy ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Wrench className="size-3.5" />
              )}
              Aplicar, commit e push
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
