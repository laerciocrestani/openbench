import { useCallback, useEffect, useRef, useState, type PointerEvent, type ReactNode } from "react"

import { cn } from "@/lib/utils"

const CHAT_RATIO_KEY = "openbench.sidebar.chatRatio"
const DEFAULT_CHAT_RATIO = 0.3
const MIN_CHAT_RATIO = 0.18
const MAX_CHAT_RATIO = 0.7

function loadChatRatio(): number {
  try {
    const raw = localStorage.getItem(CHAT_RATIO_KEY)
    if (!raw) return DEFAULT_CHAT_RATIO
    const n = Number(raw)
    if (!Number.isFinite(n)) return DEFAULT_CHAT_RATIO
    return Math.min(MAX_CHAT_RATIO, Math.max(MIN_CHAT_RATIO, n))
  } catch {
    return DEFAULT_CHAT_RATIO
  }
}

export function TerminalChatSplit({
  showChat,
  terminal,
  chat,
}: {
  showChat: boolean
  terminal: ReactNode
  chat: ReactNode
}) {
  const [chatRatio, setChatRatio] = useState(DEFAULT_CHAT_RATIO)
  const rootRef = useRef<HTMLDivElement>(null)
  const dragging = useRef(false)

  useEffect(() => {
    setChatRatio(loadChatRatio())
  }, [])

  const onPointerDown = useCallback((e: PointerEvent<HTMLDivElement>) => {
    e.preventDefault()
    dragging.current = true
    e.currentTarget.setPointerCapture(e.pointerId)
  }, [])

  const onPointerMove = useCallback((e: PointerEvent<HTMLDivElement>) => {
    if (!dragging.current || !rootRef.current) return
    const rect = rootRef.current.getBoundingClientRect()
    if (rect.height <= 0) return
    const fromBottom = rect.bottom - e.clientY
    const next = Math.min(MAX_CHAT_RATIO, Math.max(MIN_CHAT_RATIO, fromBottom / rect.height))
    setChatRatio(next)
  }, [])

  const endDrag = useCallback((e: PointerEvent<HTMLDivElement>) => {
    if (!dragging.current) return
    dragging.current = false
    try {
      e.currentTarget.releasePointerCapture(e.pointerId)
    } catch {
      /* ignore */
    }
    setChatRatio((r) => {
      try {
        localStorage.setItem(CHAT_RATIO_KEY, String(r))
      } catch {
        /* ignore */
      }
      return r
    })
  }, [])

  if (!showChat) {
    return <div className="flex h-full min-h-0 flex-col">{terminal}</div>
  }

  const chatPct = `${(chatRatio * 100).toFixed(2)}%`
  const termPct = `${((1 - chatRatio) * 100).toFixed(2)}%`

  return (
    <div ref={rootRef} className="flex h-full min-h-0 flex-col">
      <div className="min-h-0 overflow-hidden" style={{ height: termPct }}>
        {terminal}
      </div>
      <div
        role="separator"
        aria-orientation="horizontal"
        aria-label="Redimensionar terminal e chat"
        className={cn(
          "group relative z-10 flex h-2 shrink-0 cursor-row-resize items-center justify-center border-y bg-muted/40",
          "hover:bg-muted",
        )}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={endDrag}
        onPointerCancel={endDrag}
      >
        <span className="h-0.5 w-8 rounded-full bg-border group-hover:bg-foreground/30" />
      </div>
      <div className="min-h-0 overflow-hidden" style={{ height: chatPct }}>
        {chat}
      </div>
    </div>
  )
}
