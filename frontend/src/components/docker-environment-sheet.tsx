import { useEffect, useState } from "react"

import { AppService } from "../../bindings/github.com/laerciocrestani/openbench"
import type {
  DockerExecResult,
  DockerKitView,
  DockerPresetView,
  DockerServiceView,
  DockerStatus,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { Textarea } from "@/components/ui/textarea"
import {
  Download,
  Loader2,
  Play,
  TerminalSquare,
} from "lucide-react"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  docker: DockerStatus | null | undefined
  busy: boolean
  onOpenDockerShell: (service: string, presetId?: string) => void
  onError: (message: string) => void
  onStatus: (message: string) => void
}

export function DockerEnvironmentSheet({
  open,
  onOpenChange,
  docker,
  busy,
  onOpenDockerShell,
  onError,
  onStatus,
}: Props) {
  const services = docker?.services ?? []
  const [selectedService, setSelectedService] = useState("")
  const [presets, setPresets] = useState<DockerPresetView[]>([])
  const [kits, setKits] = useState<DockerKitView[]>([])
  const [presetsLoading, setPresetsLoading] = useState(false)
  const [actionBusy, setActionBusy] = useState(false)
  const [execResult, setExecResult] = useState<DockerExecResult | null>(null)
  const [resultOpen, setResultOpen] = useState(false)

  useEffect(() => {
    if (!open) return
    const def = docker?.defaultService || services[0]?.name || ""
    setSelectedService((prev) => prev || def)
  }, [open, docker?.defaultService, services])

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setPresetsLoading(true)
    void Promise.all([AppService.ListDockerPresets(), AppService.ListDockerKits()])
      .then(([p, k]) => {
        if (cancelled) return
        setPresets(p ?? [])
        setKits(k ?? [])
      })
      .catch((e) => {
        if (!cancelled) onError(e instanceof Error ? e.message : String(e))
      })
      .finally(() => {
        if (!cancelled) setPresetsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, onError])

  const run = async (fn: () => Promise<void>) => {
    setActionBusy(true)
    try {
      await fn()
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e))
    } finally {
      setActionBusy(false)
    }
  }

  const openShell = () => {
    const svc = selectedService.trim()
    if (!svc) {
      onError("Selecione um container/serviço")
      return
    }
    onOpenDockerShell(svc)
    onStatus(`Shell em ${svc}`)
  }

  const runPreset = async (preset: DockerPresetView) => {
    const svc = selectedService.trim()
    if (!svc) {
      onError("Selecione um container/serviço antes de executar o preset")
      return
    }
    await run(async () => {
      const res = await AppService.DockerRunPreset(svc, preset.id)
      if (!res) return
      if (res.interactive) {
        onOpenDockerShell(svc, preset.id)
        onStatus(res.summary)
        return
      }
      setExecResult(res)
      setResultOpen(true)
      onStatus(res.summary)
    })
  }

  const importKit = async (kitID: string) => {
    await run(async () => {
      const res = await AppService.ImportDockerKit(kitID)
      if (!res) return
      setPresets(res.presets ?? [])
      onStatus(res.message)
    })
  }

  const disabled = busy || actionBusy

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side="right" className="sm:max-w-lg w-full gap-0 p-0">
          <SheetHeader className="border-b p-4 pr-12">
            <SheetTitle className="flex items-center gap-2">
              Docker · Containers
            </SheetTitle>
            <SheetDescription>
              {docker?.composeFile
                ? docker.composeFile.split("/").pop()
                : "Compose do projeto"}
              {docker?.summary ? ` · ${docker.summary}` : ""}
            </SheetDescription>
          </SheetHeader>

          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden p-4">
            <div className="flex flex-col gap-2">
              <Label>Container / serviço</Label>
              {services.length === 0 ? (
                <Alert className="border-amber-500/40 bg-amber-500/10 text-amber-950 dark:text-amber-100">
                  <AlertDescription className="text-xs leading-relaxed">
                    {docker?.composeFile
                      ? docker.total > 0
                        ? "Containers parados. Use Start no card Docker e atualize."
                        : "Ambiente Down. Use Up no card Docker e atualize."
                      : "Nenhum container listado. Rode Up no card Docker e atualize."}
                  </AlertDescription>
                </Alert>
              ) : (
                <Select
                  value={selectedService}
                  onValueChange={(v) => setSelectedService(String(v ?? ""))}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Selecione" />
                  </SelectTrigger>
                  <SelectContent className="w-(--anchor-width)">
                    {services.map((svc) => (
                      <SelectItem key={svc.name} value={svc.name}>
                        <ServiceOption svc={svc} />
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  disabled={disabled || !selectedService}
                  onClick={openShell}
                >
                  <TerminalSquare />
                  Shell
                </Button>
              </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col gap-2">
              <div className="flex items-center justify-between gap-2">
                <Label>Presets do projeto</Label>
                {presetsLoading && <Loader2 className="size-3.5 animate-spin text-muted-foreground" />}
              </div>

              {(kits.length > 0 || presets.length === 0) && (
                <div className="flex flex-wrap gap-2">
                  {(kits.length > 0 ? kits : [{ id: "laravel", label: "Laravel / Artisan", description: "", presetCount: 0 }]).map(
                    (kit) => (
                      <Button
                        key={kit.id}
                        size="xs"
                        variant="outline"
                        disabled={disabled}
                        onClick={() => void importKit(kit.id)}
                      >
                        <Download />
                        Importar {kit.label}
                      </Button>
                    )
                  )}
                </div>
              )}

              <ScrollArea className="min-h-0 flex-1 rounded-lg border">
                <div className="flex flex-col gap-1 p-2">
                  {presets.length === 0 ? (
                    <p className="px-2 py-6 text-center text-sm text-muted-foreground">
                      Nenhum preset. Importe o kit Laravel ou edite{" "}
                      <code className="text-xs">.openbench/docker-presets.yaml</code>
                    </p>
                  ) : (
                    presets.map((p) => (
                      <div
                        key={p.id}
                        className="flex items-start gap-2 rounded-md px-2 py-2 hover:bg-muted/50"
                      >
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <span className="text-sm font-medium">{p.label}</span>
                            {p.interactive && (
                              <Badge variant="outline" className="font-normal">
                                shell
                              </Badge>
                            )}
                            {p.kit && (
                              <Badge variant="secondary" className="font-normal">
                                {p.kit}
                              </Badge>
                            )}
                          </div>
                          <p className="truncate font-mono text-[11px] text-muted-foreground">
                            {p.command}
                          </p>
                          {p.description && (
                            <p className="text-xs text-muted-foreground">{p.description}</p>
                          )}
                        </div>
                        <Button
                          size="xs"
                          variant="secondary"
                          disabled={disabled || !selectedService}
                          onClick={() => void runPreset(p)}
                        >
                          <Play />
                          Run
                        </Button>
                      </div>
                    ))
                  )}
                </div>
              </ScrollArea>
            </div>
          </div>
        </SheetContent>
      </Sheet>

      <Dialog open={resultOpen} onOpenChange={setResultOpen}>
        <DialogContent className="sm:max-w-2xl" showCloseButton>
          <DialogHeader>
            <DialogTitle className="pr-8">
              {execResult?.summary || "Resultado do comando"}
            </DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            {execResult && (
              <p className="font-mono text-xs text-muted-foreground">
                {execResult.service} · {execResult.command}
                {!execResult.ok ? ` · exit ${execResult.exitCode}` : ""}
              </p>
            )}
            <Textarea
              readOnly
              className="min-h-[280px] font-mono text-xs"
              value={execResult?.output?.trim() ? execResult.output : "(sem output)"}
            />
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}

function ServiceOption({ svc }: { svc: DockerServiceView }) {
  return (
    <span className="flex flex-col gap-0.5 text-left">
      <span>
        <span className="font-mono">{svc.name}</span>
        <span className="text-muted-foreground"> · {svc.state || "unknown"}</span>
      </span>
      {(svc.container || svc.ports) && (
        <span className="text-[11px] text-muted-foreground">
          {[svc.container, svc.ports].filter(Boolean).join(" · ")}
        </span>
      )}
    </span>
  )
}
