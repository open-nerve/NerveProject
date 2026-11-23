/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IntakeIssueCreate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { inTurn } from "@/store/fake-queue";
import { issuePageOf } from "@/store/issue/fake-issues";
import {
  acme,
  countsOf,
  countsPath,
  detailOf,
  intakeIssueOf,
  listPath,
  loadCounts,
  loadList,
  loadProjectList,
  open,
  pageOf,
  writesTab,
} from "@/store/issue/intake/fake-intake";
import type { IntakeFilters } from "@/store/issue/intake/filters";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

// The submission of a work item to a project's intake (P5a spec 2.3, R8, R30). It is no creation of a project's work
// item, and there is no work item yet to queue its write behind: it is sent at once, and nerve's answer goes to the
// caller, which opens it. No list takes it in a place of its own: each list held of the project's intake reads its
// first page again, in place of every page held, and the counts are fetched again where this generation holds them;
// neither holds the caller up, and neither fails the submission, which nerve made. No project list and no canonical
// work item takes it: it is in its project's triage state, out of the project's lists. A refusal is handed on, every
// value as it was.

const closed: IntakeFilters = { ...open, tab: "closed", statuses: ["accepted", "declined", "duplicate"] };
const [x, y, z, fresh] = ["x", "y", "z", "new"].map((name) => intakeIssueOf(name));
const theirs = pageOf([intakeIssueOf("api-one", { project_id: "p-api" })]);
const body: IntakeIssueCreate = {
  name: "new",
  description_html: "<p>new</p>",
  priority: "high",
  label_ids: [],
  assignee_ids: [],
};

let tab: Awaited<ReturnType<typeof writesTab>>;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date("2026-11-10T09:00:00Z") });
  tab = await writesTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

/**
 * p-web's open tab held as two pages (x, y; then z), its closed tab (none), p-api's open tab, acme's counts and a list
 * of p-web's work items; the requests so far forgotten. Gives the project list.
 */
async function held() {
  const { nerve, store } = tab;
  const project = await loadProjectList(tab, {}, issuePageOf([]));
  await loadList(nerve, store, "p-web", open, pageOf([x, y], "c-1", 3));
  const more = () => store.fetchMore("p-web", open);
  await answered(nerve, more, ["GET", listPath("p-web")], pageOf([z], null, 3), "the next page");
  await loadList(nerve, store, "p-web", closed, pageOf([]));
  await loadList(nerve, store, "p-api", open, theirs);
  await loadCounts(nerve, store, acme, countsOf({ "p-web": 3 }));
  nerve.calls.length = 0;
  return project;
}

/** The requests after the submission: each as its method, its path, its query's status and cursor, in path order. */
const readsAgain = () =>
  tab.nerve.calls
    .slice(1)
    .map((call) => [call.method, call.path, call.query.status, call.query.cursor])
    .toSorted((a, b) => `${a[1]} ${a[2]}`.localeCompare(`${b[1]} ${b[2]}`));

it("gives nerve's answer, inserts no row, reads each list held of the project again from its first page, and the counts (R30)", async () => {
  const project = await held();
  const before = { theirs: tab.store.getList("p-api", open), project: project.value };
  const submitted = detailOf("new");
  const submitting = track(tab.store.submit(tab.scope, body));
  await inTurn(tab.nerve, 0, ["POST", listPath("p-web")], json(201, submitted));
  expect(tab.nerve.calls[0]?.body).toEqual(body);
  await until(() => submitting.settled && tab.nerve.calls.length === 4, "the submission and the reads again");
  expect(submitting.value).toEqual(submitted);
  // no list takes it in a place of its own: each holds what it did until its read answers
  expect(tab.store.getList("p-web", open)).toEqual(pageOf([x, y, z], null, 3));
  expect(readsAgain()).toEqual([
    ["GET", listPath("p-web"), "accepted,declined,duplicate", undefined],
    ["GET", listPath("p-web"), "pending", undefined],
    ["GET", countsPath(acme), undefined, undefined],
  ]);

  const read = (status: string) => tab.nerve.calls.find((call) => call.query.status === status)!;
  read("pending").answer(json(200, pageOf([fresh, x], "c-2", 4)));
  read("accepted,declined,duplicate").answer(json(200, pageOf([])));
  tab.nerve.to(countsPath(acme))[0]!.answer(json(200, countsOf({ "p-web": 4 })));
  await until(() => tab.store.pendingCount(acme.slug, "p-web") === 4, "the counts again");
  await until(() => tab.store.getList("p-web", open)?.total === 4, "the first page again");
  // the first page in place of both pages held
  expect(tab.store.getList("p-web", open)).toEqual(pageOf([fresh, x], "c-2", 4));
  expect(tab.store.getList("p-web", closed)).toEqual(pageOf([]));
  expect(tab.store.getList("p-api", open), "another project's list is not read again").toBe(before.theirs);
  // in its project's triage state: no project list, no canonical work item takes it
  expect(project.value).toBe(before.project);
  expect(tab.root.issues.getIssueById(submitted.id)).toBeUndefined();
  await vi.advanceTimersByTimeAsync(1_000);
  expect(tab.nerve.calls).toHaveLength(4);
});

it("fails no submission on a read again nerve refuses: the lists and the counts as they were", async () => {
  await held();
  const before = { open: tab.store.getList("p-web", open), closed: tab.store.getList("p-web", closed) };
  const submitting = track(tab.store.submit(tab.scope, body));
  await inTurn(tab.nerve, 0, ["POST", listPath("p-web")], json(201, detailOf("new")));
  await until(() => submitting.settled && tab.nerve.calls.length === 4, "the submission and the reads again");
  for (const call of tab.nerve.calls.slice(1)) call.answer(problem(503, "server_busy"));
  await vi.advanceTimersByTimeAsync(1_000);
  expect(submitting.error).toBeUndefined();
  expect(tab.store.getList("p-web", open)).toBe(before.open);
  expect(tab.store.getList("p-web", closed)).toBe(before.closed);
  expect(tab.store.pendingCount(acme.slug, "p-web")).toBe(3);
});

it("hands nerve's refusal on: nothing read again, no counts, every value as it was", async () => {
  const project = await held();
  const before = { open: tab.store.getList("p-web", open), project: project.value };
  const submitting = track(tab.store.submit(tab.scope, body));
  await inTurn(tab.nerve, 0, ["POST", listPath("p-web")], problem(409, "issue.intake_disabled"));
  await until(() => submitting.settled, "the refusal");
  expect(submitting.error).toBeInstanceOf(ApiError);
  expect(submitting.error).toMatchObject({ status: 409, problem: { code: "issue.intake_disabled" } });
  await vi.advanceTimersByTimeAsync(1_000);
  expect(tab.nerve.calls).toHaveLength(1);
  expect(tab.store.getList("p-web", open)).toBe(before.open);
  expect(tab.store.pendingCount(acme.slug, "p-web")).toBe(3);
  expect(project.value).toBe(before.project);
});
