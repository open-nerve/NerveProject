/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { observable, runInAction } from "mobx";
import { observer } from "mobx-react";
import { MemoryRouter, Route, Routes, useNavigate, useParams, type NavigateFunction } from "react-router";
import { vi } from "vitest";
import type { CycleListItem, CycleTransferResult, FrozenCounts, Project } from "@nerve/api-client";
import { settle } from "@/components/common/fake-quick-actions";
import { useCycles } from "@/hooks/store/use-cycles";
import { heldChange, type HeldChange } from "@/lib/auth/fake-tab";
import { rowOf } from "@/store/planning/fake-cycles";
import { cycleOf, projectOf } from "@/store/project/fake-projects";
import { CycleActionsHost } from "./actions-host";
import { TransferIssues, TransferIssuesButton } from "./transfer-issues";

// The pages of a project's cycles as the transfer of an ended cycle's work items is on them (P5a spec 2.5), for its
// tests: its list, each row with its transfer, and a cycle's own page, with its banner; each the page of its
// CycleActionsHost, which holds the transfer's dialog. The cycles are a stand-in store's (transferCycles), which a test
// file gives the pages by mocking @/hooks/store/use-cycles with it.

/**
 * A stand-in for the cycle store over rows: a cycle, and its row, by its id; a project's cycles as its list was last
 * read, those since archived or ended by their own reads too; its transfers go to nerve, which settles the last one
 * sent when the test says (answer, refuse); and nerve answers the store's reads again (reread) when the test says.
 */
export function transferCycles(rows: CycleListItem[]) {
  const shown = observable.map(
    rows.map((row) => [row.id, row]),
    { deep: false }
  );
  const sent: HeldChange<CycleTransferResult>[] = [];
  const of = (id: string | null | undefined) => (id ? shown.get(id) : undefined);
  return {
    getCycleById: of,
    getCycleRow: of,
    getProjectCycleIds: (projectId: string | null | undefined) =>
      [...shown.values()].filter((row) => row.project_id === projectId).map((row) => row.id),
    transfer: vi.fn<(cycleId: string, targetCycleId: string) => Promise<CycleTransferResult>>(() => {
      const change = heldChange<CycleTransferResult>();
      sent.push(change);
      return change.sent;
    }),
    /** nerve answers the transfer sent last: it moved count work items. */
    answer: async (count: number) => {
      await act(async () => sent.at(-1)?.answer({ transferred_issues: count }));
      await settle();
    },
    /** nerve refuses the transfer sent last, by error. */
    refuse: async (error: unknown) => {
      await act(async () => sent.at(-1)?.refuse(error));
      await settle();
    },
    /** The store reads the cycle again: its row is now of fields. */
    reread: (id: string, fields: Partial<CycleListItem>) =>
      act(async () => runInAction(() => shown.set(id, { ...shown.get(id)!, ...fields }))),
  };
}

const ARCHIVED = "2026-11-02T09:00:00Z";
/** What an ended cycle was frozen with: counts of the five groups, their sum the total. */
function frozen(counts: Partial<FrozenCounts>): FrozenCounts {
  const groups = {
    backlog_issues: 0,
    unstarted_issues: 0,
    started_issues: 0,
    completed_issues: 0,
    cancelled_issues: 0,
  };
  const all = { ...groups, ...counts };
  const unfinished = all.backlog_issues + all.unstarted_issues + all.started_issues;
  const total = unfinished + all.completed_issues + all.cancelled_issues;
  return { ...all, total_issues: total, frozen_at: "2026-11-02T00:00:00Z" };
}

/**
 * web's cycles, in nerve's order, and one of another project's: Sprint, ended, and Frozen, ended and frozen with all
 * done, have work items unfinished; Done, ended, has none now, though it was frozen with some; Shelved, ended, is
 * archived; Now is current, Next upcoming, Later upcoming and archived, Idea a draft.
 */
export function cyclesOf(web: Project): CycleListItem[] {
  const ended = { status: "completed" } as const;
  const ops = projectOf("OPS", web.workspace_id);
  return [
    // 1 in the backlog, 2 not started and 1 started
    rowOf(cycleOf(web, "Sprint", ended), {
      total_issues: 6,
      backlog_issues: 1,
      unstarted_issues: 2,
      started_issues: 1,
      completed_issues: 1,
      cancelled_issues: 1,
    }),
    // 2 started since it was frozen
    rowOf(cycleOf(web, "Frozen", ended), {
      total_issues: 3,
      started_issues: 2,
      completed_issues: 1,
      progress_snapshot: frozen({ completed_issues: 3 }),
    }),
    rowOf(cycleOf(web, "Done", ended), {
      total_issues: 2,
      completed_issues: 1,
      cancelled_issues: 1,
      progress_snapshot: frozen({ unstarted_issues: 3 }),
    }),
    rowOf(cycleOf(web, "Shelved", { ...ended, archived_at: ARCHIVED }), { total_issues: 3, unstarted_issues: 3 }),
    rowOf(cycleOf(web, "Now", { status: "current" }), { total_issues: 2, started_issues: 2 }),
    rowOf(cycleOf(web, "Next", { status: "upcoming" })),
    rowOf(cycleOf(web, "Later", { status: "upcoming", archived_at: ARCHIVED })),
    rowOf(cycleOf(web, "Idea")),
    rowOf(cycleOf(ops, "Elsewhere", { status: "upcoming" })),
  ];
}

/** The project's list: each of its cycles a row, named by its name, with the row's transfer. */
const ListPage = observer(function ListPage() {
  const { workspaceSlug = "", projectId = "" } = useParams();
  const { getProjectCycleIds, getCycleRow } = useCycles();
  return (getProjectCycleIds(projectId) ?? []).map((id) => {
    const row = getCycleRow(id);
    return (
      row && (
        <div key={id} data-row={row.name}>
          <TransferIssuesButton row={row} workspaceSlug={workspaceSlug} projectId={projectId} />
        </div>
      )
    );
  });
});

/** A cycle's own page: the banner of its transfer. */
function OwnPage() {
  const { workspaceSlug = "", projectId = "", cycleId = "" } = useParams();
  return <TransferIssues cycleId={cycleId} workspaceSlug={workspaceSlug} projectId={projectId} />;
}

function Page({ children }: { children: ReactNode }) {
  const { workspaceSlug = "", projectId = "" } = useParams();
  return (
    <CycleActionsHost workspaceSlug={workspaceSlug} projectId={projectId}>
      {children}
    </CycleActionsHost>
  );
}

/** Where the pages are: the router's navigate, which moves them as a Back or a link would. */
export const where: { navigate?: NavigateFunction } = {};

function Where() {
  where.navigate = useNavigate();
  return null;
}

/** The pages of the project's cycles, at path; anywhere else, "elsewhere". */
export function TransferPages({ path }: { path: string }) {
  return (
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path=":workspaceSlug/projects/:projectId/cycles"
          element={
            <Page>
              <ListPage />
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
        <Route path="*" element={<p>elsewhere</p>} />
      </Routes>
      <Where />
    </MemoryRouter>
  );
}
