import type {
  Dashboard,
  DoctorView,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

export type NextToolbarStep =
  | "commit"
  | "push"
  | "pull"
  | "sync"
  | "pr"
  | "merge"
  | "hygiene"
  | null

export function isOnBase(dash: Dashboard | null): boolean {
  if (!dash || dash.detached) return false
  const base = (dash.baseBranch || "").trim()
  const branch = (dash.branch || "").trim()
  if (!base || !branch) return false
  return branch === base
}

/** Doctor found the current feature branch already has a merged PR. */
export function hasWorkOnMergedBranch(report: DoctorView | null | undefined): boolean {
  return (report?.issues ?? []).some((i) => i?.code === "work_on_merged_branch")
}

export function syncNeedsAttention(dash: Dashboard): boolean {
  return dash.baseBehind > 0
}

export function hygieneNeedsAttention(dash: Dashboard): boolean {
  return (dash.hygieneLocal ?? 0) + (dash.hygieneRemote ?? 0) > 0
}

function prMergeBlocked(dash: Dashboard | null): string | undefined {
  const pr = dash?.openPR
  if (!pr?.url) return "Nenhuma PR aberta nesta branch"
  if (pr.state && !String(pr.state).toUpperCase().startsWith("OPEN")) {
    return "PR já fechada/mergeada"
  }
  if (pr.isDraft) return "Marque Ready for review antes de mergear"
  if (String(pr.mergeable || "").toUpperCase() === "CONFLICTING") return "PR com conflitos"
  if ((pr.checksFail ?? 0) > 0) return "Checks falhando"
  return undefined
}

/**
 * Post-merge on a dead feature branch:
 * - Sync while the local base still lags origin
 * - Hygiene once the base is current (counts omit the current branch, so we
 *   still nudge Hygiene — it checks out the base, then prunes)
 */
function nextStepOnMergedBranch(
  dash: Dashboard,
  hygieneReady: boolean,
): NextToolbarStep {
  if (syncNeedsAttention(dash)) return "sync"
  // Base already matches origin (or Sync just finished). Leave the dead branch.
  if (!hygieneReady) return null
  return "hygiene"
}

/**
 * Single recommended toolbar action based on repo state.
 * After a merged PR, prefer Sync then Hygiene — never reopen PR on the dead branch.
 */
export function nextToolbarStep(
  dash: Dashboard | null,
  openPRReady = true,
  hygieneReady = true,
  gitReady = true,
  onMergedBranch = false,
): NextToolbarStep {
  if (!dash || dash.detached) return null
  if (!gitReady) return null
  if (dash.dirty) return "commit"
  // Post-merge: must win over Push/PR (commitsAheadOfBase can look "ahead" of stale main).
  if (onMergedBranch && !isOnBase(dash)) {
    return nextStepOnMergedBranch(dash, hygieneReady)
  }
  const pushCount = dash.hasUpstream
    ? dash.ahead
    : Math.max(dash.commitsAheadOfBase ?? 0, dash.unpublished ?? 0)
  if (pushCount > 0 && (dash.hasUpstream || Boolean(dash.remoteURL))) return "push"
  if (dash.behind > 0) return "pull"
  // Local base lags origin — Sync before more work.
  if (syncNeedsAttention(dash)) return "sync"
  if (
    !isOnBase(dash) &&
    dash.commitsAheadOfBase > 0 &&
    dash.hasBranchDiff &&
    dash.hasUpstream &&
    dash.ahead === 0 &&
    !dash.openPR?.url
  ) {
    // Don't pulse "abrir PR" until gh confirms there is no open PR.
    if (!openPRReady) return null
    return "pr"
  }
  if (dash.openPR?.url && dash.hasGH && !prMergeBlocked(dash)) {
    return "merge"
  }
  if (hygieneReady && hygieneNeedsAttention(dash)) return "hygiene"
  return null
}

export function nextStepTitle(step: NextToolbarStep): string {
  switch (step) {
    case "commit":
      return "Próximo passo: Commit"
    case "push":
      return "Próximo passo: Push"
    case "pull":
      return "Próximo passo: Pull"
    case "sync":
      return "Próximo passo: Sync da base"
    case "pr":
      return "Próximo passo: abrir Pull Request"
    case "merge":
      return "Próximo passo: Merge PR"
    case "hygiene":
      return "Próximo passo: Hygiene"
    default:
      return ""
  }
}
