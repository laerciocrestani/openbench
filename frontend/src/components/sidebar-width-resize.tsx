import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type PointerEvent,
} from "react"

const WIDTH_KEY = "openbench.sidebar.widthPx"
const DEFAULT_WIDTH_PX = 448 // 28rem — terminal + chat
const MIN_WIDTH_PX = 320
const MAX_WIDTH_PX = 720

function clampWidth(px: number): number {
  const max = Math.min(MAX_WIDTH_PX, Math.floor(window.innerWidth * 0.55))
  const min = Math.min(MIN_WIDTH_PX, max)
  return Math.min(max, Math.max(min, Math.round(px)))
}

function loadWidthPx(): number {
  try {
    const raw = localStorage.getItem(WIDTH_KEY)
    if (!raw) return DEFAULT_WIDTH_PX
    const n = Number(raw)
    if (!Number.isFinite(n)) return DEFAULT_WIDTH_PX
    return clampWidth(n)
  } catch {
    return DEFAULT_WIDTH_PX
  }
}

function saveWidthPx(px: number) {
  try {
    localStorage.setItem(WIDTH_KEY, String(px))
  } catch {
    /* ignore */
  }
}

function sidebarWrapper(): HTMLElement | null {
  return document.querySelector<HTMLElement>('[data-slot="sidebar-wrapper"]')
}

/** Aplica largura direto no CSS var — sem re-render do App. */
function applyWidthPx(px: number) {
  const el = sidebarWrapper()
  if (el) el.style.setProperty("--sidebar-width", `${px}px`)
}

function setResizing(active: boolean) {
  const el = sidebarWrapper()
  if (!el) return
  if (active) el.setAttribute("data-resizing", "true")
  else el.removeAttribute("data-resizing")
}

/** Hook: largura redimensionável da sidebar (CSS --sidebar-width). */
export function useSidebarWidth() {
  const [widthPx, setWidthPx] = useState(DEFAULT_WIDTH_PX)

  useEffect(() => {
    const initial = loadWidthPx()
    setWidthPx(initial)
    applyWidthPx(initial)
  }, [])

  useEffect(() => {
    const onResize = () => {
      setWidthPx((w) => {
        const next = clampWidth(w)
        applyWidthPx(next)
        return next
      })
    }
    window.addEventListener("resize", onResize)
    return () => window.removeEventListener("resize", onResize)
  }, [])

  const commitWidth = useCallback((px: number) => {
    const next = clampWidth(px)
    applyWidthPx(next)
    setWidthPx(next)
    saveWidthPx(next)
  }, [])

  const style = {
    "--sidebar-width": `${widthPx}px`,
  } as CSSProperties

  return { widthPx, commitWidth, style }
}

/**
 * Handle na borda esquerda da sidebar direita.
 * Durante o drag atualiza só o CSS var (sem setState) e desliga a transition.
 */
export function SidebarWidthRail({
  widthPx,
  onCommitWidth,
}: {
  widthPx: number
  onCommitWidth: (px: number) => void
}) {
  const dragging = useRef(false)
  const liveWidth = useRef(widthPx)
  liveWidth.current = widthPx

  const onPointerDown = useCallback((e: PointerEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    dragging.current = true
    setResizing(true)
    e.currentTarget.setPointerCapture(e.pointerId)
  }, [])

  const onPointerMove = useCallback((e: PointerEvent<HTMLDivElement>) => {
    if (!dragging.current) return
    const next = clampWidth(window.innerWidth - e.clientX)
    liveWidth.current = next
    applyWidthPx(next)
  }, [])

  const endDrag = useCallback(
    (e: PointerEvent<HTMLDivElement>) => {
      if (!dragging.current) return
      dragging.current = false
      setResizing(false)
      try {
        e.currentTarget.releasePointerCapture(e.pointerId)
      } catch {
        /* ignore */
      }
      onCommitWidth(liveWidth.current)
    },
    [onCommitWidth],
  )

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Redimensionar largura do terminal"
      aria-valuenow={widthPx}
      tabIndex={-1}
      title="Arraste para redimensionar · duplo clique restaura"
      className="absolute inset-y-0 left-0 z-20 hidden w-1.5 cursor-col-resize touch-none sm:flex hover:bg-sidebar-border/80 active:bg-sidebar-border"
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={endDrag}
      onPointerCancel={endDrag}
      onDoubleClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        onCommitWidth(DEFAULT_WIDTH_PX)
      }}
    />
  )
}
