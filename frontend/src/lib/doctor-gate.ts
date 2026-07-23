import type { DoctorIssueView, DoctorView } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

/** Git workflow actions that should respect Doctor findings. */
export type DoctorGateAction = "commit" | "push" | "pr" | "merge" | "sync"

/** Codes that make continuing the action a bad idea — prefer Doctor first. */
const BLOCK_CODES: Record<DoctorGateAction, ReadonlySet<string>> = {
  commit: new Set(["work_on_merged_branch"]),
  push: new Set(["work_on_merged_branch"]),
  pr: new Set(["work_on_merged_branch", "commits_on_base"]),
  merge: new Set(["work_on_merged_branch"]),
  // Sync is often the remedy; never hard-block.
  sync: new Set(),
}

/** Codes shown as soft warnings (action still available). */
const WARN_CODES: Record<DoctorGateAction, ReadonlySet<string>> = {
  commit: new Set(["commits_on_base", "base_diverged", "branch_diverged"]),
  push: new Set(["behind_remote", "base_diverged", "branch_diverged", "commits_on_base"]),
  pr: new Set(["behind_remote", "base_diverged", "branch_diverged"]),
  merge: new Set(["behind_remote", "base_diverged", "branch_diverged"]),
  sync: new Set(["base_diverged", "branch_diverged", "work_on_merged_branch"]),
}

/** Noise for git action modals (expected or unrelated). */
const IGNORE_CODES = new Set([
  "dirty_tree",
  "docker_missing",
  "docker_daemon",
  "docker_stopped",
  "build_artifacts",
])

export type DoctorGate = {
  blocked: boolean
  warn: boolean
  severity: "ok" | "warn" | "critical"
  title: string
  detail: string
  issues: DoctorIssueView[]
}

function relevantIssues(report: DoctorView | null, action: DoctorGateAction): DoctorIssueView[] {
  const issues = report?.issues ?? []
  const block = BLOCK_CODES[action]
  const warn = WARN_CODES[action]
  return issues.filter((iss) => {
    if (!iss?.code || IGNORE_CODES.has(iss.code)) return false
    return block.has(iss.code) || warn.has(iss.code) || iss.level === "critical"
  })
}

export function doctorGate(report: DoctorView | null, action: DoctorGateAction): DoctorGate {
  const issues = relevantIssues(report, action)
  if (issues.length === 0) {
    return {
      blocked: false,
      warn: false,
      severity: "ok",
      title: "",
      detail: "",
      issues: [],
    }
  }

  const blockCodes = BLOCK_CODES[action]
  const blocking = issues.filter((iss) => blockCodes.has(iss.code) || iss.level === "critical")
  const blocked = blocking.length > 0 && blockCodes.size > 0 && blocking.some((iss) => blockCodes.has(iss.code))
  // Soft-critical (e.g. base_diverged on sync) still warns but may not block.
  const hasCritical = issues.some((iss) => iss.level === "critical")
  const primary = (blocked ? blocking[0] : issues[0]) ?? issues[0]

  const severity: DoctorGate["severity"] = blocked || hasCritical ? "critical" : "warn"
  const title = blocked
    ? "Não é a melhor decisão agora — resolva no Doctor"
    : "Há recomendações no Doctor antes de continuar"
  const detail =
    primary?.title ||
    primary?.detail ||
    `${issues.length} achado(s) de saúde no repositório`

  return {
    blocked,
    warn: !blocked,
    severity,
    title,
    detail,
    issues,
  }
}

export function doctorBlocksAction(report: DoctorView | null, action: DoctorGateAction): boolean {
  return doctorGate(report, action).blocked
}
