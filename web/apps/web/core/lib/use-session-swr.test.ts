/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SessionState } from "@/lib/auth/token-manager";
import type { SessionKey } from "@/lib/session-key";

// useSessionSWR hands SWR the key and a fetcher; SWR and the session are stand-ins here, so the hook runs as a
// plain function, outside React. What the key is for each session is session-key.test.ts.

/** What the hook hands SWR: the key, the fetcher that SWR calls with the key, the configuration. */
type SWRCall = [key: SessionKey | null, fetcher: (key: SessionKey) => unknown, config?: unknown];

const tab = vi.hoisted((): { session: SessionState } => ({ session: { status: "starting" } }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => tab.session }));
const swr = vi.hoisted((): { calls: SWRCall[] } => ({ calls: [] }));
vi.mock("swr", () => ({
  default: (...call: SWRCall) => {
    swr.calls.push(call);
    return {};
  },
}));

const { useSessionSWR } = await import("./use-session-swr");

/** A store's fetch, which tells the arguments it was given; its Promise is what SWR must get. */
let given: string[][] = [];
const answer = Promise.resolve([]);
const fetchList = (...args: string[]) => {
  given.push(args);
  return answer;
};

beforeEach(() => {
  swr.calls = [];
  given = [];
  tab.session = { status: "signed-in", loginId: "x" };
});

describe("useSessionSWR", () => {
  it("keys the fetch by the session, and its fetcher gives the store's Promise for the fetch's arguments", () => {
    const config = { revalidateOnFocus: false };
    useSessionSWR(["WORKSPACE_MEMBERS", "acme"], fetchList, config);
    expect(swr.calls).toEqual([[["WORKSPACE_MEMBERS", "x", "acme"], expect.any(Function), config]]);
    // The tab follows another session: the same fetch is keyed by that session's loginId.
    tab.session = { status: "signed-in", loginId: "y" };
    useSessionSWR(["WORKSPACE_MEMBERS", "acme"], fetchList, config);
    expect(swr.calls[1]).toEqual([["WORKSPACE_MEMBERS", "y", "acme"], expect.any(Function), config]);
    // SWR calls each fetcher with the key it was handed: the fetcher gives the store's Promise, for the fetch's
    // arguments only.
    for (const [key, fetcher] of swr.calls) expect(key && fetcher(key)).toBe(answer);
    expect(given).toEqual([["acme"], ["acme"]]);
  });

  it("fetches nothing for no fetch, nor before the tab is signed in", () => {
    useSessionSWR(null, fetchList);
    tab.session = { status: "starting" };
    useSessionSWR(["WORKSPACES"], fetchList);
    expect(swr.calls.map(([key]) => key)).toEqual([null, null]);
  });
});
