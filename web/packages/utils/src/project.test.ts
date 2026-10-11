/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { projectIdentifierSanitizer } from "./project";

// A project's identifier as typed (M3 design 3.19, 7.6): upper case, of A-Z, 0-9 and ÇŞĞİÖÜ alone.

describe("projectIdentifierSanitizer", () => {
  it.each([
    { typed: "web", identifier: "WEB" },
    { typed: "We b-2", identifier: "WEB2" },
    { typed: "ç ş ğ i ö ü", identifier: "ÇŞĞIÖÜ" },
    { typed: "İzmir", identifier: "İZMIR" },
    // nothing of a path segment's "." or "..", which a check of the identifier would send in its address
    { typed: "..", identifier: "" },
    { typed: "w.e.b", identifier: "WEB" },
  ])("makes $typed $identifier", ({ typed, identifier }) => {
    expect(projectIdentifierSanitizer(typed)).toBe(identifier);
  });
});
