/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { landingPath } from "./landing";

/** A workspace of the caller's, created at that moment. */
const workspace = (slug: string, created_at: string) => workspaceOf(slug, { created_at, updated_at: created_at });

// nerve's order: by name, then id
const acme = workspace("acme", "2026-10-02T09:00:00Z");
const beta = workspace("beta", "2026-10-01T09:00:00Z");
const gamma = workspace("gamma", "2026-10-03T09:00:00Z");

describe("landingPath", () => {
  it("goes to the workspace opened last, while it is still one of the caller's", () => {
    expect(landingPath([acme, beta, gamma], gamma.id)).toBe("/gamma");
  });

  it("goes to the workspace created first when the last one is not listed, or there is none", () => {
    // left, removed or deleted since: the hint is not checked to exist (3.14)
    expect(landingPath([acme, beta, gamma], "id-gone")).toBe("/beta");
    expect(landingPath([acme, beta, gamma], null)).toBe("/beta");
    expect(landingPath([gamma, acme], null)).toBe("/acme");
  });

  it("goes to the page that creates a workspace when the caller has none", () => {
    expect(landingPath([], null)).toBe("/create-workspace");
    expect(landingPath([], beta.id)).toBe("/create-workspace");
  });

  it("takes, of the workspaces created first at the same moment, the first in nerve's order", () => {
    const delta = workspace("delta", beta.created_at);
    expect(landingPath([beta, delta], null)).toBe("/beta");
    expect(landingPath([delta, beta], null)).toBe("/delta");
  });

  it("compares the moments, not their text", () => {
    // nerve leaves a fraction's trailing zeros out of a moment (Go's RFC 3339): 09:00:00Z is before 09:00:00.5Z,
    // though its text sorts after
    const whole = workspace("whole", "2026-10-01T09:00:00Z");
    const half = workspace("half", "2026-10-01T09:00:00.5Z");
    expect(landingPath([half, whole], null)).toBe("/whole");
  });
});
