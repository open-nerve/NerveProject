/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { sessionKey } from "./session-key";

const X = { status: "signed-in", loginId: "x" } as const;
const Y = { status: "signed-in", loginId: "y" } as const;

describe("sessionKey", () => {
  it("keys a fetch by its name, the session's loginId and its arguments", () => {
    expect(sessionKey(X, "WORKSPACE_MEMBERS", "acme")).toEqual(["WORKSPACE_MEMBERS", "x", "acme"]);
    expect(sessionKey(X, "WORKSPACES")).toEqual(["WORKSPACES", "x"]);
  });

  it("gives the same fetch in another session another key, so that the new session fetches it again", () => {
    expect(sessionKey(Y, "WORKSPACE_MEMBERS", "acme")).not.toEqual(sessionKey(X, "WORKSPACE_MEMBERS", "acme"));
    // another sign-in as the same account is another session too
    expect(sessionKey({ status: "signed-in", loginId: "x2" }, "WORKSPACES")).not.toEqual(sessionKey(X, "WORKSPACES"));
  });

  it("keys nothing unless the tab is signed in", () => {
    for (const status of ["starting", "signed-out"] as const) expect(sessionKey({ status }, "WORKSPACES")).toBeNull();
    // the first refresh failed for a passing reason: the account is not known to be signed in yet
    expect(sessionKey({ status: "unavailable", loginId: "x", retryAt: 1 }, "WORKSPACES")).toBeNull();
  });

  it("keys nothing while an argument is unknown", () => {
    expect(sessionKey(X, "WORKSPACE_MEMBERS", undefined)).toBeNull();
    expect(sessionKey(X, "PROJECT", "acme", undefined)).toBeNull();
  });
});
