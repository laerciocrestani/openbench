import { useEffect, useMemo, useRef, useState } from "react"

import { AppService } from "../../bindings/github.com/laerciocrestani/openbench"
import type { ChatMessageView } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
import { Events } from "@wailsio/runtime"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Bubble, BubbleContent } from "@/components/ui/bubble"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import {
  Message,
  MessageAvatar,
  MessageContent,
  MessageFooter,
} from "@/components/ui/message"
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@/components/ui/message-scroller"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Bot, CircleDollarSign, Loader2, Send, ShieldAlert, Square, Trash2 } from "lucide-react"

type ChatDonePayload = {
  content?: string
  model?: string
  promptTokens?: number
  completionTokens?: number
  totalTokens?: number
  costUSD?: number | null
  usageLine?: string
}

type ChatToolRequest = {
  id?: string
  name?: string
  summary?: string
  path?: string
  command?: string
  contentPreview?: string
}

type LocalMessage = {
  id: string
  role: "user" | "assistant"
  content: string
  usageLine?: string
}

type SessionUsage = {
  turns: number
  promptTokens: number
  completionTokens: number
  totalTokens: number
  costUSD: number
  hasCost: boolean
}

const emptySession = (): SessionUsage => ({
  turns: 0,
  promptTokens: 0,
  completionTokens: 0,
  totalTokens: 0,
  costUSD: 0,
  hasCost: false,
})

function formatTokensShort(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 10_000) return `${Math.round(n / 1000)}k`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return n.toLocaleString("pt-BR")
}

function formatSessionCost(n: number): string {
  return `$${n.toFixed(6)}`
}

