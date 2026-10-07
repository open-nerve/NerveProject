/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The queue of a store's changes (oneAtATime, v0 design 7.7), for the tests of the stores that queue them, against
// a fake nerve and fake timers.

import { expect, vi } from "vitest";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";

/** The k-th request is the change sent, the last one out until nerve gives it this answer. */
export async function inTurn(nerve: FakeNerve, k: number, [method, path]: [string, string], answer: Response) {
  await until(() => nerve.calls.length === k + 1, `request ${k}`);
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls).toHaveLength(k + 1);
  expect(nerve.calls[k]).toMatchObject({ method, path });
  nerve.calls[k]?.answer(answer);
}

/**
 * A fetch while a change is out (fetches do not queue): the change is sent and left unanswered, the fetch is sent,
 * checked against [method, path] and given answer, and only then is the change refused. The test then checks what the
 * store holds: the fetch's answer.
 */
export async function fetchedWhileChangeIsOut(
  nerve: FakeNerve,
  change: () => Promise<unknown>,
  fetch: () => Promise<unknown>,
  [method, path]: [string, string],
  answer: Response
) {
  const at = nerve.calls.length;
  const changed = track(change());
  await until(() => nerve.calls.length === at + 1, "the change");
  const fetched = track(fetch());
  await until(() => nerve.calls.length === at + 2, "the fetch");
  expect(nerve.calls[at + 1]).toMatchObject({ method, path });
  nerve.calls[at + 1]?.answer(answer);
  await until(() => fetched.settled, "the fetch");
  nerve.calls[at]?.answer(problem(403, "forbidden"));
  await until(() => changed.settled, "the refusal");
}
