import type {
  ChangedFileView,
  CommitContextIndex,
  Dashboard,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

/** Pick changed-files list when merging a git-status (or similar) payload. */
export function resolveChangedFiles(
  prev: Dashboard,
  next: Dashboard,
): ChangedFileView[] | null {
  const nextDirty = Boolean(next.dirty)
  const nextFiles = next.changedFiles ?? []
  if (!nextDirty) return nextFiles
  if (nextFiles.length > 0) return nextFiles
  // Dirty payload without paths (partial) — keep porcelain list already on screen.
  if ((prev.changedFiles?.length ?? 0) > 0) return prev.changedFiles ?? []
  return nextFiles
}

/**
 * Apply a RefreshChangedFiles result. Rejects a stale empty list while the
 * dashboard still reports a dirty working tree (race with repo watcher).
 */
export function applyChangedFilesResult(
  prev: Dashboard,
  files: {
    changedFiles?: ChangedFileView[] | null
    contextIndex?: CommitContextIndex | null
  },
): Dashboard {
  const nextFiles = files.changedFiles ?? []
  const dirtyCounts =
    (prev.staged ?? 0) + (prev.modified ?? 0) + (prev.untracked ?? 0)
  const prevDirty = Boolean(prev.dirty) || dirtyCounts > 0

  if (prevDirty && nextFiles.length === 0 && (prev.changedFiles?.length ?? 0) > 0) {
    return {
      ...prev,
      contextIndex: files.contextIndex ?? prev.contextIndex,
    }
  }

  return {
    ...prev,
    changedFiles: nextFiles,
    contextIndex: files.contextIndex ?? prev.contextIndex,
  }
}

/**
 * Merge a new dashboard into the previous one.
 * Shell payloads (statusLabel === "…") preserve enriched git/PR/docker/CI so
 * refresh doesn't blank the toolbar while background loaders re-run.
 */
export function mergeFastDashboard(prev: Dashboard | null, next: Dashboard): Dashboard {
  if (!prev || prev.path !== next.path) return next
  const sameBranch = prev.branch === next.branch && !next.detached
  const isShell = next.statusLabel === "…"
  const dockerStub =
    next.hasDocker &&
    (!next.docker || next.docker.summary === "carregando…" || next.docker.total === 0)
  const preservedPR = sameBranch ? prev.openPR : undefined
  const openPR =
    next.openPR ??
    (preservedPR && (!preservedPR.state || String(preservedPR.state).toUpperCase().startsWith("OPEN"))
      ? preservedPR
      : undefined)
  const preserveCI = sameBranch && !next.ciLabel && !!prev.ciLabel
  // Partial loaders historically omitted HasGH (Go zero → false). Never wipe a
  // known-true value or Merge PR stays disabled while still showing "próximo".
  const hasGH = Boolean(next.hasGH || prev.hasGH)
  const hasDocker = Boolean(next.hasDocker || prev.hasDocker)

  if (isShell && sameBranch) {
    return {
      ...prev,
      ...next,
      dirty: prev.dirty,
      staged: prev.staged,
      modified: prev.modified,
      untracked: prev.untracked,
      ahead: prev.ahead,
      behind: prev.behind,
      hasUpstream: prev.hasUpstream,
      unpublished: next.unpublished ?? prev.unpublished,
      commitsAheadOfBase: prev.commitsAheadOfBase,
      hasBranchDiff: prev.hasBranchDiff,
      baseBehind: prev.baseBehind,
      statusLabel: prev.statusLabel === "…" ? "…" : prev.statusLabel,
      changedFiles: prev.changedFiles?.length ? prev.changedFiles : next.changedFiles,
      contextIndex: prev.contextIndex ?? next.contextIndex,
      hygieneLocal: prev.hygieneLocal,
      hygieneRemote: prev.hygieneRemote,
      remoteURL: next.remoteURL || prev.remoteURL,
      hasGH,
      hasDocker,
      openPR,
      docker: dockerStub && prev.docker?.total ? prev.docker : (next.docker ?? prev.docker),
      ...(preserveCI
        ? {
            ciState: prev.ciState,
            ciLabel: prev.ciLabel,
            ciFromCache: prev.ciFromCache,
            ciHost: prev.ciHost,
          }
        : {}),
    }
  }

  return {
    ...next,
    hasGH,
    hasDocker,
    openPR,
    changedFiles: resolveChangedFiles(prev, next),
    docker: dockerStub && prev.docker?.total ? prev.docker : next.docker,
    ...(preserveCI
      ? {
          ciState: prev.ciState,
          ciLabel: prev.ciLabel,
          ciFromCache: prev.ciFromCache,
          ciHost: prev.ciHost,
        }
      : {}),
  }
}
