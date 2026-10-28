/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, describe, expect, it, vi } from "vitest";
import { reaction } from "mobx";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { Reconciled, type Change } from "@/lib/reconciled";

type Value = { left: number; right: number; mark: string; stale: boolean };
const initial: Value = { left: 1, right: 2, mark: "", stale: false };
const mergeLeft = (held: Value, answer: Value): Value => ({ ...held, left: answer.left });
const mergeRight = (held: Value, answer: Value): Value => ({ ...held, right: answer.right });

// Only the read's completion is controlled; every reconciliation runs in the real class.
function deferred() {
  let resolve!: (value: Value) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<Value>((done, fail) => {
    resolve = done;
    reject = fail;
  });
  return { resolve, reject, read: vi.fn(() => promise) };
}

const disposers: (() => void)[] = [];
afterEach(() => disposers.splice(0).forEach((dispose) => dispose()));

function watched() {
  const held = new Reconciled<Value>();
  const history: (Value | undefined)[] = [];
  disposers.push(
    reaction(
      () => held.value,
      (value) => history.push(value),
      { fireImmediately: true }
    )
  );
  return { held, history };
}

describe("Reconciled partial reads", () => {
  // Calling read before a whole value exists would create a partial canonical value.
  it("fetchPartIsInertBeforeFetch", async () => {
    const { held, history } = watched();
    const part = deferred();
    expect(await held.fetchPart("left", part.read, mergeLeft)).toBeUndefined();
    const whole = deferred();
    const fetching = held.fetch(whole.read);
    expect(await held.fetchPart("left", part.read, mergeLeft)).toBeUndefined();
    expect(part.read).not.toHaveBeenCalled();
    expect(history).toEqual([undefined]);
    whole.resolve(initial);
    expect(await fetching).toEqual(initial);
    const reading = held.fetchPart("left", part.read, mergeLeft);
    part.resolve({ ...initial, left: 3 });
    expect(await reading).toEqual({ ...initial, left: 3 });
    expect(whole.read).toHaveBeenCalledTimes(1);
    expect(part.read).toHaveBeenCalledTimes(1);
    expect(history).toEqual([undefined, initial, { ...initial, left: 3 }]);
  });

  // Removing whole-fetch invalidation would let either old part overwrite the newer whole.
  it("wholeFetchSupersedesAllOlderParts", async () => {
    const { held, history } = watched();
    await held.fetch(async () => initial);
    const left = deferred();
    const right = deferred();
    const whole = deferred();
    const readingLeft = held.fetchPart("left", left.read, mergeLeft);
    const readingRight = held.fetchPart("right", right.read, mergeRight);
    const fetching = held.fetch(whole.read);
    const answeredAt = held.answeredAt;
    left.resolve({ ...initial, left: 10 });
    expect(await readingLeft).toBeUndefined();
    expect(held.answeredAt).toBe(answeredAt);
    whole.resolve({ ...initial, left: 20, right: 30 });
    expect(await fetching).toEqual({ ...initial, left: 20, right: 30 });
    const wholeAnsweredAt = held.answeredAt;
    expect(wholeAnsweredAt).toBeGreaterThan(answeredAt);
    right.resolve({ ...initial, right: 11 });
    expect(await readingRight).toBeUndefined();
    expect(held.answeredAt).toBe(wholeAnsweredAt);
    expect(history).toEqual([undefined, initial, { ...initial, left: 20, right: 30 }]);
    for (const read of [left, right, whole]) expect(read.read).toHaveBeenCalledTimes(1);
  });

  // Both arrival orders matter: an obsolete finally must not clear its successor's changes.
  it.each([true, false])("samePartSupersedesOlderAnswer (older first: %s)", async (olderFirst) => {
    const { held, history } = watched();
    await held.fetch(async () => initial);
    const older = deferred();
    const newer = deferred();
    const oldRead = held.fetchPart("left", older.read, mergeLeft);
    const newRead = held.fetchPart("left", newer.read, mergeLeft);
    if (olderFirst) {
      older.resolve({ ...initial, left: 10 });
      expect(await oldRead).toBeUndefined();
    }
    held.confirm((value) => ({ ...value, left: 99 }));
    newer.resolve({ ...initial, left: 20 });
    expect(await newRead).toEqual({ ...initial, left: 99 });
    if (!olderFirst) {
      older.resolve({ ...initial, left: 10 });
      expect(await oldRead).toBeUndefined();
    }
    expect(history).toEqual([undefined, initial, { ...initial, left: 99 }, { ...initial, left: 99 }]);
    expect(older.read).toHaveBeenCalledTimes(1);
    expect(newer.read).toHaveBeenCalledTimes(1);
  });

  // Merging against the value at dispatch loses another part that has already answered.
  it("differentPartsMergeIntoLatestValue", async () => {
    const { held, history } = watched();
    await held.fetch(async () => initial);
    const left = deferred();
    const right = deferred();
    const readingLeft = held.fetchPart("left", left.read, mergeLeft);
    const readingRight = held.fetchPart("right", right.read, mergeRight);
    right.resolve({ ...initial, right: 20 });
    expect(await readingRight).toEqual({ ...initial, right: 20 });
    const firstAnsweredAt = held.answeredAt;
    held.confirm((value) => ({ ...value, left: 99 }));
    left.resolve({ ...initial, left: 10 });
    expect(await readingLeft).toEqual({ ...initial, left: 99, right: 20 });
    expect(held.answeredAt).toBeGreaterThan(firstAnsweredAt);
    expect(history).toEqual([
      undefined,
      initial,
      { ...initial, right: 20 },
      { ...initial, left: 99, right: 20 },
      { ...initial, left: 99, right: 20 },
    ]);
    expect(left.read).toHaveBeenCalledTimes(1);
    expect(right.read).toHaveBeenCalledTimes(1);
  });

  // Passing the merged/replayed value as proof fabricates fields the read never answered.
  it.each(["whole", "part"])("replaysConfirmationsInOrderWithRawAnswer (%s)", async (kind) => {
    const { held, history } = watched();
    await held.fetch(async () => initial);
    const read = deferred();
    const fetching = kind === "whole" ? held.fetch(read.read) : held.fetchPart("left", read.read, mergeLeft);
    const calls: { name: string; value: Value; answered: Value | undefined; arity: number }[] = [];
    const change = (name: string, mark: string): Change<Value> =>
      function (value, answered) {
        calls.push({ name, value, answered, arity: arguments.length });
        return { ...value, mark };
      };
    held.confirm(change("first", "A"));
    held.confirm(change("second", "B"));
    const raw = { ...initial, left: 10, right: 200, mark: "raw" };
    read.resolve(raw);
    const right = kind === "whole" ? 200 : 2;
    expect(await fetching).toEqual({ ...initial, left: 10, right, mark: "B" });
    expect(calls.map(({ name, answered, arity }) => [name, answered, arity])).toEqual([
      ["first", undefined, 1],
      ["second", undefined, 1],
      ["first", raw, 2],
      ["second", raw, 2],
    ]);
    expect(calls[2].answered).toBe(raw);
    expect(calls[3].answered).toBe(raw);
    expect(calls[2].value).toEqual({ ...initial, left: 10, right, mark: kind === "whole" ? "raw" : "B" });
    expect(calls[3].value).toEqual({ ...initial, left: 10, right, mark: "A" });
    expect(history).toEqual([
      undefined,
      initial,
      { ...initial, mark: "A" },
      { ...initial, mark: "B" },
      { ...initial, left: 10, right, mark: "B" },
    ]);
    expect(read.read).toHaveBeenCalledTimes(1);
  });

  // Obsolete rejection and session changes are inert; a live ordinary failure must reach its caller.
  it.each(["whole", "part"])("obsoleteFailureAndSessionChangeDoNotWrite (%s)", async (kind) => {
    const { held, history } = watched();
    await held.fetch(async () => initial);
    const fetch = (read: () => Promise<Value>) =>
      kind === "whole" ? held.fetch(read) : held.fetchPart("left", read, mergeLeft);
    const older = deferred();
    const newer = deferred();
    const oldRead = fetch(older.read);
    const newRead = fetch(newer.read);
    older.reject(new Error("obsolete failure"));
    expect(await oldRead).toBeUndefined();
    held.confirm((value) => ({ ...value, left: 99 }));
    newer.resolve({ ...initial, left: 20 });
    expect(await newRead).toEqual({ ...initial, left: 99 });
    const answeredAt = held.answeredAt;
    const session = deferred();
    const sessionRead = fetch(session.read);
    session.reject(new SessionChangedError());
    expect(await sessionRead).toBeUndefined();
    const failed = deferred();
    const failedRead = fetch(failed.read);
    const error = new Error("current failure");
    const rejection = expect(failedRead).rejects.toBe(error);
    failed.reject(error);
    await rejection;
    expect(held.answeredAt).toBe(answeredAt);
    expect(history).toEqual([undefined, initial, { ...initial, left: 99 }, { ...initial, left: 99 }]);
    for (const read of [older, newer, session, failed]) expect(read.read).toHaveBeenCalledTimes(1);
  });

  // Invalidation preserves held stale data and cannot let an old finally clear a new read's replay.
  it.each(["whole", "part"])("invalidatedReadsCannotClearHeldStaleValue (%s)", async (kind) => {
    const { held, history } = watched();
    await held.fetch(async () => initial);
    const oldWhole = deferred();
    const oldPart = deferred();
    const fetchingWhole = held.fetch(oldWhole.read);
    const fetchingPart = held.fetchPart("left", oldPart.read, mergeLeft);
    held.confirm((value) => ({ ...value, stale: true }));
    const stale = held.value;
    const answeredAt = held.answeredAt;
    held.invalidate();
    expect(held.value).toBe(stale);
    expect(held.answeredAt).toBe(answeredAt);
    const next = deferred();
    const fetchingNext = kind === "whole" ? held.fetch(next.read) : held.fetchPart("left", next.read, mergeLeft);
    oldWhole.resolve({ ...initial, left: 10 });
    expect(await fetchingWhole).toBeUndefined();
    oldPart.resolve({ ...initial, left: 20 });
    expect(await fetchingPart).toBeUndefined();
    expect(held.value).toBe(stale);
    expect(held.answeredAt).toBe(answeredAt);
    held.confirm((value) => ({ ...value, left: 99, stale: true }));
    next.resolve({ ...initial, left: 30 });
    expect(await fetchingNext).toEqual({ ...initial, left: 99, stale: true });
    expect(history).toEqual([
      undefined,
      initial,
      { ...initial, stale: true },
      { ...initial, left: 99, stale: true },
      { ...initial, left: 99, stale: true },
    ]);
    for (const read of [oldWhole, oldPart, next]) expect(read.read).toHaveBeenCalledTimes(1);
  });
});
