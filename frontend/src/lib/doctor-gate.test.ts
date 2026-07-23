import { describe, expect, it } from "vitest"

import { doctorBlocksAction, doctorGate } from "./doctor-gate"
import type { DoctorView } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

function report(issues: DoctorView["issues"], overall = "warn"): DoctorView {
  return {
    overall,
    branch: "feature/chat-4",
    base: "main",
    issues,
    recommendations: [],
    lines: [],
    explained: false,
  }
}

describe("doctorGate", () => {
  it("blocks commit/push/pr on merged branch", () => {
    const r = report([
      {
        level: "critical",
        code: "work_on_merged_branch",
        title: 'Branch "feature/chat-4" já tem PR mergeada',
        detail: "PR #12 já foi mergeada",
      },
    ], "critical")
    expect(doctorBlocksAction(r, "commit")).toBe(true)
    expect(doctorBlocksAction(r, "push")).toBe(true)
    expect(doctorBlocksAction(r, "pr")).toBe(true)
    expect(doctorBlocksAction(r, "sync")).toBe(false)
    expect(doctorGate(r, "commit").title).toMatch(/Doctor/)
  })

  it("warns but does not block commit on base_diverged", () => {
    const r = report([
      {
        level: "critical",
        code: "base_diverged",
        title: "Base main divergiu",
        detail: "local ↑0 · remoto ↑2",
      },
    ], "critical")
    expect(doctorBlocksAction(r, "commit")).toBe(false)
    expect(doctorGate(r, "commit").warn || doctorGate(r, "commit").severity !== "ok").toBe(true)
  })

  it("ignores dirty_tree and docker noise for commit", () => {
    const r = report([
      { level: "warn", code: "dirty_tree", title: "dirty", detail: "" },
      { level: "warn", code: "docker_daemon", title: "docker", detail: "" },
    ])
    expect(doctorGate(r, "commit").severity).toBe("ok")
  })
})
