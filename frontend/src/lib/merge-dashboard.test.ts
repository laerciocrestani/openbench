import { describe, expect, it } from "vitest"

import {
  applyChangedFilesResult,
  mergeFastDashboard,
} from "./merge-dashboard"
import type { Dashboard } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

function dash(partial: Partial<Dashboard> & Pick<Dashboard, "path">): Dashboard {
  return {
    path: partial.path,
    repoName: partial.repoName ?? "openbench",
    branch: partial.branch ?? "feature/x",
    detached: partial.detached ?? false,
    dirty: partial.dirty ?? false,
    staged: partial.staged ?? 0,
    modified: partial.modified ?? 0,
    untracked: partial.untracked ?? 0,
    ahead: partial.ahead ?? 0,
    behind: partial.behind ?? 0,
    hasUpstream: partial.hasUpstream ?? true,
    baseBranch: partial.baseBranch ?? "main",
    commitsAheadOfBase: partial.commitsAheadOfBase ?? 0,
    unpublished: partial.unpublished ?? 0,
    hasBranchDiff: partial.hasBranchDiff ?? false,
    baseBehind: partial.baseBehind ?? 0,
    headHash: partial.headHash ?? "abc",
    remoteURL: partial.remoteURL ?? "https://github.com/o/r",
    statusLabel: partial.statusLabel ?? "clean",
    changedFiles: partial.changedFiles ?? [],
    nextSteps: partial.nextSteps ?? [],
    hasGH: partial.hasGH ?? false,
    hasDocker: partial.hasDocker ?? false,
    openPR: partial.openPR,
    docker: partial.docker,
    ...partial,
  }
}

describe("mergeFastDashboard", () => {
  it("preserves hasGH when shell refresh sends false", () => {
    const prev = dash({
      path: "/repo",
      hasGH: true,
      openPR: {
        number: 17,
        title: "fix",
        url: "https://github.com/o/r/pull/17",
        state: "OPEN",
      },
    })
    const next = dash({
      path: "/repo",
      statusLabel: "…",
      hasGH: false,
    })
    const merged = mergeFastDashboard(prev, next)
    expect(merged.hasGH).toBe(true)
    expect(merged.openPR?.number).toBe(17)
  })

  it("preserves hasGH when git status omit/false overwrites", () => {
    const prev = dash({
      path: "/repo",
      hasGH: true,
      statusLabel: "clean",
      openPR: {
        number: 17,
        title: "fix",
        url: "https://github.com/o/r/pull/17",
        state: "OPEN",
      },
    })
    const next = dash({
      path: "/repo",
      hasGH: false,
      statusLabel: "1 staged · 0 modified · 0 untracked",
      dirty: true,
      staged: 1,
    })
    const merged = mergeFastDashboard(prev, next)
    expect(merged.hasGH).toBe(true)
    expect(merged.dirty).toBe(true)
    expect(merged.openPR?.number).toBe(17)
  })

  it("applies dirty changedFiles from git status on main", () => {
    const prev = dash({
      path: "/repo",
      branch: "main",
      hasGH: true,
      statusLabel: "clean",
      changedFiles: [],
    })
    const next = dash({
      path: "/repo",
      branch: "main",
      hasGH: true,
      dirty: true,
      modified: 1,
      statusLabel: "0 staged · 1 modified · 0 untracked",
      changedFiles: [{ path: "frontend/src/App.tsx", status: "modified", insertions: 0, deletions: 0 }],
    })
    const merged = mergeFastDashboard(prev, next)
    expect(merged.dirty).toBe(true)
    expect(merged.changedFiles).toHaveLength(1)
    expect(merged.changedFiles?.[0]?.path).toBe("frontend/src/App.tsx")
  })
})

describe("applyChangedFilesResult", () => {
  it("does not wipe porcelain list with a stale empty numstat result", () => {
    const prev = dash({
      path: "/repo",
      branch: "main",
      dirty: true,
      modified: 1,
      changedFiles: [{ path: "a.ts", status: "modified", insertions: 0, deletions: 0 }],
    })
    const next = applyChangedFilesResult(prev, { changedFiles: [] })
    expect(next.changedFiles).toHaveLength(1)
    expect(next.changedFiles?.[0]?.path).toBe("a.ts")
  })

  it("applies fresh file list when dirty", () => {
    const prev = dash({
      path: "/repo",
      dirty: true,
      modified: 1,
      changedFiles: [{ path: "a.ts", status: "modified", insertions: 0, deletions: 0 }],
    })
    const next = applyChangedFilesResult(prev, {
      changedFiles: [{ path: "a.ts", status: "modified", insertions: 3, deletions: 1 }],
    })
    expect(next.changedFiles?.[0]?.insertions).toBe(3)
    expect(next.changedFiles?.[0]?.deletions).toBe(1)
  })

  it("clears files when working tree is clean", () => {
    const prev = dash({
      path: "/repo",
      dirty: false,
      changedFiles: [{ path: "a.ts", status: "modified", insertions: 1, deletions: 0 }],
    })
    const next = applyChangedFilesResult(prev, { changedFiles: [] })
    expect(next.changedFiles).toEqual([])
  })
})
