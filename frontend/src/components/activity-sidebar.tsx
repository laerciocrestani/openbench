import { useCallback, useEffect, useState, type ReactNode } from "react"

import { cn } from "@/lib/utils"

const ACTIVITY_OPEN_KEY = "openbench.activitySidebar.open"
const ACTIVITY_WIDTH = "22rem"

function loadOpen(defaultOpen: boolean): boolean {
  try {
    const raw = localStorage.getItem(ACTIVITY_OPEN_KEY)
    if (raw === null) return defaultOpen
    return raw === "1" || raw === "true"
  } catch {
    return defaultOpen
  }
}

function saveOpen(open: boolean) {
  try {
    localStorage.setItem(ACTIVITY_OPEN_KEY, open ? "1" : "0")
  } catch {
    /* ignore */
  }
}

/** Left column for activity timeline + calendar (below the full-width header). */
export function useActivitySidebar(defaultOpen = true) {
  const [open, setOpenState] = useState(defaultOpen)

  useEffect(() => {
    setOpenState(loadOpen(defaultOpen))
  }, [defaultOpen])

  const setOpen = useCallback((next: boolean | ((v: boolean) => boolean)) => {
    setOpenState((prev) => {
      const value = typeof next === "function" ? next(prev) : next
      saveOpen(value)
      return value
    })
  }, [])

  const toggle = useCallback(() => {
    setOpen((v) => !v)
  }, [setOpen])

  return { open, setOpen, toggle }
}

export function ActivitySidebar({
  open,
  children,
  className,
}: {
  open: boolean
  children: ReactNode
  className?: string
}) {
  if (!open) return null

  return (
    <aside
      data-slot="activity-sidebar"
      className={cn(
        "flex h-full min-h-0 w-(--activity-sidebar-width) shrink-0 flex-col border-r bg-sidebar text-sidebar-foreground",
        className,
      )}
      style={{ ["--activity-sidebar-width" as string]: ACTIVITY_WIDTH }}
    >
      {children}
    </aside>
  )
}
