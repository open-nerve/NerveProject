/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { observable, runInAction } from "mobx";
import { createRoot } from "react-dom/client";
import { vi } from "vitest";
import type { Cycle, CycleListItem, CycleProgress, CycleUpdate, Progress } from "@nerve/api-client";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import { CycleDetailsSidebar } from "./root";

// A cycle's progress sidebar as a person reads it (P5a spec 2.5), for its tests: the cycle store is a stand-in a test
// file gives the sidebar by mocking @/hooks/store/use-cycles with it (sidebarCycles); nerve's progress (live); and
// where the sidebar renders.

/**
 * A stand-in for the cycle store, as the sidebar reads the cycle: the cycle, its list's row and its progress, which a
 * test sets (show); its change of the cycle goes to nerve, which settles the last one sent when the test says (sent);
 * answered, the store writes nerve's answer into the cycle before the change resolves, as the store does.
 */
export function sidebarCycles(cycle: Cycle, held: { row?: CycleListItem; progress?: CycleProgress } = {}) {
  const shown = observable.box<Cycle | undefined>(cycle, { deep: false });
  const row = observable.box(held.row, { deep: false });
  const progress = observable.box(held.progress, { deep: false });
  const sent: { cycleId: string; body: CycleUpdate; change: HeldChange<Cycle> }[] = [];
  const of =
    <T,>(box: { get: () => T }) =>
    (cycleId: string | null | undefined) =>
      cycleId === cycle.id ? box.get() : undefined;
  return {
    getCycleById: of(shown),
    getCycleRow: of(row),
    getCycleProgress: of(progress),
    update: vi.fn(async (cycleId: string, body: CycleUpdate): Promise<Cycle> => {
      const change = heldChange<Cycle>();
      sent.push({ cycleId, body, change });
      const answer = await change.sent;
      runInAction(() => shown.set(answer));
      return answer;
    }),
    sent,
    /** The store now gives these of the cycle. */
    show: (fields: { cycle?: Cycle; row?: CycleListItem; progress?: CycleProgress }) =>
      runInAction(() => {
        if ("cycle" in fields) shown.set(fields.cycle);
        if ("row" in fields) row.set(fields.row);
        if ("progress" in fields) progress.set(fields.progress);
      }),
  };
}

/** nerve's live progress: of no work items unless fields say so. */
export const live = (fields: Partial<Progress> = {}): Progress => ({
  total_issues: 0,
  backlog_issues: 0,
  unstarted_issues: 0,
  started_issues: 0,
  completed_issues: 0,
  cancelled_issues: 0,
  assignees: [],
  labels: [],
  burndown: null,
  ...fields,
});

/** Where a test renders the sidebar; mount() starts it afresh, and unmount() ends it. */
export const page: { host?: HTMLDivElement; root?: ReturnType<typeof createRoot> } = {};

export function mount() {
  page.host = document.createElement("div");
  document.body.append(page.host);
  page.root = createRoot(page.host);
}

export async function unmount() {
  await act(async () => page.root?.unmount());
  page.host?.remove();
}

/** Renders the sidebar of the cycle of cycleId, of web in acme; close is its handleClose. */
export const renderSidebar = (props: Omit<ComponentProps<typeof CycleDetailsSidebar>, "workspaceSlug">) =>
  act(async () => page.root?.render(<CycleDetailsSidebar workspaceSlug="acme" {...props} />));