function newID(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function eventData<T>(ev: unknown): T | null {
  if (ev == null) return null
  if (typeof ev === "object" && ev !== null && "data" in ev) {
    return (ev as { data: T }).data ?? null
  }
  return ev as T
}

export function ProjectChatPanel({
  projectPath,
  visible,
}: {
  projectPath: string
  visible: boolean
}) {
  const [messages, setMessages] = useState<LocalMessage[]>([])
  const [draft, setDraft] = useState("")
  const [streaming, setStreaming] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [toolReq, setToolReq] = useState<ChatToolRequest | null>(null)
  const [toolBusy, setToolBusy] = useState(false)
  const [sessionUsage, setSessionUsage] = useState<SessionUsage>(emptySession)
  const [models, setModels] = useState<string[]>([])
  const [model, setModel] = useState("")
  const assistantID = useRef<string | null>(null)
  const projectRef = useRef(projectPath)
  const decidingTool = useRef(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await AppService.GetChatModels()
        if (cancelled || !res) return
        const list = (res.models ?? []).filter(Boolean)
        const def = (res.defaultModel || list[0] || "").trim()
        setModels(list.length > 0 ? list : def ? [def] : [])
        setModel(def)
      } catch {
        // keep empty — send still works with backend default
      }
    })()
    return () => {
      cancelled = true
    }
  }, [visible, projectPath])

  useEffect(() => {
    if (projectRef.current === projectPath) return
    projectRef.current = projectPath
    setMessages([])
    setDraft("")
    setError(null)
    setStreaming(false)
    setToolReq(null)
    setSessionUsage(emptySession())
    assistantID.current = null
    void AppService.ChatCancel()
  }, [projectPath])

  useEffect(() => {
    if (!visible) return

    const offChunk = Events.On("chat:chunk", (ev) => {
      const delta = String(eventData<string>(ev) ?? "")
      if (!delta) return
      const id = assistantID.current
      if (!id) return
      setMessages((prev) =>
        prev.map((m) => (m.id === id ? { ...m, content: m.content + delta } : m)),
      )
    })

    const offTool = Events.On("chat:tool_request", (ev) => {
      const req = eventData<ChatToolRequest>(ev)
      if (!req) return
      setToolReq(req)
      setToolBusy(false)
    })

    const offDone = Events.On("chat:done", (ev) => {
      const payload = eventData<ChatDonePayload>(ev)
      const id = assistantID.current
      setStreaming(false)
      setToolReq(null)
      assistantID.current = null

      const prompt = payload?.promptTokens ?? 0
      const completion = payload?.completionTokens ?? 0
      const total = payload?.totalTokens ?? prompt + completion
      const cost = payload?.costUSD
      if (prompt > 0 || completion > 0 || total > 0 || (cost != null && cost > 0)) {
        setSessionUsage((prev) => ({
          turns: prev.turns + 1,
          promptTokens: prev.promptTokens + prompt,
          completionTokens: prev.completionTokens + completion,
          totalTokens: prev.totalTokens + total,
          costUSD: prev.costUSD + (typeof cost === "number" ? cost : 0),
          hasCost: prev.hasCost || typeof cost === "number",
        }))
      }

      if (!id) return
      setMessages((prev) =>
        prev.map((m) => {
          if (m.id !== id) return m
          return {
            ...m,
            content: payload?.content?.trim() ? payload.content : m.content,
            usageLine: payload?.usageLine || undefined,
          }
        }),
      )
    })

    const offError = Events.On("chat:error", (ev) => {
      const msg = String(eventData<string>(ev) ?? "erro no chat")
      setStreaming(false)
      setToolReq(null)
      assistantID.current = null
      setError(msg)
    })

    return () => {
      offChunk()
      offTool()
      offDone()
      offError()
    }
  }, [visible])

  const historyForAPI = useMemo((): ChatMessageView[] => {
    return messages
      .filter((m) => m.content.trim() !== "")
      .map((m) => ({ role: m.role, content: m.content }))
  }, [messages])

  const send = async () => {
    const text = draft.trim()
    if (!text || streaming) return
    setError(null)
    setDraft("")

    const userMsg: LocalMessage = { id: newID("u"), role: "user", content: text }
    const asstID = newID("a")
    assistantID.current = asstID
    const history = historyForAPI
    setMessages((prev) => [
      ...prev,
      userMsg,
      { id: asstID, role: "assistant", content: "" },
    ])
    setStreaming(true)
    try {
      await AppService.ChatStream(text, history, model)
    } catch (e) {
      setStreaming(false)
      assistantID.current = null
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  const cancel = () => {
    void AppService.ChatCancel()
    setStreaming(false)
    setToolReq(null)
    assistantID.current = null
  }

  const clear = () => {
    if (streaming) cancel()
    setMessages([])
    setError(null)
    setToolReq(null)
    setSessionUsage(emptySession())
  }

  const approveTool = async () => {
    if (!toolReq?.id || toolBusy) return
    decidingTool.current = true
    setToolBusy(true)
    try {
      await AppService.ChatApproveTool(toolReq.id)
      setToolReq(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setToolBusy(false)
    } finally {
      decidingTool.current = false
    }
  }

  const denyTool = async () => {
    if (!toolReq?.id || toolBusy) return
    decidingTool.current = true
    setToolBusy(true)
    try {
      await AppService.ChatDenyTool(toolReq.id)
      setToolReq(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setToolBusy(false)
    } finally {
      decidingTool.current = false
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col bg-background">
      <div className="flex shrink-0 items-center gap-1.5 border-b px-2 py-1.5">
        <Bot className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="shrink-0 text-xs font-medium">Chat IA</span>
        {sessionUsage.turns > 0 && (
          <>
            <span className="truncate text-[10px] tabular-nums text-muted-foreground">
              {formatTokensShort(sessionUsage.totalTokens)} tok
              {sessionUsage.hasCost ? ` · ${formatSessionCost(sessionUsage.costUSD)}` : ""}
            </span>
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <button
                    type="button"
                    className="inline-flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
                    aria-label="Gasto da sessão"
                  />
                }
              >
                <CircleDollarSign className="size-3.5" />
              </TooltipTrigger>
              <TooltipContent
                side="bottom"
                align="start"
                className="flex max-w-xs flex-col gap-1 px-3 py-2 text-left leading-snug"
              >
                <p className="font-medium">Gasto da sessão</p>
                <p>
                  {sessionUsage.turns} turno{sessionUsage.turns === 1 ? "" : "s"} neste chat
                </p>
                <p className="tabular-nums">
                  {sessionUsage.promptTokens.toLocaleString("pt-BR")} input +{" "}
                  {sessionUsage.completionTokens.toLocaleString("pt-BR")} output ={" "}
                  {sessionUsage.totalTokens.toLocaleString("pt-BR")} tokens
                </p>
                <p className="tabular-nums">
                  {sessionUsage.hasCost
                    ? `${formatSessionCost(sessionUsage.costUSD)} USD`
                    : "Custo não informado pelo provider"}
                </p>
              </TooltipContent>
            </Tooltip>
          </>
        )}
        <span className="min-w-0 flex-1" />
        <span className="truncate text-[10px] text-muted-foreground" title={projectPath}>
          {projectPath.split(/[/\\]/).pop()}
        </span>
        <Button
          variant="ghost"
          size="icon-xs"
          title="Limpar conversa"
          disabled={messages.length === 0 && !streaming && sessionUsage.turns === 0}
          onClick={clear}
        >
          <Trash2 />
        </Button>
      </div>

      <div className="min-h-0 flex-1">
        <MessageScrollerProvider autoScroll>
          <MessageScroller>
            <MessageScrollerViewport>
              <MessageScrollerContent className="gap-3 p-2">
                {messages.length === 0 && (
                  <p className="px-2 py-6 text-center text-xs text-muted-foreground">
                    Pergunte sobre o projeto ou peça alterações — writes e comandos pedem
                    sua aprovação.
                  </p>
                )}
                {messages.map((m) => (
                  <MessageScrollerItem
                    key={m.id}
                    messageId={m.id}
                    scrollAnchor={m.role === "user"}
                  >
                    <Message align={m.role === "user" ? "end" : "start"}>
                      <MessageAvatar>
                        <Avatar className="size-7">
                          <AvatarFallback className="text-[10px]">
                            {m.role === "user" ? "EU" : "AI"}
                          </AvatarFallback>
                        </Avatar>
                      </MessageAvatar>
                      <MessageContent>
                        <Bubble variant={m.role === "user" ? "default" : "muted"}>
                          <BubbleContent className="whitespace-pre-wrap break-words text-xs leading-relaxed">
                            {m.content ||
                              (streaming && m.id === assistantID.current ? "…" : "")}
                          </BubbleContent>
                        </Bubble>
                        {m.usageLine && (
                          <MessageFooter className="text-[10px] text-muted-foreground">
                            {m.usageLine}
                          </MessageFooter>
                        )}
                      </MessageContent>
                    </Message>
                  </MessageScrollerItem>
                ))}
              </MessageScrollerContent>
            </MessageScrollerViewport>
            <MessageScrollerButton />
          </MessageScroller>
        </MessageScrollerProvider>
      </div>

      {error && (
        <div className="shrink-0 border-t px-2 py-1.5 text-[11px] text-destructive">{error}</div>
      )}

      <form
        className="flex shrink-0 flex-col gap-1.5 border-t p-2"
        onSubmit={(e) => {
          e.preventDefault()
          void send()
        }}
      >
        <Textarea
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault()
              void send()
            }
          }}
          placeholder="Pergunte à IA…"
          disabled={streaming}
          rows={3}
          className="min-h-[4.5rem] resize-none field-sizing-fixed text-xs"
        />
        <div className="flex items-center gap-1.5">
          {models.length > 0 && model ? (
            <Select
              key={model}
              value={model}
              onValueChange={(v) => {
                const next = String(v ?? "")
                if (next) setModel(next)
              }}
              disabled={streaming}
            >
              <SelectTrigger size="sm" className="h-8 min-w-0 flex-1 text-xs">
                <SelectValue placeholder="Modelo">{model}</SelectValue>
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false} className="max-w-[min(100vw-2rem,24rem)]">
                {models.map((m) => (
                  <SelectItem key={m} value={m} className="text-xs font-mono">
                    {m}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <div className="flex h-8 min-w-0 flex-1 items-center truncate rounded-lg border border-input px-2 text-xs text-muted-foreground">
              {model || "Carregando modelo…"}
            </div>
          )}
          {streaming ? (
            <Button type="button" size="icon-sm" variant="outline" onClick={cancel} title="Parar">
              <Square />
            </Button>
          ) : (
            <Button type="submit" size="icon-sm" disabled={!draft.trim()} title="Enviar">
              <Send />
            </Button>
          )}
        </div>
      </form>

      <AlertDialog
        open={toolReq != null}
        onOpenChange={(open) => {
          if (!open && toolReq && !toolBusy && !decidingTool.current) {
            void denyTool()
          }
        }}
      >
        <AlertDialogContent className="flex max-h-[min(90vh,36rem)] max-w-[calc(100%-2rem)] flex-col gap-3 data-[size=default]:max-w-[calc(100%-2rem)] data-[size=default]:sm:max-w-lg">
          <AlertDialogHeader className="shrink-0">
            <AlertDialogMedia className="bg-amber-500/15 text-amber-700 dark:text-amber-400">
              <ShieldAlert />
            </AlertDialogMedia>
            <AlertDialogTitle>Permissão da IA</AlertDialogTitle>
            <AlertDialogDescription>
              {toolReq?.summary || "A IA pediu para executar uma ação no projeto."}
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="min-h-0 flex-1 space-y-2 overflow-auto text-xs">
            {toolReq?.path && (
              <div>
                <div className="mb-1 font-medium text-muted-foreground">Arquivo</div>
                <code className="block rounded-md bg-muted px-2 py-1.5 break-all">
                  {toolReq.path}
                </code>
              </div>
            )}
            {toolReq?.command && (
              <div>
                <div className="mb-1 font-medium text-muted-foreground">Comando</div>
                <code className="block rounded-md bg-muted px-2 py-1.5 break-all">
                  {toolReq.command}
                </code>
              </div>
            )}
            {toolReq?.contentPreview != null && toolReq.contentPreview !== "" && (
              <div>
                <div className="mb-1 font-medium text-muted-foreground">Conteúdo</div>
                <pre className="max-h-48 overflow-auto rounded-md bg-muted px-2 py-1.5 whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed">
                  {toolReq.contentPreview}
                </pre>
              </div>
            )}
          </div>

          <AlertDialogFooter className="shrink-0">
            <AlertDialogCancel disabled={toolBusy}>Negar</AlertDialogCancel>
            <AlertDialogAction
              disabled={toolBusy}
              onClick={(e) => {
                e.preventDefault()
                void approveTool()
              }}
            >
              {toolBusy ? <Loader2 className="animate-spin" /> : "Permitir"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
