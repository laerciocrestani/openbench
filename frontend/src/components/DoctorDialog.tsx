import { AlertTriangle, CheckCircle2, Loader2, RefreshCw, Sparkles, Stethoscope, XCircle } from "lucide-react"

import type { DoctorView } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
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

function overallMeta(overall: string): {
  label: string
  variant: "default" | "secondary" | "destructive" | "outline"
  Icon: typeof CheckCircle2
} {
  switch (overall) {
    case "critical":
      return { label: "Crítico", variant: "destructive", Icon: XCircle }
    case "warn":
      return { label: "Atenção", variant: "secondary", Icon: AlertTriangle }
    default:
      return { label: "Saudável", variant: "default", Icon: CheckCircle2 }
  }
}

function issueTone(level: string): string {
  switch (level) {
    case "critical":
      return "border-destructive/40 bg-destructive/5"
    case "warn":
      return "border-amber-500/30 bg-amber-500/5"
    default:
      return "border-border bg-muted/20"
  }
}

export function DoctorDialog({
  open,
  onOpenChange,
  report,
  loading,
  explaining,
  onRefresh,
  onExplain,
  onStartCommit,
  onOpenFix,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  report: DoctorView | null
  loading: boolean
  explaining: boolean
  onRefresh: () => void
  onExplain: () => void
  onStartCommit: () => void
  onOpenFix: () => void
}) {
  const meta = overallMeta(report?.overall || "ok")
  const OverallIcon = meta.Icon
  const issues = report?.issues ?? []
  const recs = report?.recommendations ?? []
  const ai = report?.ai

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] w-[min(48rem,calc(100%-2rem))] max-w-none flex-col gap-0 overflow-hidden p-0 sm:max-w-none">
        <DialogHeader className="shrink-0 space-y-2 border-b px-4 py-3 text-left">
          <DialogTitle className="flex items-center gap-2 text-base">
            <Stethoscope className="size-4 text-muted-foreground" />
            Doctor
            {report && (
              <Badge variant={meta.variant} className="gap-1 font-normal">
                <OverallIcon className="size-3" />
                {meta.label}
              </Badge>
            )}
          </DialogTitle>
          <DialogDescription className="text-xs">
            {report
              ? `${report.branch || "—"} · base ${report.base || "—"}`
              : "Panorama de saúde do repositório"}
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden">
          <div className="space-y-4 px-4 py-3">
            {loading && !report ? (
              <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
                Analisando repositório…
              </div>
            ) : null}

            {report && issues.length === 0 && (
              <p className="rounded-lg border border-border bg-muted/20 px-3 py-2 text-sm text-muted-foreground">
                Nenhum problema detectado. Working tree e sync parecem ok.
              </p>
            )}

            {issues.length > 0 && (
              <section className="space-y-2">
                <h3 className="text-xs font-medium text-muted-foreground">Problemas</h3>
                <ul className="space-y-2">
                  {issues.map((issue) => (
                    <li
                      key={`${issue.code}-${issue.title}`}
                      className={cn("rounded-lg border px-3 py-2", issueTone(issue.level))}
                    >
                      <div className="flex items-start gap-2">
                        <Badge variant="outline" className="mt-0.5 shrink-0 text-[10px] capitalize">
                          {issue.level}
                        </Badge>
                        <div className="min-w-0">
                          <p className="text-sm font-medium text-foreground">{issue.title}</p>
                          {issue.detail ? (
                            <p className="mt-0.5 text-xs text-muted-foreground">{issue.detail}</p>
                          ) : null}
                        </div>
                      </div>
                    </li>
                  ))}
                </ul>
              </section>
            )}

            {(recs.length > 0 || issues.length > 0) && (
              <section className="space-y-2">
                {recs.length > 0 && (
                  <>
                    <h3 className="text-xs font-medium text-muted-foreground">Recomendações</h3>
                    <ul className="list-inside list-disc space-y-1 text-sm text-foreground">
                      {recs.map((rec) => (
                        <li key={rec} className="text-muted-foreground">
                          <span className="text-foreground">{rec}</span>
                        </li>
                      ))}
                    </ul>
                  </>
                )}
                <div className="flex flex-wrap gap-1.5 pt-1">
                  {issues.some((i) => i.code === "dirty_tree") &&
                    !issues.some((i) => i.code === "work_on_merged_branch") && (
                      <Button size="sm" onClick={onStartCommit}>
                        Commit
                      </Button>
                    )}
                  {issues.some((i) =>
                    [
                      "work_on_merged_branch",
                      "behind_remote",
                      "branch_diverged",
                      "base_diverged",
                      "commits_on_base",
                      "build_artifacts",
                    ].includes(i.code),
                  ) && (
                    <Button size="sm" variant="outline" onClick={onOpenFix}>
                      Ajustar com o Doctor
                    </Button>
                  )}
                </div>
              </section>
            )}

            {ai && (
              <section className="space-y-2 rounded-lg border border-border bg-muted/15 px-3 py-2">
                <h3 className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                  <Sparkles className="size-3.5" />
                  Explicação IA
                  {ai.risk ? (
                    <Badge variant="outline" className="text-[10px]">
                      risco {ai.risk}
                    </Badge>
                  ) : null}
                </h3>
                {ai.summary ? <p className="text-sm break-words whitespace-normal">{ai.summary}</p> : null}
                {ai.cause ? (
                  <p className="text-xs break-words whitespace-normal text-muted-foreground">
                    <span className="font-medium text-foreground">Causa: </span>
                    {ai.cause}
                  </p>
                ) : null}
                {(ai.steps ?? []).length > 0 && (
                  <ol className="list-inside list-decimal space-y-1 text-xs break-words text-muted-foreground">
                    {ai.steps!.map((step) => (
                      <li key={step}>{step}</li>
                    ))}
                  </ol>
                )}
                {(ai.warnings ?? []).length > 0 && (
                  <ul className="space-y-1 text-xs break-words text-amber-600 dark:text-amber-400">
                    {ai.warnings!.map((w) => (
                      <li key={w}>⚠ {w}</li>
                    ))}
                  </ul>
                )}
              </section>
            )}

            {report?.explainError ? (
              <p className="text-xs text-destructive">{report.explainError}</p>
            ) : null}
          </div>
        </div>

        <DialogFooter className="mx-0 mb-0 shrink-0 gap-2 rounded-none border-t px-4 py-3 sm:justify-between">
          <Button
            size="sm"
            variant="outline"
            onClick={onExplain}
            disabled={loading || explaining || !report}
          >
            {explaining ? <Loader2 className="animate-spin" /> : <Sparkles />}
            Explicar com IA
          </Button>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={onRefresh} disabled={loading || explaining}>
              {loading ? <Loader2 className="animate-spin" /> : <RefreshCw />}
              Atualizar
            </Button>
            <Button size="sm" variant="secondary" onClick={() => onOpenChange(false)}>
              Fechar
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
