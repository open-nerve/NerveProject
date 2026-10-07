/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { WEB_SWR_CONFIG } from "@nerve/constants";
import type { SessionState } from "@/lib/auth/token-manager";
import type { SessionKey } from "@/lib/session-key";

// useSessionSWR hands SWR the key and a fetcher; SWR and the session are stand-ins here, so the hook runs as a
// plain function, outside React. What the key is for each session is session-key.test.ts.

/** What the hook hands SWR: the key, the fetcher that SWR calls with the key, the configuration. */
type SWRCall = [key: SessionKey | null, fetcher: (key: SessionKey) => unknown, config: object];

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
    useSessionSWR(["WORKSPACE_MEMBERS", "id-acme", "acme"], fetchList);
    // The tab follows another session: the same fetch is keyed by that session's loginId.
    tab.session = { status: "signed-in", loginId: "y" };
    useSessionSWR(["WORKSPACE_MEMBERS", "id-acme", "acme"], fetchList);
    expect(swr.calls.map(([key]) => key)).toEqual([
      ["WORKSPACE_MEMBERS", "x", "id-acme", "acme"],
      ["WORKSPACE_MEMBERS", "y", "id-acme", "acme"],
    ]);
    // SWR calls each fetcher with the key it was handed: the fetcher gives the store's Promise, for the fetch's
    // arguments only.
    for (const [key, fetcher] of swr.calls) expect(key && fetcher(key)).toBe(answer);
    expect(given).toEqual([
      ["id-acme", "acme"],
      ["id-acme", "acme"],
    ]);
  });

  it("gives every session fetch one configuration: fetched as a page mounts, not on focus, a refusal not retried", () => {
    useSessionSWR(["WORKSPACES"], fetchList);
    const config = { revalidateOnFocus: false, shouldRetryOnError: false };
    expect(swr.calls).toEqual([[["WORKSPACES", "x"], expect.any(Function), config]]);
    // What SWR does is that over the app's configuration (app/provider.tsx). SWR's rule: a set revalidateOnMount
    // decides a hook's first fetch as it mounts (revalidateIfStale then has no say), and the app's is true.
    expect({ ...WEB_SWR_CONFIG, ...swr.calls[0]?.[2] }).toMatchObject({
      revalidateOnMount: true,
      revalidateOnFocus: false,
      shouldRetryOnError: false,
    });
  });

  it("fetches nothing for no fetch, nor before the tab is signed in", () => {
    useSessionSWR(null, fetchList);
    tab.session = { status: "starting" };
    useSessionSWR(["WORKSPACES"], fetchList);
    expect(swr.calls.map(([key]) => key)).toEqual([null, null]);
  });
});
