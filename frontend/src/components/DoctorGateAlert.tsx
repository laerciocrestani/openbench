import { AlertTriangle, Stethoscope } from "lucide-react"

import type { DoctorView } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { doctorGate, type DoctorGateAction } from "@/lib/doctor-gate"
import { cn } from "@/lib/utils"

export function DoctorGateAlert({
  report,
  action,
  onOpenDoctor,
  className,
}: {
  report: DoctorView | null
  action: DoctorGateAction
  onOpenDoctor: () => void
  className?: string
}) {
  const gate = doctorGate(report, action)
  if (gate.severity === "ok") return null

  const critical = gate.severity === "critical" || gate.blocked

  return (
    <Alert
      variant={critical ? "destructive" : "default"}
      className={cn(
        !critical && "border-amber-500/40 bg-amber-500/10 text-amber-950 dark:text-amber-100",
        className,
      )}
    >
      <AlertTriangle className="size-4" />
      <AlertTitle className="text-sm">{gate.title}</AlertTitle>
      <AlertDescription className="mt-1 space-y-2">
        <p className="text-xs leading-relaxed">{gate.detail}</p>
        {gate.issues.length > 1 && (
          <ul className="list-disc space-y-0.5 pl-4 text-xs opacity-90">
            {gate.issues.slice(0, 3).map((iss) => (
              <li key={`${iss.code}-${iss.title}`}>{iss.title}</li>
            ))}
          </ul>
        )}
        <Button
          type="button"
          size="sm"
          variant={critical ? "secondary" : "outline"}
          className="mt-1"
          onClick={onOpenDoctor}
        >
          <Stethoscope />
          Resolver no Doctor
        </Button>
      </AlertDescription>
    </Alert>
  )
}
