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

/** Independent left Activity sidebar (does not share terminal SidebarProvider state). */
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
  const state = open ? "expanded" : "collapsed"

  return (
    <div
      className={cn("group/activity peer hidden text-sidebar-foreground md:block", className)}
      data-state={state}
      data-collapsible={state === "collapsed" ? "offcanvas" : ""}
      data-side="left"
      data-slot="activity-sidebar"
      style={{ ["--activity-sidebar-width" as string]: ACTIVITY_WIDTH }}
    >
      <div
        data-slot="activity-sidebar-gap"
        className={cn(
          "relative w-(--activity-sidebar-width) bg-transparent transition-[width] duration-200 ease-linear",
          "group-data-[collapsible=offcanvas]/activity:w-0",
        )}
      />
      <div
        data-slot="activity-sidebar-container"
        className={cn(
          "fixed inset-y-0 left-0 z-10 hidden h-svh w-(--activity-sidebar-width) border-r bg-sidebar text-sidebar-foreground transition-[left] duration-200 ease-linear md:flex",
          "group-data-[collapsible=offcanvas]/activity:left-[calc(var(--activity-sidebar-width)*-1)]",
        )}
      >
        <div className="flex size-full flex-col">{children}</div>
      </div>
    </div>
  )
}
