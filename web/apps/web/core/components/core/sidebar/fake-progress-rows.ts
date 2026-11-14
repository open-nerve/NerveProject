// Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only
// The rows of a progress's breakdowns as a page shows them (P5b spec 2.8, P6b spec 2.7), for the tests of the pages
// whose rows choose (a cycle's or a module's page, the card of the cycle in progress): a progress with a row of each
// kind, the project's two labels that name its rows, and how a test finds a row, reads the rows and opens a tab.
import { act } from "react";
import type { Progress } from "@nerve/api-client";
import type { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { answered } from "@/lib/auth/fake-nerve";
import { labelOf } from "@/store/project/fake-projects";

export const ANN = "019a0000-0000-7000-8000-0000000000a1";
export const BUG = "019a0000-0000-7000-8000-0000000000b1";
export const UI = "019a0000-0000-7000-8000-0000000000b2";

/** A progress with a row of each kind: work items backlog, started, completed; Ann's, none's; Bug's, UI's, none's. */
export const progressRows: Partial<Progress> = {
  total_issues: 6,
  backlog_issues: 1,
  started_issues: 3,
  completed_issues: 2,
  assignees: [
    { assignee_id: ANN, total_issues: 4, completed_issues: 1, pending_issues: 3 },
    { assignee_id: null, total_issues: 2, completed_issues: 1, pending_issues: 1 },
  ],
  labels: [
    { label_id: BUG, total_issues: 3, completed_issues: 1, pending_issues: 2 },
    { label_id: UI, total_issues: 2, completed_issues: 1, pending_issues: 1 },
    { label_id: null, total_issues: 1, completed_issues: 0, pending_issues: 1 },
  ],
};

type Tab = Pick<Awaited<ReturnType<typeof issueSelectionTab>>, "nerve" | "app" | "project">;
/** The tab's store reads its project's labels, and nerve gives Bug and UI, which name the progress's label rows. */
export function loadRowLabels(tab: Tab) {
  const labels = [labelOf(tab.project, "Bug", 1, { id: BUG }), labelOf(tab.project, "UI", 2, { id: UI })];
  return answered(
    tab.nerve,
    () => tab.app.label.fetchProjectLabels(tab.project.id),
    ["GET", `/api/v0/projects/${tab.project.id}/labels`],
    { data: labels },
    "the labels"
  );
}

/** The button of a row within a part of the page (or the page), whose text has name. */
export const rowButton = (name: string, within: ParentNode = document) =>
  [...within.querySelectorAll<HTMLButtonElement>("li button")].find((one) => one.textContent?.includes(name));

/** The rows within a part of the page (or the page), each by whether it is a button, and its name before its share. */
export const rowsIn = (within: ParentNode = document) =>
  [...within.querySelectorAll("li")].map((row) => [
    row.querySelector("button") !== null,
    row.textContent?.split(/\d/)[0],
  ]);

/** The part of the page labelled label. */
export const partOf = (label: string) => document.querySelector<HTMLElement>(`[aria-label="${label}"]`);

/** Clicks the tab named name within a part of the page (or the page). */
export const openTab = (name: string, within: ParentNode = document) =>
  act(async () =>
    [...within.querySelectorAll<HTMLButtonElement>('[role="tab"]')].find((one) => one.textContent === name)?.click()
  );
