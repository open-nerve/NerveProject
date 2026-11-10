/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { observable, runInAction } from "mobx";
import { vi } from "vitest";
import type { CycleListItem } from "@nerve/api-client";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";

// The cycles of a project's pages of cycles (its list, its archive, a cycle's preview; P5a spec 2.2, 2.5) as they read
// them, for their tests: the cycle store is a stand-in a test file gives the pages by mocking @/hooks/store/use-cycles
// with it (listedCycles). Its reads go to nerve, which settles each when the test says.

/** A list of a project's the store holds: its cycles that are not archived, or its archive. */
type List = "cycles" | "archive";

/**
 * A stand-in for the cycle store as the pages of a project's cycles read it: each of a project's two lists as nerve
 * last answered its read (show sets one as if it had); a cycle's row, of either list, by its id; the project's cycle in
 * progress, the first of its list whose status is current. Each read (a project's two lists, a cycle, its progress)
 * goes to nerve, which settles the last one of its kind when the test says (answer, refuse); a list nerve answers is
 * in the store before the read resolves, as the store does.
 */
export function listedCycles() {
  const lists = observable.map<string, CycleListItem[]>({}, { deep: false });
  const reads: { what: string; of: string; nerve: HeldChange<unknown> }[] = [];
  const read = (what: string, keep?: List) =>
    vi.fn(async (of: string) => {
      const nerve = heldChange<unknown>();
      reads.push({ what, of, nerve });
      const answer = await nerve.sent;
      if (keep) runInAction(() => lists.set(`${keep} ${of}`, answer as CycleListItem[]));
      return answer;
    });
  const listOf = (list: List, projectId: string | null | undefined) => lists.get(`${list} ${projectId}`);
  const rowOf = (cycleId: string | null | undefined) => [...lists.values()].flat().find((row) => row.id === cycleId);
  return {
    getProjectCycleIds: (projectId: string | null | undefined) => listOf("cycles", projectId)?.map((row) => row.id),
    getArchivedCycleIds: (projectId: string | null | undefined) => listOf("archive", projectId)?.map((row) => row.id),
    getActiveCycleId: (projectId: string | null | undefined) => {
      const rows = listOf("cycles", projectId);
      return rows && (rows.find((row) => row.status === "current")?.id ?? null);
    },
    getCycleRow: rowOf,
    getCycleById: rowOf,
    fetchCycles: read("cycles", "cycles"),
    fetchArchivedCycles: read("archive", "archive"),
    fetchCycle: read("cycle"),
    fetchProgress: read("progress"),
    /** The project's list is rows, as if nerve had answered its read. */
    show: (list: List, projectId: string, rows: CycleListItem[]) =>
      runInAction(() => lists.set(`${list} ${projectId}`, rows)),
    /** What the pages have read so far, each as "what of id". */
    asked: () => reads.map((one) => `${one.what} of ${one.of}`),
    /** nerve answers the last read of what. */
    answer: (what: string, body: unknown) =>
      act(async () => reads.findLast((one) => one.what === what)?.nerve.answer(body)),
    /** nerve refuses the last read of what. */
    refuse: (what: string, error: unknown) =>
      act(async () => reads.findLast((one) => one.what === what)?.nerve.refuse(error)),
  };
}
