import { useEffect, useMemo, useState } from "react"
import { format, parseISO } from "date-fns"
import { ptBR } from "date-fns/locale"

import type {
  CommitActivityView,
  DayActivityView,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
import { Calendar, CalendarDayButton } from "@/components/ui/calendar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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
  selectedDay,
  onSelectDay,
  className,
}: {
  activity: CommitActivityView | null
  loading: boolean
  authorOnly: boolean
  onToggleAuthorOnly: () => void
  selectedDay: string
  onSelectDay: (day: string) => void
  className?: string
}) {
  const [month, setMonth] = useState<Date>(() => {
    try {
      return selectedDay ? parseISO(selectedDay) : new Date()
    } catch {
      return new Date()
    }
  })

  useEffect(() => {
    if (!selectedDay) return
    try {
      setMonth(parseISO(selectedDay))
    } catch {
      /* ignore */
    }
  }, [selectedDay])

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

  const selectedDate = useMemo(() => {
    try {
      return selectedDay ? parseISO(selectedDay) : undefined
    } catch {
      return undefined
    }
  }, [selectedDay])

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
            selected={selectedDate}
            onSelect={(date) => {
              if (date) onSelectDay(dayKey(date))
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
                const isSelected = key === selectedDay
                return (
                  <CalendarDayButton
                    day={day}
                    modifiers={modifiers}
                    className={cn(
                      dayClass,
                      "size-5 min-h-5 min-w-5 rounded-full border-0 p-0 text-[9px] leading-none font-semibold tabular-nums shadow-none",
                      "hover:opacity-90 focus-visible:ring-1 focus-visible:ring-ring",
                      isSelected && "ring-2 ring-ring ring-offset-1 ring-offset-background",
                      heat,
                    )}
                    title={
                      count > 0
                        ? `${count} commit${count === 1 ? "" : "s"} em ${key} — ver atividade`
                        : `Nenhuma atividade em ${key}`
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
    </div>
  )
}
