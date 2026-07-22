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
import { CalendarDays, Loader2 } from "lucide-react"

function dayKey(d: Date): string {
  return format(d, "yyyy-MM-dd")
}

function intensityClass(count: number): string {
  if (count <= 0) return "bg-muted/40"
  if (count === 1) return "bg-emerald-500/25 hover:bg-emerald-500/35"
  if (count <= 3) return "bg-emerald-500/45 hover:bg-emerald-500/55"
  if (count <= 6) return "bg-emerald-600/70 text-primary-foreground hover:bg-emerald-600/80"
  return "bg-emerald-700 text-primary-foreground hover:bg-emerald-700/90"
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
    <div className={cn("flex h-full min-h-0 flex-col gap-2", className)}>
      <div className="flex flex-wrap items-center gap-2">
        <CalendarDays className="size-4 text-muted-foreground" />
        <span className="text-sm font-medium">Commits</span>
        {activity && (
          <Badge variant="outline" className="font-normal">
            {activity.total} em 12 meses
          </Badge>
        )}
        <Button
          size="xs"
          variant={authorOnly ? "secondary" : "outline"}
          className="ml-auto"
          onClick={onToggleAuthorOnly}
          disabled={loading}
          title={
            authorOnly
              ? "Mostrando commits do seu user.email git"
              : "Mostrando commits de todos os autores"
          }
        >
          {authorOnly ? "Meus commits" : "Todos"}
        </Button>
      </div>

      {loading && !activity ? (
        <div className="flex flex-1 items-center justify-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="size-3.5 animate-spin" />
          Carregando calendário…
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
            className="w-full p-0 [--cell-size:--spacing(6)]"
            classNames={{
              weekdays: "hidden",
              weekday: "hidden",
              week: "mt-1 flex w-full gap-1",
              day: "flex-1 p-0",
              month: "flex w-full flex-col gap-2",
            }}
            modifiers={{
              committed: committedDates,
            }}
            components={{
              DayButton: ({ modifiers, day, className: dayClass, ...props }) => {
                const key = dayKey(day.date)
                const count = byDate.get(key)?.count ?? 0
                const heat = intensityClass(count)
                return (
                  <CalendarDayButton
                    day={day}
                    modifiers={modifiers}
                    className={cn(
                      dayClass,
                      "h-7 min-h-7 gap-0 rounded-sm border-0 text-[10px] font-medium",
                      heat,
                    )}
                    title={
                      count > 0
                        ? `${count} commit${count === 1 ? "" : "s"} em ${key}`
                        : `Nenhum commit em ${key}`
                    }
                    {...props}
                  >
                    {count > 0 ? count : null}
                  </CalendarDayButton>
                )
              },
            }}
          />

          <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
            <span>Menos</span>
            <span className="size-2.5 rounded-sm bg-muted/40" />
            <span className="size-2.5 rounded-sm bg-emerald-500/25" />
            <span className="size-2.5 rounded-sm bg-emerald-500/45" />
            <span className="size-2.5 rounded-sm bg-emerald-600/70" />
            <span className="size-2.5 rounded-sm bg-emerald-700" />
            <span>Mais</span>
          </div>

          {activity?.authorOnly && activity.authorEmail && (
            <p className="truncate text-[10px] text-muted-foreground">
              Autor: {activity.authorName || activity.authorEmail}
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
