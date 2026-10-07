/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The queue of a store's changes (oneAtATime, v0 design 7.7), for the tests of the stores that queue them, against
// a fake nerve and fake timers.

import { expect, vi } from "vitest";
import type { Endpoint, FakeNerve } from "@/lib/auth/fake-nerve";
import { answered, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";

/** The k-th request is the change sent, the last one out until nerve gives it this answer. */
export async function inTurn(nerve: FakeNerve, k: number, [method, path]: Endpoint, answer: Response) {
  await until(() => nerve.calls.length === k + 1, `request ${k}`);
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls).toHaveLength(k + 1);
  expect(nerve.calls[k]).toMatchObject({ method, path });
  nerve.calls[k]?.answer(answer);
}

/** A store's request, as the tests make it: what sends it, and the request it is. */
type Sent = { send: () => Promise<unknown>; request: Endpoint };

/**
 * A fetch while a change is out (fetches do not queue): the change is sent, checked against its request and left
 * unanswered; the fetch is sent and answered with body (answered); only then is the change refused. The test then
 * checks what the store holds: the fetch's answer, which the refused change leaves as it is.
 */
export async function fetchedWhileChangeIsOut(nerve: FakeNerve, change: Sent, fetch: Sent & { body: unknown }) {
  const at = nerve.calls.length;
  const changed = track(change.send());
  await until(() => nerve.calls.length === at + 1, "the change");
  const [method, path] = change.request;
  expect(nerve.calls[at]).toMatchObject({ method, path });
  await answered(nerve, fetch.send, fetch.request, fetch.body, "the fetch");
  nerve.calls[at]?.answer(problem(403, "forbidden"));
  await until(() => changed.settled, "the refusal");
}
