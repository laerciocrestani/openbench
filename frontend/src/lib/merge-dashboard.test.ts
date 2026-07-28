import { describe, expect, it } from "vitest"

import { mergeFastDashboard } from "./merge-dashboard"
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
})
