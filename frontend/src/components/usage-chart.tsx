import { useEffect, useMemo, useState } from "react"
import { Bar, BarChart, CartesianGrid, XAxis } from "recharts"
import { Loader2, RefreshCw } from "lucide-react"

import { AppService } from "../../bindings/github.com/laerciocrestani/openbench"
import type { UsageReportView } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

const PERIODS = [
  { key: "24h", label: "Últimas 24h" },
  { key: "7d", label: "Últimos 7 dias" },
  { key: "30d", label: "Últimos 30 dias" },
  { key: "90d", label: "Últimos 90 dias" },
  { key: "month", label: "Mês atual" },
  { key: "all", label: "Todo o histórico" },
] as const

const chartConfig = {
  tokens: { label: "Tokens" },
  input: {
    label: "Entrada",
    color: "var(--chart-1)",
  },
  output: {
    label: "Saída",
    color: "var(--chart-2)",
  },
} satisfies ChartConfig

type ActiveMetric = "input" | "output"

function formatTokens(n: number): string {
  return n.toLocaleString("pt-BR")
}

function formatCost(n: number): string {
  return `$${n.toFixed(6)}`
}

function formatTick(value: string, hourly: boolean): string {
  if (hourly) {
    const d = new Date(`${value}:00:00`)
    if (Number.isNaN(d.getTime())) return value
    return d.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" })
  }
  const d = new Date(`${value}T12:00:00`)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleDateString("pt-BR", { month: "short", day: "numeric" })
}

function formatLabel(value: string, hourly: boolean): string {
  if (hourly) {
    const d = new Date(`${value}:00:00`)
    if (Number.isNaN(d.getTime())) return value
    return d.toLocaleString("pt-BR", {
      day: "2-digit",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
    })
  }
  const d = new Date(`${value}T12:00:00`)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleDateString("pt-BR", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  })
}

