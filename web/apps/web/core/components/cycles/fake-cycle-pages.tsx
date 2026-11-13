/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useRef, type ReactNode } from "react";
import { observable, runInAction } from "mobx";
import { observer } from "mobx-react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import { Where, settle } from "@/components/common/fake-quick-actions";
import { useCycles } from "@/hooks/store/use-cycles";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import { CycleActionsHost } from "./actions-host";
import { CycleQuickActions } from "./quick-actions";

// The pages of a project's cycles as a cycle's quick actions are on them (P5a spec 2.5), for their tests: its list (its
// cycles not archived), a cycle's own page and its archive, each the page of its CycleActionsHost with a row for each
// cycle it shows, the cycle's quick actions in it; and another page, elsewhere. The router's address is the test's to
// read and move (where, fake-quick-actions.tsx, with what a test does on the pages). The cycles are a stand-in store's
// (standInCycles), which a test file gives the pages by mocking @/hooks/store/use-cycles with it.

/**
 * A stand-in for the cycle store over cycles: it gives them by id and lists them as their project's two lists do; its
 * changes (an archive, a restore, a deletion) go to nerve, which settles the last one sent when the test says
 * (answer, refuse); once answered, the store writes it, which takes the cycle's row away, then the change resolves.
 */
export function standInCycles(cycles: Cycle[]) {
  const held = observable.map(
    cycles.map((cycle) => [cycle.id, cycle]),
    { deep: false }
  );
  const sent: HeldChange<void>[] = [];
  /** The cycle of id once nerve has answered its change. */
  const answered = async (id: string): Promise<Cycle> => {
    const change = heldChange<void>();
    sent.push(change);
    await change.sent;
    const cycle = held.get(id);
    if (!cycle) throw new Error("Cycle not found");
    return cycle;
  };
  const moved = (archived_at: string | null) =>
    vi.fn(async (id: string): Promise<Cycle> => {
      const cycle = { ...(await answered(id)), archived_at };
      runInAction(() => held.set(id, cycle));
      return cycle;
    });
  const listed = (archived: boolean) =>
    [...held.values()].filter((cycle) => (cycle.archived_at !== null) === archived).map((cycle) => cycle.id);
  return {
    getCycleById: (id: string | null | undefined) => (id ? held.get(id) : undefined),
    getProjectCycleIds: () => listed(false),
    getArchivedCycleIds: () => listed(true),
    archive: moved("2026-11-11T09:00:00Z"),
    unarchive: moved(null),
    remove: vi.fn(async (id: string): Promise<void> => {
      await answered(id);
      runInAction(() => held.delete(id));
    }),
    /** nerve answers the change sent last. */
    answer: async () => {
      await act(async () => sent.at(-1)?.answer(undefined));
      await settle();
    },
    /** nerve refuses the change sent last, by error. */
    refuse: async (error: unknown) => {
      await act(async () => sent.at(-1)?.refuse(error));
      await settle();
    },
  };
}

/** A row of a page: the cycle's name, and its quick actions, whose right-click menu is the row's. */
const Row = observer(function Row({ cycleId }: { cycleId: string }) {
  const { workspaceSlug = "", projectId = "" } = useParams();
  const { getCycleById } = useCycles();
  const ref = useRef<HTMLDivElement>(null);
  const cycle = getCycleById(cycleId);
  if (!cycle) return null;
  return (
    <div ref={ref} data-row={cycleId}>
      <span>{cycle.name}</span>
      <CycleQuickActions parentRef={ref} cycleId={cycleId} projectId={projectId} workspaceSlug={workspaceSlug} />
    </div>
  );
});

/** A list of the project's cycles: those not archived, or its archive. */
const List = observer(function List({ archived }: { archived: boolean }) {
  const { projectId } = useParams();
  const { getProjectCycleIds, getArchivedCycleIds } = useCycles();
  const ids = (archived ? getArchivedCycleIds : getProjectCycleIds)(projectId) ?? [];
  return ids.map((id) => <Row key={id} cycleId={id} />);
});

function OwnPage() {
  const { cycleId = "" } = useParams();
  return <Row cycleId={cycleId} />;
}

function Page({ children }: { children: ReactNode }) {
  const { workspaceSlug = "", projectId = "" } = useParams();
  return (
    <CycleActionsHost workspaceSlug={workspaceSlug} projectId={projectId}>
      {children}
    </CycleActionsHost>
  );
}

/** The pages of the project's cycles, at path. */
export function CyclePages({ path }: { path: string }) {
  return (
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path=":workspaceSlug/projects/:projectId/cycles"
          element={
            <Page>
              <List archived={false} />
            </Page>
          }
        />
        <Route
          path=":workspaceSlug/projects/:projectId/cycles/:cycleId"
          element={
            <Page>
              <OwnPage />
            </Page>
          }
        />
        <Route
          path=":workspaceSlug/projects/:projectId/archives/cycles"
          element={
            <Page>
              <List archived />
            </Page>
          }
        />
        <Route path="*" element={<p>elsewhere</p>} />
      </Routes>
      <Where />
    </MemoryRouter>
  );
}
