import { useEffect, useRef, useState, type ReactNode } from "react"
import { Terminal } from "@xterm/xterm"
import { FitAddon } from "@xterm/addon-fit"
import { WebLinksAddon } from "@xterm/addon-web-links"
import "@xterm/xterm/css/xterm.css"

import { AppService } from "../../bindings/github.com/laerciocrestani/openbench"
import { Events } from "@wailsio/runtime"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { Plus, RotateCcw, TerminalSquare, X } from "lucide-react"

export type TerminalSessionSpec =
  | { kind: "host" }
  | { kind: "docker"; service: string; presetId?: string }

export type DockerShellRequest = {
  service: string
  presetId?: string
  /** Monotonic key so the same service can be re-requested after close. */
  key: number
}

const MAX_TABS = 8

type Tab = {
  tabId: string
  spec: TerminalSessionSpec
}

function decodeBase64(b64: string): string {
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return new TextDecoder("utf-8", { fatal: false }).decode(bytes)
}

function newTabId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID()
  }
  return `tab-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function hostTab(): Tab {
  return { tabId: newTabId(), spec: { kind: "host" } }
}

/** Basename of the project path, or "~" when none — easier to scan than "host". */
function hostBaseLabel(projectPath: string | null): string {
  if (!projectPath) return "~"
  const parts = projectPath.replace(/\\/g, "/").split("/").filter(Boolean)
  return parts[parts.length - 1] || "~"
}

function sessionLabel(
  session: TerminalSessionSpec,
  projectPath: string | null,
  hostOrdinal?: number
): string {
  if (session.kind === "docker") {
    const extra = session.presetId ? ` · ${session.presetId}` : ""
    return `${session.service}${extra}`
  }
  const base = hostBaseLabel(projectPath)
  if (hostOrdinal && hostOrdinal > 1) return `${base} · ${hostOrdinal}`
  return base
}

function dockerMatch(a: TerminalSessionSpec, service: string, presetId?: string): boolean {
  return (
    a.kind === "docker" &&
    a.service === service &&
    (a.presetId ?? "") === (presetId?.trim() || "")
  )
}

function parseTermEvent(raw: unknown): { id: string; data?: string } | null {
  const s = String(raw ?? "")
  if (!s) return null
  try {
    const parsed = JSON.parse(s) as { id?: string; data?: string }
    if (!parsed?.id) return null
    return { id: parsed.id, data: parsed.data }
  } catch {
    return null
  }
}

function TerminalPane({
  projectPath,
  visible,
  active,
  spec,
}: {
  projectPath: string | null
  visible: boolean
  active: boolean
  spec: TerminalSessionSpec
}) {
  const hostRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const sessionIdRef = useRef<string | null>(null)
  const startedRef = useRef(false)

  useEffect(() => {
    const host = hostRef.current
    if (!host || termRef.current) return

    const term = new Terminal({
      cursorBlink: true,
      cursorStyle: "block",
      disableStdin: false,
      fontSize: 12,
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
      theme: {
        background: "#0c0c0c",
        foreground: "#e5e5e5",
        cursor: "#e5e5e5",
        selectionBackground: "#264f78",
      },
      allowProposedApi: true,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.loadAddon(new WebLinksAddon())
    term.open(host)
    fit.fit()

    // WebView/browser steals Tab for focus navigation; keep it in the PTY
    // so shell completion (cd <Tab>, etc.) works like a real terminal.
    term.attachCustomKeyEventHandler((ev) => {
      if (ev.type === "keydown" && ev.key === "Tab") {
        ev.preventDefault()
      }
      return true
    })

    term.onData((data) => {
      const id = sessionIdRef.current
      if (!id) return
      void AppService.TerminalWrite(id, data).catch(() => {
        /* PTY still starting / already closed */
      })
    })

    termRef.current = term
    fitRef.current = fit

    const onResize = () => {
      if (!fitRef.current || !termRef.current) return
      fitRef.current.fit()
      const dims = fitRef.current.proposeDimensions()
      const id = sessionIdRef.current
      if (dims && id) {
        void AppService.TerminalResize(id, dims.cols, dims.rows).catch(() => {})
      }
    }
    const ro = new ResizeObserver(() => onResize())
    ro.observe(host)

    const focusTerm = () => term.focus()
    host.addEventListener("mousedown", focusTerm)

    return () => {
      host.removeEventListener("mousedown", focusTerm)
      ro.disconnect()
      const id = sessionIdRef.current
      if (id) {
        void AppService.TerminalStop(id)
        sessionIdRef.current = null
      }
      term.dispose()
      termRef.current = null
      fitRef.current = null
      startedRef.current = false
    }
  }, [])

  useEffect(() => {
    const offData = Events.On("terminal:data", (ev) => {
      const parsed = parseTermEvent(ev?.data)
      if (!parsed?.data || parsed.id !== sessionIdRef.current || !termRef.current) return
      try {
        termRef.current.write(decodeBase64(parsed.data))
      } catch {
        /* ignore bad chunk */
      }
    })
    const offExit = Events.On("terminal:exit", (ev) => {
      const parsed = parseTermEvent(ev?.data)
      if (!parsed || parsed.id !== sessionIdRef.current) return
      sessionIdRef.current = null
      startedRef.current = false
      termRef.current?.writeln("\r\n\x1b[90m[processo encerrado]\x1b[0m")
    })
    return () => {
      offData()
      offExit()
    }
  }, [])

  useEffect(() => {
    // Inactive panes must leave the tab order, otherwise Tab jumps between
    // hidden xterm textareas instead of completing in the active shell.
    const textarea = hostRef.current?.querySelector("textarea.xterm-helper-textarea")
    if (textarea instanceof HTMLTextAreaElement) {
      textarea.tabIndex = active ? 0 : -1
    }
    if (!visible || !active || !termRef.current || !fitRef.current) return
    fitRef.current.fit()
    const dims = fitRef.current.proposeDimensions()
    const id = sessionIdRef.current
    if (dims && id) {
      void AppService.TerminalResize(id, dims.cols, dims.rows).catch(() => {})
    }
    requestAnimationFrame(() => termRef.current?.focus())
  }, [visible, active])

  useEffect(() => {
    if (!visible || !termRef.current || !fitRef.current) return
    if (spec.kind === "docker" && !projectPath) return

    let cancelled = false
    let startedId: string | null = null

    const start = async () => {
      fitRef.current?.fit()
      const dims = fitRef.current?.proposeDimensions()
      const cols = dims?.cols ?? 80
      const rows = dims?.rows ?? 24
      termRef.current?.reset()
      try {
        const id =
          spec.kind === "docker"
            ? await AppService.DockerShellStart(
                spec.service,
                cols,
                rows,
                spec.presetId ?? ""
              )
            : await AppService.TerminalStart(cols, rows)
        if (cancelled) {
          void AppService.TerminalStop(id)
          return
        }
        startedId = id
        sessionIdRef.current = id
        startedRef.current = true
        requestAnimationFrame(() => termRef.current?.focus())
      } catch (e) {
        if (cancelled) return
        const msg = e instanceof Error ? e.message : String(e)
        termRef.current?.writeln(`\x1b[31m${msg}\x1b[0m`)
        startedRef.current = false
        sessionIdRef.current = null
      }
    }
    void start()

    return () => {
      cancelled = true
      const id = startedId ?? sessionIdRef.current
      if (id) {
        void AppService.TerminalStop(id)
        if (sessionIdRef.current === id) {
          sessionIdRef.current = null
          startedRef.current = false
        }
      }
    }
    // Do not depend on `active`: switching tabs must not spawn extra PTYs.
  }, [visible, projectPath, spec])

  const restart = async () => {
    if (!fitRef.current || !termRef.current) return
    if (spec.kind === "docker" && !projectPath) return
    fitRef.current.fit()
    const dims = fitRef.current.proposeDimensions()
    const cols = dims?.cols ?? 80
    const rows = dims?.rows ?? 24
    termRef.current.reset()
    try {
      const existing = sessionIdRef.current
      let id: string
      if (existing) {
        id = await AppService.TerminalRestart(existing, cols, rows)
      } else if (spec.kind === "docker") {
        id = await AppService.DockerShellStart(
          spec.service,
          cols,
          rows,
          spec.presetId ?? ""
        )
      } else {
        id = await AppService.TerminalStart(cols, rows)
      }
      sessionIdRef.current = id
      startedRef.current = true
      requestAnimationFrame(() => termRef.current?.focus())
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      termRef.current.writeln(`\x1b[31m${msg}\x1b[0m`)
    }
  }

  return (
    <div
      className={cn(
        "absolute inset-0 flex min-h-0 flex-col",
        !active && "invisible pointer-events-none"
      )}
      aria-hidden={!active}
      inert={!active ? true : undefined}
    >
      <div className="absolute top-1.5 right-1.5 z-10">
        <Button
          variant="secondary"
          size="icon-xs"
          className="size-6 bg-background/80 shadow-sm backdrop-blur-sm"
          title="Reiniciar sessão"
          disabled={spec.kind === "docker" && !projectPath}
          onClick={() => void restart()}
        >
          <RotateCcw className="size-3" />
        </Button>
      </div>

      <div
        ref={hostRef}
        className="min-h-0 flex-1 cursor-text overflow-hidden bg-[#0c0c0c] p-1 [&_.xterm]:h-full [&_.xterm-screen]:h-full [&_textarea]:pointer-events-auto"
        onMouseDown={() => termRef.current?.focus()}
      />

      {spec.kind === "docker" && !projectPath && (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-[#0c0c0c]/95 p-6 text-center text-sm text-muted-foreground">
          <TerminalSquare className="size-8 opacity-40" />
          <p>Abra um projeto para usar o shell no container Docker.</p>
        </div>
      )}
    </div>
  )
}

export type TerminalPanelSlots = {
  toolbar: ReactNode
  body: ReactNode
}

export function TerminalPanel({
  projectPath,
  visible,
  dockerRequest,
  children,
}: {
  projectPath: string | null
  visible: boolean
  dockerRequest?: DockerShellRequest | null
  children: (slots: TerminalPanelSlots) => ReactNode
}) {
  const initial = useRef<Tab | null>(null)
  if (!initial.current) initial.current = hostTab()
  const [tabs, setTabs] = useState<Tab[]>(() => [initial.current!])
  const [activeId, setActiveId] = useState(() => initial.current!.tabId)
  const lastDockerKey = useRef<number | null>(null)
  const pathKey = useRef<string | null | undefined>(undefined)

  // Reset tabs when project changes (backend stops all PTYs).
  useEffect(() => {
    if (pathKey.current === undefined) {
      pathKey.current = projectPath
      return
    }
    if (pathKey.current === projectPath) return
    pathKey.current = projectPath
    const t = hostTab()
    setTabs([t])
    setActiveId(t.tabId)
    lastDockerKey.current = null
  }, [projectPath])

  // Open / focus docker tab from parent request.
  useEffect(() => {
    if (!dockerRequest) return
    if (lastDockerKey.current === dockerRequest.key) return
    lastDockerKey.current = dockerRequest.key

    const service = dockerRequest.service.trim()
    if (!service) return
    const presetId = dockerRequest.presetId?.trim() || undefined

    setTabs((prev) => {
      const existing = prev.find((t) => dockerMatch(t.spec, service, presetId))
      if (existing) {
        queueMicrotask(() => setActiveId(existing.tabId))
        return prev
      }
      if (prev.length >= MAX_TABS) return prev
      const tab: Tab = {
        tabId: newTabId(),
        spec: { kind: "docker", service, presetId },
      }
      queueMicrotask(() => setActiveId(tab.tabId))
      return [...prev, tab]
    })
  }, [dockerRequest])

  const addHostTab = () => {
    if (tabs.length >= MAX_TABS) return
    const tab = hostTab()
    setTabs((prev) => [...prev, tab])
    setActiveId(tab.tabId)
  }

  const closeTab = (tabId: string) => {
    if (tabs.length <= 1) {
      const t = hostTab()
      setTabs([t])
      setActiveId(t.tabId)
      return
    }
    const idx = tabs.findIndex((t) => t.tabId === tabId)
    if (idx < 0) return
    const next = tabs.filter((t) => t.tabId !== tabId)
    setTabs(next)
    if (activeId === tabId) {
      const fallback = next[Math.max(0, idx - 1)] ?? next[0]
      setActiveId(fallback.tabId)
    }
  }

  const hostIndex = (tab: Tab, index: number) => {
    if (tab.spec.kind !== "host") return null
    let n = 0
    for (let i = 0; i <= index; i++) {
      if (tabs[i]?.spec.kind === "host") n++
    }
    return n
  }

  const toolbar = (
    <div className="flex min-w-0 flex-1 items-center gap-0.5">
      <div className="flex min-w-0 flex-1 items-center gap-0.5 overflow-x-auto">
        {tabs.map((tab, i) => {
          const hi = hostIndex(tab, i)
          const label = sessionLabel(
            tab.spec,
            projectPath,
            hi ?? undefined
          )
          const title =
            tab.spec.kind === "host"
              ? projectPath || "~"
              : label
          const active = tab.tabId === activeId
          return (
            <div
              key={tab.tabId}
              className={cn(
                "group flex h-7 max-w-[9rem] shrink-0 items-center gap-0.5 rounded-md px-1.5 text-[11px]",
                active
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
              )}
            >
              <button
                type="button"
                className="min-w-0 flex-1 truncate text-left font-mono"
                title={title}
                onClick={() => setActiveId(tab.tabId)}
              >
                {label}
              </button>
              <button
                type="button"
                className={cn(
                  "inline-flex size-4 shrink-0 items-center justify-center rounded-sm opacity-0 hover:bg-background/80 group-hover:opacity-100",
                  active && "opacity-70"
                )}
                title="Fechar aba"
                onClick={(e) => {
                  e.stopPropagation()
                  closeTab(tab.tabId)
                }}
              >
                <X className="size-3" />
              </button>
            </div>
          )
        })}
      </div>
      <Button
        variant="ghost"
        size="icon-xs"
        className="shrink-0"
        title={
          tabs.length >= MAX_TABS
            ? `Limite de ${MAX_TABS} terminais`
            : "Novo terminal (host)"
        }
        disabled={tabs.length >= MAX_TABS}
        onClick={addHostTab}
      >
        <Plus />
      </Button>
    </div>
  )

  const body = (
    <div className="relative h-full min-h-0 overflow-hidden rounded-lg ring-1 ring-foreground/10">
      {tabs.map((tab) => (
        <TerminalPane
          key={tab.tabId}
          projectPath={projectPath}
          visible={visible}
          active={tab.tabId === activeId}
          spec={tab.spec}
        />
      ))}
    </div>
  )

  return <>{children({ toolbar, body })}</>
}
