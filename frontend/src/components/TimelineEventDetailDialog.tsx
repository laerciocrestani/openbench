import { useEffect, useState } from "react"
import { format, formatDistanceToNow, parseISO } from "date-fns"
import { ptBR } from "date-fns/locale"
import { Browser } from "@wailsio/runtime"
import {
  CheckCircle2,
  ExternalLink,
  GitCommit,
  GitMerge,
  GitPullRequest,
  Loader2,
  XCircle,
  Circle,
} from "lucide-react"

import { AppService } from "../../bindings/github.com/laerciocrestani/openbench"
import type {
  PRDetailView,
  TimelineEventView,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
import { Badge } from "@/components/ui/badge"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

function kindLabel(kind: string): string {
  switch (kind) {
    case "merge":
      return "merge"
    case "pr_opened":
      return "PR aberto"
    case "pr_merged":
      return "PR mergeado"
    case "pr_closed":
      return "PR fechado"
    default:
      return "commit"
  }
}

function KindIcon({ kind }: { kind: string }) {
  const cls = "size-3.5 shrink-0"
  switch (kind) {
    case "merge":
      return <GitMerge className={cn(cls, "text-violet-500")} />
    case "pr_opened":
      return <GitPullRequest className={cn(cls, "text-sky-500")} />
    case "pr_merged":
      return <GitMerge className={cn(cls, "text-emerald-500")} />
    case "pr_closed":
      return <GitPullRequest className={cn(cls, "text-muted-foreground")} />
    default:
      return <GitCommit className={cn(cls, "text-muted-foreground")} />
  }
}

function formatWhen(at: string): { absolute: string; relative: string } {
  try {
    const d = parseISO(at)
    return {
      absolute: format(d, "dd MMM yyyy HH:mm", { locale: ptBR }),
      relative: formatDistanceToNow(d, { addSuffix: true, locale: ptBR }),
    }
  } catch {
    return { absolute: at, relative: "" }
  }
}

async function openExternalURL(url: string) {
  const value = url.trim()
  if (!value) return
  try {
    await Browser.OpenURL(value)
  } catch {
    window.open(value, "_blank", "noopener,noreferrer")
  }
}

function checkIcon(bucket: string) {
  switch (bucket.toLowerCase()) {
    case "pass":
      return <CheckCircle2 className="size-3.5 text-emerald-500" />
    case "fail":
      return <XCircle className="size-3.5 text-destructive" />
    case "pending":
      return <Loader2 className="size-3.5 animate-spin text-amber-500" />
    default:
      return <Circle className="size-3.5 text-muted-foreground" />
  }
}

function convKindLabel(kind: string): string {
  switch (kind) {
    case "description":
      return "Descrição"
    case "review":
      return "Review"
    default:
      return "Comentário"
  }
}

export function TimelineEventDetailDialog({
  event,
  onOpenChange,
}: {
  event: TimelineEventView | null
  onOpenChange: (open: boolean) => void
}) {
  const isPR = Boolean(event?.kind.startsWith("pr_"))
  const isMerge = event?.kind === "merge"
  const when = event ? formatWhen(event.at) : null
  const [detail, setDetail] = useState<PRDetailView | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [tab, setTab] = useState("conversation")

  useEffect(() => {
    if (!event || !isPR || !event.prNumber) {
      setDetail(null)
      setError(null)
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    setDetail(null)
    setTab("conversation")
    void (async () => {
      try {
        const d = await AppService.LoadPRDetail(event.prNumber!)
        if (!cancelled) setDetail(d ?? null)
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e))
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [event, isPR])

  const summary =
    detail?.headRefName ||
    event?.subtitle ||
    (event?.refs?.length ? event.refs.join(", ") : "")

  return (
    <Dialog open={!!event} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] w-[min(52rem,calc(100%-2rem))] max-w-none flex-col gap-3 overflow-hidden sm:max-w-none">
        <DialogHeader className="space-y-1">
          <DialogTitle className="flex items-center gap-2">
            {event ? <KindIcon kind={event.kind} /> : null}
            {event ? kindLabel(event.kind) : "Detalhe"}
            {event?.shortHash ? (
              <span className="font-mono text-sm font-normal text-muted-foreground">
                {event.shortHash}
              </span>
            ) : null}
            {event?.prNumber ? (
              <span className="text-sm font-normal text-muted-foreground">#{event.prNumber}</span>
            ) : null}
          </DialogTitle>
          <DialogDescription className="text-xs">
            {isMerge
              ? "Detalhes do merge (somente visualização)"
              : isPR
                ? "Informação e detalhes do pull request"
                : "Detalhes do commit (somente visualização)"}
          </DialogDescription>
        </DialogHeader>

        {event && when ? (
          <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
            {/* Informação condensada */}
            <section className="shrink-0 space-y-2 rounded-lg border bg-muted/15 px-3 py-2.5">
              <p className="text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
                Informação
              </p>
              <p className="text-sm font-medium leading-snug break-words">{event.title}</p>
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
                <span className="inline-flex items-center gap-1">
                  <Badge variant="outline" className="h-5 px-1.5 text-[10px] font-normal capitalize">
                    {kindLabel(event.kind)}
                  </Badge>
                </span>
                <span title={when.absolute}>
                  {when.absolute}
                  {when.relative ? ` (${when.relative})` : ""}
                </span>
                {event.author ? <span>@{event.author}</span> : null}
                {event.prNumber ? <span>PR #{event.prNumber}</span> : null}
                {event.shortHash || event.hash ? (
                  <span className="font-mono">{event.shortHash || event.hash}</span>
                ) : null}
                {summary ? <span className="font-mono">{summary}</span> : null}
                {detail && (detail.additions > 0 || detail.deletions > 0) ? (
                  <span>
                    <span className="text-emerald-600 dark:text-emerald-400">+{detail.additions}</span>
                    {" / "}
                    <span className="text-destructive">-{detail.deletions}</span>
                    {detail.changedFiles > 0 ? ` · ${detail.changedFiles} files` : ""}
                  </span>
                ) : null}
                {event.url ? (
                  <button
                    type="button"
                    className="inline-flex max-w-full items-center gap-1 text-sky-600 hover:underline dark:text-sky-400"
                    onClick={() => void openExternalURL(event.url!)}
                  >
                    <ExternalLink className="size-3 shrink-0" />
                    <span className="truncate">{event.url}</span>
                  </button>
                ) : null}
              </div>
            </section>

            {/* Detalhes */}
            {isPR ? (
              <section className="flex min-h-0 flex-1 flex-col gap-2 overflow-hidden">
                <p className="shrink-0 text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
                  Detalhes
                </p>
                {loading ? (
                  <div className="flex flex-1 items-center justify-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="size-4 animate-spin" />
                    Carregando PR…
                  </div>
                ) : error ? (
                  <p className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive">
                    {error}
                  </p>
                ) : (
                  <Tabs
                    value={tab}
                    onValueChange={setTab}
                    className="flex w-full min-h-0 flex-1 flex-col gap-2"
                  >
                    <TabsList
                      variant="line"
                      className="h-auto w-full shrink-0 justify-start overflow-visible group-data-horizontal/tabs:h-auto"
                    >
                      <TabsTrigger value="conversation" className="flex-none">
                        Conversação
                      </TabsTrigger>
                      <TabsTrigger value="commits" className="flex-none">
                        Commits
                        {detail?.commits?.length ? (
                          <Badge variant="secondary" className="h-4 px-1 text-[10px]">
                            {detail.commits.length}
                          </Badge>
                        ) : null}
                      </TabsTrigger>
                      <TabsTrigger value="checks" className="flex-none">
                        Checks
                        {detail?.checks?.length ? (
                          <Badge variant="secondary" className="h-4 px-1 text-[10px]">
                            {detail.checks.length}
                          </Badge>
                        ) : null}
                      </TabsTrigger>
                      <TabsTrigger value="files" className="flex-none">
                        Files changed
                        {detail?.files?.length || detail?.changedFiles ? (
                          <Badge variant="secondary" className="h-4 px-1 text-[10px]">
                            {detail.files?.length || detail.changedFiles}
                          </Badge>
                        ) : null}
                      </TabsTrigger>
                    </TabsList>

                    <TabsContent
                      value="conversation"
                      className="mt-0 min-h-0 flex-1 overflow-y-auto rounded-lg border p-3"
                    >
                      {(detail?.conversation?.length ?? 0) === 0 ? (
                        <p className="text-xs text-muted-foreground">Sem conversação.</p>
                      ) : (
                        <ul className="space-y-3">
                          {detail!.conversation!.map((item, i) => {
                            const at = item.at ? formatWhen(item.at) : null
                            return (
                              <li key={`${item.kind}-${item.at}-${i}`} className="space-y-1 border-b pb-3 last:border-0">
                                <div className="flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
                                  <Badge variant="outline" className="h-5 px-1.5 text-[10px] font-normal">
                                    {convKindLabel(item.kind)}
                                  </Badge>
                                  {item.author ? <span>@{item.author}</span> : null}
                                  {item.state ? <span>{item.state}</span> : null}
                                  {at ? <span title={at.absolute}>{at.relative || at.absolute}</span> : null}
                                </div>
                                {item.body ? (
                                  <pre className="whitespace-pre-wrap break-words font-sans text-sm text-foreground">
                                    {item.body}
                                  </pre>
                                ) : (
                                  <p className="text-xs text-muted-foreground italic">Sem texto</p>
                                )}
                              </li>
                            )
                          })}
                        </ul>
                      )}
                    </TabsContent>

                    <TabsContent
                      value="commits"
                      className="mt-0 min-h-0 flex-1 overflow-y-auto rounded-lg border"
                    >
                      {(detail?.commits?.length ?? 0) === 0 ? (
                        <p className="p-3 text-xs text-muted-foreground">Nenhum commit listado.</p>
                      ) : (
                        <table className="w-full table-fixed text-sm">
                          <thead className="sticky top-0 bg-popover [&_tr]:border-b">
                            <tr className="border-b">
                              <th className="h-9 w-20 px-3 text-left font-medium">Hash</th>
                              <th className="h-9 px-3 text-left font-medium">Mensagem</th>
                              <th className="h-9 w-36 px-3 text-left font-medium">Autor</th>
                            </tr>
                          </thead>
                          <tbody>
                            {detail!.commits!.map((c) => (
                              <tr key={c.oid} className="border-b last:border-0">
                                <td className="px-3 py-2 align-top font-mono text-xs">
                                  {c.shortOid || c.oid.slice(0, 7)}
                                </td>
                                <td className="px-3 py-2 align-top break-words whitespace-normal">
                                  {c.messageHeadline}
                                </td>
                                <td className="px-3 py-2 align-top text-xs text-muted-foreground">
                                  {(c.authors ?? []).join(", ") || "—"}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </TabsContent>

                    <TabsContent
                      value="checks"
                      className="mt-0 min-h-0 flex-1 overflow-y-auto rounded-lg border p-3"
                    >
                      {(detail?.checks?.length ?? 0) === 0 ? (
                        <p className="text-xs text-muted-foreground">Nenhum check disponível.</p>
                      ) : (
                        <ul className="space-y-2">
                          {detail!.checks!.map((ch) => (
                            <li
                              key={`${ch.name}-${ch.state}`}
                              className="flex items-start gap-2 rounded-md border px-2.5 py-2"
                            >
                              <span className="mt-0.5">{checkIcon(ch.bucket || ch.state)}</span>
                              <div className="min-w-0 flex-1">
                                <p className="text-sm font-medium">{ch.name}</p>
                                <p className="text-[11px] text-muted-foreground">
                                  {ch.bucket || ch.state}
                                </p>
                              </div>
                              {ch.link ? (
                                <button
                                  type="button"
                                  className="text-sky-600 dark:text-sky-400"
                                  onClick={() => void openExternalURL(ch.link!)}
                                  title={ch.link}
                                >
                                  <ExternalLink className="size-3.5" />
                                </button>
                              ) : null}
                            </li>
                          ))}
                        </ul>
                      )}
                    </TabsContent>

                    <TabsContent
                      value="files"
                      className="mt-0 min-h-0 flex-1 overflow-y-auto rounded-lg border"
                    >
                      {(detail?.files?.length ?? 0) === 0 ? (
                        <p className="p-3 text-xs text-muted-foreground">Nenhum arquivo listado.</p>
                      ) : (
                        <table className="w-full table-fixed text-sm">
                          <thead className="sticky top-0 bg-popover [&_tr]:border-b">
                            <tr className="border-b">
                              <th className="h-9 px-3 text-left font-medium">Arquivo</th>
                              <th className="h-9 w-16 px-3 text-right font-medium">+</th>
                              <th className="h-9 w-16 px-3 text-right font-medium">−</th>
                            </tr>
                          </thead>
                          <tbody>
                            {detail!.files!.map((f) => (
                              <tr key={f.path} className="border-b last:border-0">
                                <td className="px-3 py-2 align-top font-mono text-xs break-all">
                                  {f.changeType ? (
                                    <span className="mr-1.5 text-muted-foreground">{f.changeType}</span>
                                  ) : null}
                                  {f.path}
                                </td>
                                <td className="px-3 py-2 align-top text-right text-xs text-emerald-600 dark:text-emerald-400">
                                  {f.additions}
                                </td>
                                <td className="px-3 py-2 align-top text-right text-xs text-destructive">
                                  {f.deletions}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </TabsContent>
                  </Tabs>
                )}
              </section>
            ) : (
              <div className="min-h-0 flex-1 overflow-y-auto rounded-lg border p-3 text-sm text-muted-foreground">
                {(event.refs?.length ?? 0) > 0 ? (
                  <p className="mb-2 font-mono text-xs">{event.refs!.join(" · ")}</p>
                ) : null}
                <p>Somente visualização do evento na timeline.</p>
              </div>
            )}
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
