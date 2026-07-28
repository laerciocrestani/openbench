import { describe, expect, it } from "vitest"

import {
  hasWorkOnMergedBranch,
  nextToolbarStep,
  syncNeedsAttention,
} from "./next-toolbar-step"
import type {
  Dashboard,
  DoctorView,
} from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

function dash(partial: Partial<Dashboard> = {}): Dashboard {
  return {
    path: partial.path ?? "/repo",
    repoName: partial.repoName ?? "api",
    branch: partial.branch ?? "feature/address-initial",
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
    remoteURL: partial.remoteURL ?? "https://github.com/PI-do-Brasil/api",
    statusLabel: partial.statusLabel ?? "clean",
    changedFiles: partial.changedFiles ?? [],
    nextSteps: partial.nextSteps ?? [],
    hasGH: partial.hasGH ?? true,
    hasDocker: partial.hasDocker ?? false,
    hygieneLocal: partial.hygieneLocal ?? 0,
    hygieneRemote: partial.hygieneRemote ?? 0,
    openPR: partial.openPR,
    docker: partial.docker,
    ...partial,
  }
}

function doctorMerged(): DoctorView {
  return {
    overall: "critical",
    branch: "feature/address-initial",
    base: "main",
    issues: [
      {
        level: "critical",
        code: "work_on_merged_branch",
        title: 'Branch "feature/address-initial" já tem PR mergeada',
        detail: "PR #1 já foi mergeada",
      },
    ],
    recommendations: [],
    lines: [],
    explained: false,
  }
}

describe("nextToolbarStep post-merge", () => {
  it("detects work_on_merged_branch", () => {
    expect(hasWorkOnMergedBranch(doctorMerged())).toBe(true)
    expect(hasWorkOnMergedBranch(null)).toBe(false)
  })

  it("pulses Sync when local base lags after merge", () => {
    const d = dash({
      commitsAheadOfBase: 2,
      hasBranchDiff: true,
      baseBehind: 2,
      openPR: undefined,
    })
    // baseBehind alone already recommends Sync (even without doctor).
    expect(nextToolbarStep(d, true, true, true, false)).toBe("sync")
    expect(nextToolbarStep(d, true, true, true, true)).toBe("sync")
    expect(syncNeedsAttention(d)).toBe(true)
  })

  it("without baseBehind, merged doctor still beats false PR suggestion", () => {
    const d = dash({
      commitsAheadOfBase: 2,
      hasBranchDiff: true,
      baseBehind: 0,
      openPR: undefined,
    })
    expect(nextToolbarStep(d, true, true, true, false)).toBe("pr")
    expect(nextToolbarStep(d, true, true, true, true)).toBe("hygiene")
  })

  it("after Sync (base current), pulses Hygiene even with zero counts", () => {
    // LocalPruneCandidates skips the current branch, so counts often stay 0
    // while still checked out on the merged feature.
    const d = dash({
      baseBehind: 0,
      hygieneLocal: 0,
      hygieneRemote: 0,
      commitsAheadOfBase: 0,
    })
    expect(nextToolbarStep(d, true, true, true, true)).toBe("hygiene")
    expect(syncNeedsAttention(d)).toBe(false)
  })

  it("still prefers Sync when baseBehind > 0 without doctor", () => {
    const d = dash({
      branch: "feature/x",
      baseBehind: 2,
      commitsAheadOfBase: 0,
    })
    expect(nextToolbarStep(d)).toBe("sync")
  })

  it("waits for hygieneReady before pulsing Hygiene post-sync", () => {
    const d = dash({ baseBehind: 0 })
    expect(nextToolbarStep(d, true, false, true, true)).toBe(null)
  })

  it("dirty tree still suggests commit first", () => {
    const d = dash({ dirty: true, baseBehind: 2 })
    expect(nextToolbarStep(d, true, true, true, true)).toBe("commit")
  })
})
