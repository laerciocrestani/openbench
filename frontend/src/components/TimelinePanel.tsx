import { useEffect, useMemo, useRef, useState, type MouseEvent } from "react"
import { format, formatDistanceToNow, parseISO } from "date-fns"
import { ptBR } from "date-fns/locale"
import { Browser, Clipboard } from "@wailsio/runtime"

import type {
  TimelineEventView,
  TimelineView,
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import { ScrollArea } from "@/components/ui/scroll-area"
import { TimelineEventDetailDialog } from "@/components/TimelineEventDetailDialog"
import { cn } from "@/lib/utils"
import {
  Copy,
  ExternalLink,
  GitBranch,
  GitCommit,
  GitMerge,
  GitPullRequest,
  Loader2,
  RotateCcw,
  Trash2,
  Undo2,
} from "lucide-react"

export type TimelineConfirmAction =
  | { type: "revert"; hash: string; isMerge: boolean; title: string }
  | { type: "reset"; hash: string; mode: "soft" | "mixed" | "hard"; title: string }
  | { type: "delete-branch"; name: string; title: string }
  | { type: "merge-pr"; number: number; method: "squash" | "merge" | "rebase"; title: string }

type CursorMenu = {
  event: TimelineEventView
  x: number
  y: number
}

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

function localBranches(refs: string[] | null | undefined): string[] {
  const out: string[] = []
  for (const ref of refs ?? []) {
    const name = ref.trim()
    if (!name || name === "HEAD") continue
    if (name.startsWith("origin/") || name.startsWith("tag:")) continue
    out.push(name)
  }
  return out
}

async function copyText(text: string) {
  const value = text.trim()
  if (!value) return
  try {
    await Clipboard.SetText(value)
  } catch {
    try {
      await navigator.clipboard.writeText(value)
    } catch {
      /* ignore */
    }
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

function cursorAnchor(x: number, y: number) {
  return {
    getBoundingClientRect: () =>
      DOMRect.fromRect({
        x,
        y,
        width: 0,
        height: 0,
      }),
  }
}

export function TimelinePanel({
  timeline,
  loading,
  onLoadMore,
  onConfirmAction,
  onCheckoutBranch,
  actionBusy,
  className,
  compact,
}: {
  timeline: TimelineView | null
  loading: boolean
  onLoadMore: () => void
  onConfirmAction: (action: TimelineConfirmAction) => Promise<void> | void
  onCheckoutBranch: (name: string) => Promise<void> | void
  actionBusy?: boolean
  className?: string
  compact?: boolean
}) {
  const events = timeline?.events ?? []
  const hasMore = Boolean(timeline?.hasMore)
  const sentinelRef = useRef<HTMLDivElement | null>(null)
  const [pending, setPending] = useState<TimelineConfirmAction | null>(null)
  const [menu, setMenu] = useState<CursorMenu | null>(null)
  const [menuOpen, setMenuOpen] = useState(false)
  const [detail, setDetail] = useState<TimelineEventView | null>(null)

  const hint = useMemo(() => {
    if (!timeline) return null
    if (timeline.prIncluded) return "Commits + PRs"
    if (timeline.hasGH) return "Commits (PRs indisponíveis)"
    return "Commits locais"
  }, [timeline])

  useEffect(() => {
    if (!hasMore || loading) return
    const node = sentinelRef.current
    if (!node) return
    const root =
      node.closest("[data-slot=scroll-area-viewport]") ??
      node.closest("[data-slot=scroll-area]")
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) onLoadMore()
      },
      { root: root instanceof Element ? root : null, rootMargin: "40px", threshold: 0 },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [hasMore, loading, onLoadMore, events.length])

  const confirmCopy = useMemo(() => {
    if (!pending) return { title: "", description: "" }
    switch (pending.type) {
      case "revert":
        return {
          title: pending.isMerge ? "Reverter merge?" : "Reverter commit?",
          description: pending.isMerge
            ? `Será criado um novo commit com git revert -m 1 ${pending.hash.slice(0, 7)}. Working tree precisa estar limpa.`
            : `Será criado um novo commit com git revert ${pending.hash.slice(0, 7)}. Working tree precisa estar limpa.`,
        }
      case "reset":
        return {
          title: `Reset --${pending.mode}?`,
          description:
            pending.mode === "hard"
              ? `ATENÇÃO: git reset --hard ${pending.hash.slice(0, 7)} descarta alterações no working tree e no índice.`
              : `Move HEAD para ${pending.hash.slice(0, 7)} (--${pending.mode}). Commits posteriores saem do branch (recuperáveis via reflog).`,
        }
      case "delete-branch":
        return {
          title: `Apagar branch ${pending.name}?`,
          description: `Executa git branch -D ${pending.name}. Não remove a branch remota.`,
        }
      case "merge-pr":
        return {
          title: `Mergear PR #${pending.number}?`,
          description: `Executa gh pr merge ${pending.number} --${pending.method}. A PR precisa estar mergeável no GitHub.`,
        }
    }
  }, [pending])

  const openMenu = (ev: TimelineEventView, e: MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (actionBusy) return
    setMenu({ event: ev, x: e.clientX, y: e.clientY })
    setMenuOpen(true)
  }

  const menuEvent = menu?.event
  const isPR = Boolean(menuEvent?.kind.startsWith("pr_"))
  const isPROpen = menuEvent?.kind === "pr_opened"
  const isMerge = menuEvent?.kind === "merge"
  const isCommitLike = menuEvent?.kind === "commit" || isMerge
  const hash = menuEvent?.hash
  const shortHash = menuEvent?.shortHash
  const url = menuEvent?.url
  const prNumber = menuEvent?.prNumber ?? 0
  const branches = localBranches(menuEvent?.refs)

  return (
    <div className={cn("flex h-full min-h-0 flex-col", className)}>
      {!compact && (
        <div className="mb-2 flex flex-wrap items-center gap-2">
          {events.length > 0 && (
            <Badge variant="outline" className="font-normal">
              {events.length} eventos
            </Badge>
          )}
          {hint && <span className="text-[10px] text-muted-foreground">{hint}</span>}
        </div>
      )}

      {loading && events.length === 0 ? (
        <div className="flex flex-1 items-center justify-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="size-3.5 animate-spin" />
          Carregando…
        </div>
      ) : events.length === 0 ? (
        <p className="text-xs text-muted-foreground">Nenhum evento recente.</p>
      ) : (
        <ScrollArea className="min-h-0 flex-1">
          <ol className="relative space-y-0 border-l border-border/70 pl-4 pr-1">
            {events.map((ev) => (
              <TimelineRow
                key={ev.id}
                event={ev}
                disabled={actionBusy}
                onOpen={() => setDetail(ev)}
                onContextMenu={(e) => openMenu(ev, e)}
              />
            ))}
          </ol>
          <div ref={sentinelRef} className="h-1 w-full" aria-hidden />
          <div className="flex flex-col items-center gap-2 py-2">
            {loading && (
              <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
                <Loader2 className="size-3 animate-spin" />
                Carregando mais…
              </div>
            )}
            {hasMore && !loading && (
              <Button size="xs" variant="outline" onClick={onLoadMore}>
                Carregar mais
              </Button>
            )}
            {!hasMore && events.length > 0 && (
              <span className="text-[10px] text-muted-foreground">Fim da timeline</span>
            )}
          </div>
        </ScrollArea>
      )}

      <TimelineEventDetailDialog event={detail} onOpenChange={(open) => !open && setDetail(null)} />

      <DropdownMenu
        open={menuOpen}
        onOpenChange={(open) => {
          setMenuOpen(open)
          if (!open) {
            // Keep menu payload briefly so item onClick (open/copy) can finish.
            queueMicrotask(() => setMenu(null))
          }
        }}
        modal={false}
      >
        {menu ? (
          <DropdownMenuContent
            className="min-w-52"
            side="right"
            align="start"
            sideOffset={0}
            alignOffset={0}
            positionMethod="fixed"
            anchor={cursorAnchor(menu.x, menu.y)}
          >
            <DropdownMenuLabel>Ações</DropdownMenuLabel>
            <DropdownMenuSeparator />

            {isCommitLike && hash ? (
              <>
                <DropdownMenuItem onClick={() => void copyText(shortHash || hash)}>
                  <Copy className="size-3.5" />
                  Copiar hash
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    setPending({
                      type: "revert",
                      hash,
                      isMerge: Boolean(isMerge),
                      title: menuEvent?.title ?? "",
                    })
                  }
                >
                  <Undo2 className="size-3.5" />
                  Reverter ({isMerge ? "merge -m 1" : "commit"})
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    setPending({
                      type: "reset",
                      hash,
                      mode: "soft",
                      title: menuEvent?.title ?? "",
                    })
                  }
                >
                  <RotateCcw className="size-3.5" />
                  Reset --soft até aqui
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    setPending({
                      type: "reset",
                      hash,
                      mode: "mixed",
                      title: menuEvent?.title ?? "",
                    })
                  }
                >
                  <RotateCcw className="size-3.5" />
                  Reset --mixed até aqui
                </DropdownMenuItem>
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() =>
                    setPending({
                      type: "reset",
                      hash,
                      mode: "hard",
                      title: menuEvent?.title ?? "",
                    })
                  }
                >
                  <RotateCcw className="size-3.5" />
                  Reset --hard até aqui
                </DropdownMenuItem>
              </>
            ) : null}

            {isPR && url ? (
              <>
                <DropdownMenuItem
                  onClick={() => {
                    void openExternalURL(url)
                  }}
                >
                  <ExternalLink className="size-3.5" />
                  Abrir PR no GitHub
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => {
                    void copyText(url)
                  }}
                >
                  <Copy className="size-3.5" />
                  Copiar URL do PR
                </DropdownMenuItem>
              </>
            ) : null}

            {isPROpen && prNumber > 0 ? (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuLabel>Merge PR #{prNumber}</DropdownMenuLabel>
                <DropdownMenuItem
                  onClick={() =>
                    setPending({
                      type: "merge-pr",
                      number: prNumber,
                      method: "squash",
                      title: menuEvent?.title ?? "",
                    })
                  }
                >
                  <GitMerge className="size-3.5" />
                  Merge squash
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    setPending({
                      type: "merge-pr",
                      number: prNumber,
                      method: "merge",
                      title: menuEvent?.title ?? "",
                    })
                  }
                >
                  <GitMerge className="size-3.5" />
                  Merge commit
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    setPending({
                      type: "merge-pr",
                      number: prNumber,
                      method: "rebase",
                      title: menuEvent?.title ?? "",
                    })
                  }
                >
                  <GitMerge className="size-3.5" />
                  Merge rebase
                </DropdownMenuItem>
              </>
            ) : null}

            {branches.length > 0 ? (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuLabel>Branches</DropdownMenuLabel>
                {branches.map((name) => (
                  <DropdownMenuItem
                    key={`co-${name}`}
                    onClick={() => void onCheckoutBranch(name)}
                  >
                    <GitBranch className="size-3.5" />
                    Checkout {name}
                  </DropdownMenuItem>
                ))}
                {branches.map((name) => (
                  <DropdownMenuItem
                    key={`del-${name}`}
                    variant="destructive"
                    onClick={() =>
                      setPending({
                        type: "delete-branch",
                        name,
                        title: name,
                      })
                    }
                  >
                    <Trash2 className="size-3.5" />
                    Apagar {name}
                  </DropdownMenuItem>
                ))}
              </>
            ) : null}
          </DropdownMenuContent>
        ) : null}
      </DropdownMenu>

      <AlertDialog
        open={pending != null}
        onOpenChange={(open) => {
          if (!open && !actionBusy) setPending(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{confirmCopy.title}</AlertDialogTitle>
            <AlertDialogDescription>{confirmCopy.description}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={actionBusy}>Cancelar</AlertDialogCancel>
            <AlertDialogAction
              disabled={actionBusy}
              variant={pending?.type === "reset" && pending.mode === "hard" ? "destructive" : "default"}
              onClick={(e) => {
                e.preventDefault()
                const action = pending
                if (!action) return
                void (async () => {
                  await onConfirmAction(action)
                  setPending(null)
                })()
              }}
            >
              {actionBusy ? <Loader2 className="size-3.5 animate-spin" /> : null}
              Confirmar
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function TimelineRow({
  event,
  disabled,
  onOpen,
  onContextMenu,
}: {
  event: TimelineEventView
  disabled?: boolean
  onOpen: () => void
  onContextMenu: (e: MouseEvent) => void
}) {
  const when = formatWhen(event.at)
  const isPR = event.kind.startsWith("pr_")
  const url = event.url

  return (
    <li className="relative pb-3 last:pb-1">
      <span className="absolute -left-[1.28rem] top-1 flex size-5 items-center justify-center rounded-full border bg-background">
        <KindIcon kind={event.kind} />
      </span>
      <div
        role="button"
        tabIndex={0}
        onClick={onOpen}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault()
            onOpen()
          }
        }}
        onContextMenu={onContextMenu}
        className={cn(
          "flex w-full cursor-pointer items-start gap-2 rounded-md px-1 py-1 hover:bg-muted/40",
          disabled && "pointer-events-none opacity-60",
        )}
      >
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-1.5">
            <Badge variant="outline" className="h-5 px-1.5 text-[10px] font-normal">
              {kindLabel(event.kind)}
            </Badge>
            <span className="text-[10px] text-muted-foreground" title={when.absolute}>
              {when.relative || when.absolute}
            </span>
          </div>
          <p className="mt-0.5 line-clamp-2 text-sm font-medium text-foreground">{event.title}</p>
          <p className="font-mono text-[10px] text-muted-foreground">
            {[event.subtitle, event.author].filter(Boolean).join(" · ")}
          </p>
        </div>
        {isPR && url ? (
          <ExternalLink className="mt-1 size-3.5 shrink-0 text-sky-500" />
        ) : null}
      </div>
    </li>
  )
}

