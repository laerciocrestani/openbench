import { useMemo, useState } from "react"
import { format, parseISO } from "date-fns"
import { ptBR } from "date-fns/locale"

import type {
  CommitActivityView,
  DayActivityView,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
import { Calendar, CalendarDayButton } from "@/components/ui/calendar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { cn } from "@/lib/utils"
import { Loader2 } from "lucide-react"

function dayKey(d: Date): string {
  return format(d, "yyyy-MM-dd")
}

function intensityClass(count: number): string {
  if (count <= 0) {
    return "bg-muted/50 text-muted-foreground/70"
  }
  if (count === 1) {
    return "bg-emerald-500/30 text-emerald-900 dark:text-emerald-100"
  }
  if (count <= 3) {
    return "bg-emerald-500/55 text-emerald-950 dark:text-emerald-50"
  }
  if (count <= 6) {
    return "bg-emerald-600 text-primary-foreground"
  }
  return "bg-emerald-700 text-primary-foreground"
}

export function CommitCalendarCard({
  activity,
  loading,
  authorOnly,
  onToggleAuthorOnly,
  className,
}: {
  activity: CommitActivityView | null
  loading: boolean
  authorOnly: boolean
  onToggleAuthorOnly: () => void
  className?: string
}) {
  const [month, setMonth] = useState<Date>(new Date())
  const [dialogDay, setDialogDay] = useState<DayActivityView | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)

  const byDate = useMemo(() => {
    const map = new Map<string, DayActivityView>()
    for (const d of activity?.days ?? []) {
      map.set(d.date, d)
    }
    return map
  }, [activity])

  const committedDates = useMemo(
    () =>
      (activity?.days ?? [])
        .filter((d) => d.count > 0)
        .map((d) => parseISO(d.date)),
    [activity],
  )

  const openDay = (date: Date) => {
    const key = dayKey(date)
    const day = byDate.get(key) ?? { date: key, count: 0, commits: [] }
    setDialogDay(day)
    setDialogOpen(true)
  }

  return (
    <div className={cn("flex h-full min-h-0 flex-col gap-1.5", className)}>
      <div className="flex flex-wrap items-center gap-1.5">
        {activity && (
          <Badge variant="outline" className="h-5 px-1.5 text-[10px] font-normal">
            {activity.total} / 12m
          </Badge>
        )}
        <Button
          size="xs"
          variant={authorOnly ? "secondary" : "outline"}
          className="ml-auto h-5 px-1.5 text-[10px]"
          onClick={onToggleAuthorOnly}
          disabled={loading}
          title={
            authorOnly
              ? "Mostrando commits do seu user.email git"
              : "Mostrando commits de todos os autores"
          }
        >
          {authorOnly ? "Meus" : "Todos"}
        </Button>
      </div>

      {loading && !activity ? (
        <div className="flex flex-1 items-center justify-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="size-3.5 animate-spin" />
          Carregando…
        </div>
      ) : (
        <>
          <Calendar
            mode="single"
            locale={ptBR}
            month={month}
            onMonthChange={setMonth}
            onSelect={(date) => {
              if (date) openDay(date)
            }}
            showOutsideDays={false}
            className="w-full p-0 [--cell-size:1.25rem]"
            classNames={{
              weekdays: "hidden",
              weekday: "hidden",
              week: "mt-0.5 flex w-full justify-between gap-0.5",
              day: "flex size-(--cell-size) items-center justify-center p-0",
              month: "flex w-full flex-col gap-1",
              month_caption:
                "flex h-6 w-full items-center justify-center px-6 text-xs",
              button_previous: "size-6 p-0",
              button_next: "size-6 p-0",
              nav: "absolute inset-x-0 top-0 flex w-full items-center justify-between",
            }}
            modifiers={{
              committed: committedDates,
            }}
            components={{
              DayButton: ({ modifiers, day, className: dayClass, ...props }) => {
                const key = dayKey(day.date)
                const count = byDate.get(key)?.count ?? 0
                const heat = intensityClass(count)
                const label =
                  count > 9 ? "9+" : count > 0 ? String(count) : ""
                return (
                  <CalendarDayButton
                    day={day}
                    modifiers={modifiers}
                    className={cn(
                      dayClass,
                      "size-5 min-h-5 min-w-5 rounded-full border-0 p-0 text-[9px] leading-none font-semibold tabular-nums shadow-none",
                      "hover:opacity-90 focus-visible:ring-1 focus-visible:ring-ring",
                      "data-[selected-single=true]:ring-2 data-[selected-single=true]:ring-ring",
                      heat,
                    )}
                    title={
                      count > 0
                        ? `${count} commit${count === 1 ? "" : "s"} em ${key}`
                        : `Nenhum commit em ${key}`
                    }
                    {...props}
                  >
                    {label || (
                      <span className="size-1 rounded-full bg-current opacity-30" />
                    )}
                  </CalendarDayButton>
                )
              },
            }}
          />

          <div className="flex items-center gap-1 text-[9px] text-muted-foreground">
            <span>Menos</span>
            <span className="size-2 rounded-full bg-muted/50" />
            <span className="size-2 rounded-full bg-emerald-500/30" />
            <span className="size-2 rounded-full bg-emerald-500/55" />
            <span className="size-2 rounded-full bg-emerald-600" />
            <span className="size-2 rounded-full bg-emerald-700" />
            <span>Mais</span>
          </div>

          {activity?.authorOnly && activity.authorEmail && (
            <p className="truncate text-[9px] text-muted-foreground">
              {activity.authorName || activity.authorEmail}
            </p>
          )}
        </>
      )}

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="flex max-h-[85vh] w-[min(48rem,calc(100%-2rem))] max-w-none flex-col gap-4 overflow-hidden sm:max-w-none">
          <DialogHeader>
            <DialogTitle>
              {dialogDay
                ? dialogDay.count > 0
                  ? `${dialogDay.count} commit${dialogDay.count === 1 ? "" : "s"} · ${dialogDay.date}`
                  : `Nenhum commit · ${dialogDay.date}`
                : "Commits"}
            </DialogTitle>
          </DialogHeader>

          {dialogDay && dialogDay.count > 0 ? (
            <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden rounded-lg border">
              <table className="w-full table-fixed caption-bottom text-sm">
                <thead className="sticky top-0 bg-popover [&_tr]:border-b">
                  <tr className="border-b">
                    <th className="h-10 w-24 px-3 text-left font-medium">Hash</th>
                    <th className="h-10 px-3 text-left font-medium">Mensagem</th>
                    <th className="h-10 w-40 px-3 text-left font-medium">Autor</th>
                  </tr>
                </thead>
                <tbody>
                  {(dialogDay.commits ?? []).map((c) => (
                    <tr key={c.hash} className="border-b last:border-0">
                      <td className="px-3 py-2 align-top font-mono text-xs">
                        {c.shortHash}
                      </td>
                      <td className="px-3 py-2 align-top break-words whitespace-normal">
                        {c.subject}
                      </td>
                      <td className="px-3 py-2 align-top text-xs break-words whitespace-normal text-muted-foreground">
                        {c.author}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              Nenhum commit neste dia.
            </p>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