export function UsageChartPanel({ open }: { open: boolean }) {
  const [period, setPeriod] = useState("30d")
  const [report, setReport] = useState<UsageReportView | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [activeChart, setActiveChart] = useState<ActiveMetric>("input")

  const load = async (periodKey: string) => {
    setBusy(true)
    setError(null)
    try {
      const res = await AppService.GetUsageReport(periodKey)
      setReport(res)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setReport(null)
    } finally {
      setBusy(false)
    }
  }

  useEffect(() => {
    if (!open) return
    void load(period)
  }, [open, period])

  const chartData = useMemo(
    () =>
      (report?.series ?? []).map((p) => ({
        date: p.date,
        input: p.input,
        output: p.output,
      })),
    [report],
  )

  const totals = useMemo(
    () => ({
      input: report?.totalInput ?? 0,
      output: report?.totalOutput ?? 0,
    }),
    [report],
  )

  const hourly = report?.granularity === "hour"

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <Select value={period} onValueChange={(v) => setPeriod(String(v ?? "30d"))}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Período" />
          </SelectTrigger>
          <SelectContent>
            {PERIODS.map((p) => (
              <SelectItem key={p.key} value={p.key}>
                {p.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void load(period)}
          disabled={busy}
        >
          {busy ? <Loader2 className="animate-spin" /> : <RefreshCw />}
          Atualizar
        </Button>
        {report?.periodLabel && (
          <span className="text-xs text-muted-foreground">{report.periodLabel}</span>
        )}
      </div>

      {error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      {busy && !report ? (
        <div className="flex items-center justify-center gap-2 py-16 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Carregando uso…
        </div>
      ) : (
        <>
          <Card className="py-0 pb-4">
            <CardHeader className="flex flex-col items-stretch border-b p-0! sm:flex-row">
              <div className="flex flex-1 flex-col justify-center gap-1 px-6 pt-4 pb-3 sm:py-0!">
                <CardTitle>Uso de tokens</CardTitle>
                <CardDescription>
                  Entrada e saída do ledger de IA (mesmo histórico do{" "}
                  <span className="font-mono">ob report</span>)
                </CardDescription>
                <div className="mt-1 flex flex-wrap gap-2 text-xs text-muted-foreground">
                  <Badge variant="secondary" className="font-normal">
                    {report?.calls ?? 0} chamada(s)
                  </Badge>
                  {report?.hasCost && (
                    <Badge variant="outline" className="font-normal">
                      Total {formatCost(report.totalCost)} USD
                    </Badge>
                  )}
                </div>
              </div>
              <div className="flex">
                {(["input", "output"] as const).map((key) => (
                  <button
                    key={key}
                    type="button"
                    data-active={activeChart === key}
                    className="relative z-30 flex flex-1 flex-col justify-center gap-1 border-t px-6 py-4 text-left even:border-l data-[active=true]:bg-muted/50 sm:border-t-0 sm:border-l sm:px-8 sm:py-6"
                    onClick={() => setActiveChart(key)}
                  >
                    <span className="text-xs text-muted-foreground">
                      {chartConfig[key].label}
                    </span>
                    <span className="text-lg leading-none font-bold sm:text-3xl tabular-nums">
                      {formatTokens(totals[key])}
                    </span>
                  </button>
                ))}
              </div>
            </CardHeader>
            <CardContent className="space-y-4 px-2 pt-4 sm:px-6">
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-lg border bg-muted/30 px-4 py-3">
                  <p className="text-xs font-medium text-muted-foreground">Chat IA</p>
                  <p className="mt-1 text-lg font-semibold tabular-nums">
                    {report?.chat?.hasCost
                      ? formatCost(report.chat.cost)
                      : report?.chat?.calls
                        ? "—"
                        : formatCost(0)}
                    {report?.chat?.hasCost ? (
                      <span className="ml-1 text-xs font-normal text-muted-foreground">
                        USD
                      </span>
                    ) : null}
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground tabular-nums">
                    {report?.chat?.calls ?? 0} chamada(s) ·{" "}
                    {formatTokens((report?.chat?.input ?? 0) + (report?.chat?.output ?? 0))}{" "}
                    tokens
                  </p>
                </div>
                <div className="rounded-lg border bg-muted/30 px-4 py-3">
                  <p className="text-xs font-medium text-muted-foreground">
                    Commits, PRs e outros
                  </p>
                  <p className="mt-1 text-lg font-semibold tabular-nums">
                    {report?.other?.hasCost
                      ? formatCost(report.other.cost)
                      : report?.other?.calls
                        ? "—"
                        : formatCost(0)}
                    {report?.other?.hasCost ? (
                      <span className="ml-1 text-xs font-normal text-muted-foreground">
                        USD
                      </span>
                    ) : null}
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground tabular-nums">
                    {report?.other?.calls ?? 0} chamada(s) ·{" "}
                    {formatTokens(
                      (report?.other?.input ?? 0) + (report?.other?.output ?? 0),
                    )}{" "}
                    tokens
                  </p>
                </div>
              </div>
              {chartData.length === 0 || (report?.calls ?? 0) === 0 ? (
                <p className="py-12 text-center text-sm text-muted-foreground">
                  Nenhum uso registrado neste período.
                </p>
              ) : (
                <ChartContainer
                  config={chartConfig}
                  className="aspect-auto h-[250px] w-full"
                >
                  <BarChart
                    accessibilityLayer
                    data={chartData}
                    margin={{ left: 12, right: 12 }}
                  >
                    <CartesianGrid vertical={false} />
                    <XAxis
                      dataKey="date"
                      tickLine={false}
                      axisLine={false}
                      tickMargin={8}
                      minTickGap={32}
                      tickFormatter={(value) =>
                        formatTick(String(value), hourly)
                      }
                    />
                    <ChartTooltip
                      content={
                        <ChartTooltipContent
                          className="w-[160px]"
                          nameKey="tokens"
                          labelFormatter={(value) =>
                            formatLabel(String(value), hourly)
                          }
                        />
                      }
                    />
                    <Bar
                      dataKey={activeChart}
                      fill={`var(--color-${activeChart})`}
                      radius={4}
                    />
                  </BarChart>
                </ChartContainer>
              )}
            </CardContent>
          </Card>

          {(report?.byModel?.length ?? 0) > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Por modelo</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Modelo</TableHead>
                      <TableHead className="text-right">Chamadas</TableHead>
                      <TableHead className="text-right">Entrada</TableHead>
                      <TableHead className="text-right">Saída</TableHead>
                      <TableHead className="text-right">Custo</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {report!.byModel!.map((m) => (
                      <TableRow key={m.name}>
                        <TableCell className="font-mono text-xs">{m.name}</TableCell>
                        <TableCell className="text-right tabular-nums">
                          {m.calls}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {formatTokens(m.input)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {formatTokens(m.output)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {m.hasCost ? formatCost(m.cost) : "—"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}

          {(report?.byProject?.length ?? 0) > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Por projeto</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Projeto</TableHead>
                      <TableHead className="text-right">Chamadas</TableHead>
                      <TableHead className="text-right">Entrada</TableHead>
                      <TableHead className="text-right">Saída</TableHead>
                      <TableHead className="text-right">Custo</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {report!.byProject!.map((p) => (
                      <TableRow key={p.name}>
                        <TableCell className="font-mono text-xs">{p.name}</TableCell>
                        <TableCell className="text-right tabular-nums">
                          {p.calls}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {formatTokens(p.input)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {formatTokens(p.output)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {p.hasCost ? formatCost(p.cost) : "—"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </>
      )}
    </div>
  )
}
