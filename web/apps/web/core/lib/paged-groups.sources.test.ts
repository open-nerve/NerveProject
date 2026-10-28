/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, test } from "vitest";
import { confirmGroups, decodePage, mergePages } from "./paged-groups";
import { listContext, listIssue, pageMeta } from "@/store/issue/fake-issues";
import { heldCounterSources } from "./paged-groups.evidence";

const context = listContext({ group_by: "state_id" });
const x = listIssue("x", 20, { state_id: "B", updated_at: "2026-10-29T00:00:00Z" });
const whole = (sourceID: number) =>
  decodePage(
    {
      total: 10,
      groups: [
        { key: "A", total: 5, data: [listIssue("a", 10, { state_id: "A" })], next_cursor: "A-next" },
        { key: "B", total: 5, data: [listIssue("b", 10, { state_id: "B" })], next_cursor: "B-next" },
      ],
    },
    { ...pageMeta(), sourceID },
    context
  );

test("W1 wholeAlreadyCountsCreateOutsideWindowDoesNotGuessAnotherIncrement", () => {
  const raw = whole(2);
  const copy = structuredClone(raw);
  const result = confirmGroups(raw, { id: "x", beforeSnapshot: { kind: "absent" }, after: x }, context, new Set([1]));
  expect([result.value.total, result.value.groups.get("A")?.total, result.value.groups.get("B")?.total]).toEqual([
    10, 5, 5,
  ]);
  expect(result.value.groups.get("A")?.stale).toBe(false);
  expect(result.value.groups.get("B")?.stale).toBe(true);
  expect(raw).toEqual(copy);
});

test("W1 beforeDispatchHeldWholeKeepsKnownOffscreenMoveExactWithoutStale", () => {
  const raw = whole(7);
  const result = confirmGroups(
    raw,
    {
      id: "x",
      beforeSnapshot: { kind: "known", issue: { ...x, state_id: "A", updated_at: "2026-10-28T00:00:00Z" } },
      after: x,
    },
    context,
    new Set([7])
  );
  expect([result.value.total, result.value.groups.get("A")?.total, result.value.groups.get("B")?.total]).toEqual([
    10, 4, 6,
  ]);
  expect([...result.value.groups.values()].every((group) => !group.stale)).toBe(true);
});

test("W1 partialNewSourceKeepsItsCounterWhileOldSharedSourceUsesFrozenBefore", () => {
  const held = whole(3);
  const raw = decodePage(
    { total: 6, data: [listIssue("tail", 10, { state_id: "B" })], next_cursor: "next" },
    { ...pageMeta({ group: "B" }), sourceID: 4 },
    context
  );
  const result = confirmGroups(
    mergePages(held, raw, context),
    { id: "x", beforeSnapshot: { kind: "absent" }, after: x },
    context,
    new Set([3])
  );
  expect([result.value.total, result.value.groups.get("A")?.total, result.value.groups.get("B")?.total]).toEqual([
    11, 5, 6,
  ]);
  expect(result.value.groups.get("B")?.stale).toBe(true);
});

test("W1 allocatedSourceBelowHeldIDIsNotProofThatItsAnswerWasHeld", () => {
  const raw = whole(4);
  const result = confirmGroups(raw, { id: "x", beforeSnapshot: { kind: "absent" }, after: x }, context, new Set([5]));
  expect([result.value.total, result.value.groups.get("B")?.total]).toEqual([10, 5]);
  expect(result.value.groups.get("B")?.stale).toBe(true);
});

test("W1 wholeReplacementRetiresPartialSourcesAndTheirConfirmedFacts", () => {
  const partial = decodePage(
    { total: 6, data: [listIssue("tail", 10, { state_id: "B" })], next_cursor: "next" },
    { ...pageMeta({ group: "B" }), sourceID: 4 },
    context
  );
  const mixed = mergePages(whole(3), partial, context);
  expect(heldCounterSources(mixed)).toEqual(new Set([3, 4]));
  const old = confirmGroups(
    mixed,
    { id: "x", beforeSnapshot: { kind: "absent" }, after: x },
    context,
    new Set([3, 4])
  ).value;
  const replaced = mergePages(old, whole(5), context);
  expect(heldCounterSources(replaced)).toEqual(new Set([5]));
  const result = confirmGroups(
    replaced,
    { id: "x", beforeSnapshot: { kind: "absent" }, after: x },
    context,
    new Set([3, 4])
  );
  expect([result.value.total, result.value.groups.get("B")?.total]).toEqual([10, 5]);
  expect(result.value.groups.get("B")?.stale).toBe(true);
});
