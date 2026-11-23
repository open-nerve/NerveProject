/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IntakeTriage, IssueIntake } from "@nerve/api-client";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { inTurn } from "@/store/fake-queue";
import { issueOf, issuePageOf, requestedIssue } from "@/store/issue/fake-issues";
import {
  acme,
  countsOf,
  countsPath,
  detailOf,
  holdDetail,
  intakeIssueOf,
  loadCounts,
  loadList,
  loadProjectList,
  open,
  pageOf,
  shownList,
  snoozedToo,
  writesTab,
} from "@/store/issue/intake/fake-intake";
import type { IntakeFilters } from "@/store/issue/intake/filters";
import { loadProjects, projectOf } from "@/store/project/fake-projects";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

// The writes of a work item of an intake that read it again in their turn (P5a spec 2.4, R4, R34; re-94 I1). A triage:
// nerve's answer goes to the intake lists, then the work item is read again from after the triage, so that a read sent
// before it cannot undo it (a triage but an acceptance moves no updated_at); canonical takes that read, and the counts
// are fetched again. An acceptance is confirmed as a creation: the read again goes into every project list held, once,
// whatever their reads out or answered meanwhile. A read again that gives nothing takes the work item out of every
// intake list; one that fails leaves the error the work item's own; neither fails the triage. A deletion: its intake row
// is gone, so the work item leaves every intake list, and it is read again, as nerve deletes an accepted one's intake
// row alone; only a read that finds none deletes it from canonical.

const NOW = "2026-11-10T09:00:00Z";
const TOMORROW = "2026-11-11T08:00:00Z";
const at = (n: number) => `2026-11-10T00:00:0${n}Z`;
const intake = (status: IssueIntake["status"], fields: Partial<IssueIntake> = {}): IssueIntake => ({
  status,
  snoozed_till: null,
  duplicate_to_id: null,
  ...fields,
});
const pending = intake("pending");
const snoozed = intake("snoozed", { snoozed_till: TOMORROW });
const accepted = intake("accepted");
const declined = intake("declined");
const duplicate = intake("duplicate", { duplicate_to_id: "i-y" });
const closed: IntakeFilters = { ...open, tab: "closed", statuses: ["accepted", "declined", "duplicate"] };
const every: IntakeFilters = { ...open, statuses: ["pending", "snoozed", "accepted", "declined", "duplicate"] };
const [x, y] = [intakeIssueOf("x"), intakeIssueOf("y")];
const ISSUE = "/api/v0/issues/i-x";
const DELETE_PATH = "/api/v0/intake-issues/i-x";
const TRIAGE = `${DELETE_PATH}/triage`;

let tab: Awaited<ReturnType<typeof writesTab>>;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date(NOW) });
  tab = await writesTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

const triage = (body: IntakeTriage) => track(tab.root.writes.triageIntake("i-x", body, tab.scope));

/** p-web's open tab (x, y) and closed tab (none) held, and acme's counts; the requests so far forgotten. */
async function intakeHeld() {
  await loadList(tab.nerve, tab.store, "p-web", open, pageOf([x, y]));
  await loadList(tab.nerve, tab.store, "p-web", closed, pageOf([]));
  await loadCounts(tab.nerve, tab.store, acme, countsOf({ "p-web": 2 }));
  tab.nerve.calls.length = 0;
}

/** The triage's kinds: the body sent, the intake row held before it, the one nerve answers. */
const kinds: [string, IntakeTriage, IssueIntake, IssueIntake][] = [
  ["declined", { status: "declined" }, pending, declined],
  ["snoozed", { status: "snoozed", snoozed_till: TOMORROW }, pending, snoozed],
  ["duplicate", { status: "duplicate", duplicate_to_id: "i-y" }, pending, duplicate],
  ["pending", { status: "pending" }, snoozed, pending],
  ["accepted", { status: "accepted" }, pending, accepted],
];
/** nerve's answer to a triage to row: an acceptance moves the work item to its default state, at its moment. */
const triaged = (row: IssueIntake) =>
  detailOf(
    "x",
    row.status === "accepted" ? { intake: row, state_id: "s-default", updated_at: at(1) } : { intake: row }
  );

it.each(kinds)(
  "%s: reads the work item again in its turn, from after the triage, so that a read sent before cannot undo it",
  async (_, body, from, to) => {
    holdDetail(tab, { intake: from });
    const old = await requestedIssue(tab.nerve, () => tab.root.issues.refreshAuthoritative("i-x"), ["GET", ISSUE]);
    const triaging = triage(body);
    await inTurn(tab.nerve, 1, ["POST", TRIAGE], json(200, triaged(to)));
    expect(tab.nerve.calls[1]?.body).toEqual(body);
    await until(() => tab.nerve.to(ISSUE).length === 2, "the read again");
    expect(triaging.settled, "the triage waits for its read again").toBe(false);
    old.call.answer(json(200, detailOf("x", { intake: from })));
    await old.promise;
    tab.nerve.to(ISSUE)[1]!.answer(json(200, triaged(to)));
    await until(() => triaging.settled, "the triage");
    expect(triaging.error).toBeUndefined();
    expect(tab.root.issues.getIssueDetailById("i-x")?.intake).toEqual(to);
  }
);

it("accepts as a creation: a project list held shows the work item read again, once, the list not read again", async () => {
  holdDetail(tab);
  const project = await loadProjectList(tab, {}, issuePageOf([]));
  await loadList(tab.nerve, tab.store, "p-web", open, pageOf([x, y]));
  await loadList(tab.nerve, tab.store, "p-web", closed, pageOf([]));
  await loadCounts(tab.nerve, tab.store, acme, countsOf({ "p-web": 2 }));
  tab.nerve.calls.length = 0;
  const accepting = triage({ status: "accepted" });
  await inTurn(tab.nerve, 0, ["POST", TRIAGE], json(200, triaged(accepted)));
  await inTurn(tab.nerve, 1, ["GET", ISSUE], json(200, { ...triaged(accepted), name: "read again" }));
  await until(() => accepting.settled, "the acceptance");
  expect(accepting.error).toBeUndefined();
  expect(shownList(project)).toEqual({ total: 1, groups: [["", 1, ["i-x"]]] });
  expect(tab.root.issues.getIssueById("i-x")?.name).toBe("read again");
  expect(tab.store.getList("p-web", open)).toEqual(pageOf([y], null, 1));
  expect(tab.store.getList("p-web", closed), "a list that lacks it gets none").toEqual(pageOf([]));
  await until(() => tab.nerve.to(countsPath(acme)).length === 1, "the counts again");
  expect(tab.nerve.to("/api/v0/projects/p-web/issues")).toEqual([]);
});

it("accepts once into a project list whose read is out and answers with the work item", async () => {
  holdDetail(tab);
  const project = await loadProjectList(tab, {}, issuePageOf([]));
  tab.nerve.calls.length = 0;
  const reading = track(project.fetchFirst());
  await until(() => tab.nerve.calls.length === 1, "the list's read");
  const accepting = triage({ status: "accepted" });
  await inTurn(tab.nerve, 1, ["POST", TRIAGE], json(200, triaged(accepted)));
  await inTurn(tab.nerve, 2, ["GET", ISSUE], json(200, triaged(accepted)));
  await until(() => accepting.settled, "the acceptance");
  expect(shownList(project)).toEqual({ total: 1, groups: [["", 1, ["i-x"]]] });
  const { description_html, is_subscribed, intake: row, ...listed } = triaged(accepted);
  tab.nerve.calls[0]!.answer(json(200, issuePageOf([listed])));
  await until(() => reading.settled, "the list's read");
  expect(shownList(project)).toEqual({ total: 1, groups: [["", 1, ["i-x"]]] });
});

it("accepts once into a project list read after the acceptance was sent and answered before it", async () => {
  holdDetail(tab);
  const project = await loadProjectList(tab, {}, issuePageOf([]));
  tab.nerve.calls.length = 0;
  const accepting = triage({ status: "accepted" });
  await until(() => tab.nerve.calls.length === 1, "the acceptance");
  const { description_html, is_subscribed, intake: row, ...listed } = triaged(accepted);
  const read = await requestedIssue(tab.nerve, () => project.fetchFirst(), ["GET", "/api/v0/projects/p-web/issues"]);
  read.call.answer(json(200, issuePageOf([listed])));
  await read.promise;
  expect(shownList(project)).toEqual({ total: 1, groups: [["", 1, ["i-x"]]] });
  tab.nerve.calls[0]!.answer(json(200, triaged(accepted)));
  await inTurn(tab.nerve, 2, ["GET", ISSUE], json(200, triaged(accepted)));
  await until(() => accepting.settled, "the acceptance");
  expect(accepting.error).toBeUndefined();
  expect(shownList(project)).toEqual({ total: 1, groups: [["", 1, ["i-x"]]] });
});

it("accepts into an incomplete project list as a new work item: one more in its total, never one it had", async () => {
  holdDetail(tab);
  const other = { ...issueOf("i-z", "p-web", acme.id), sort_order: 500 };
  const project = await loadProjectList(tab, {}, { ...issuePageOf([other], "c-2"), total: 2 });
  tab.nerve.calls.length = 0;
  const accepting = triage({ status: "accepted" });
  await inTurn(tab.nerve, 0, ["POST", TRIAGE], json(200, triaged(accepted)));
  await inTurn(tab.nerve, 1, ["GET", ISSUE], json(200, triaged(accepted)));
  await until(() => accepting.settled, "the acceptance");
  expect(accepting.error).toBeUndefined();
  expect(shownList(project), "after the rows loaded, counted once").toEqual({ total: 3, groups: [["", 3, ["i-z"]]] });
});

it("accepts into a project list read after the acceptance was sent: its total as that read counted it", async () => {
  holdDetail(tab);
  const project = await loadProjectList(tab, {}, issuePageOf([]));
  tab.nerve.calls.length = 0;
  const accepting = triage({ status: "accepted" });
  await until(() => tab.nerve.calls.length === 1, "the acceptance");
  const other = { ...issueOf("i-z", "p-web", acme.id), sort_order: 500 };
  const read = await requestedIssue(tab.nerve, () => project.fetchFirst(), ["GET", "/api/v0/projects/p-web/issues"]);
  read.call.answer(json(200, { ...issuePageOf([other], "c-2"), total: 2 }));
  await read.promise;
  tab.nerve.calls[0]!.answer(json(200, triaged(accepted)));
  await inTurn(tab.nerve, 2, ["GET", ISSUE], json(200, triaged(accepted)));
  await until(() => accepting.settled, "the acceptance");
  expect(accepting.error).toBeUndefined();
  expect(project.value?.total, "the read may have counted it already: not one more").toBe(2);
});

it.each(kinds.filter(([kind]) => kind !== "accepted"))(
  "%s confirms nerve's answer to the intake lists alone, reads the detail's activity again, counts again",
  async (_, body, from, to) => {
    holdDetail(tab, { intake: from });
    const other = { ...issueOf("i-z", "p-web", acme.id) };
    const project = await loadProjectList(tab, {}, issuePageOf([other]));
    const held = { ...x, intake: from };
    await loadList(tab.nerve, tab.store, "p-web", open, pageOf([held, y]));
    await loadList(tab.nerve, tab.store, "p-web", snoozedToo, pageOf([held, y]));
    await loadList(tab.nerve, tab.store, "p-web", closed, pageOf([]));
    await loadCounts(tab.nerve, tab.store, acme, countsOf({ "p-web": 2 }));
    tab.nerve.calls.length = 0;
    const activity = vi.spyOn(tab.root.issueDetail, "refreshAfterChange");
    const triaging = triage(body);
    await inTurn(tab.nerve, 0, ["POST", TRIAGE], json(200, triaged(to)));
    await inTurn(tab.nerve, 1, ["GET", ISSUE], json(200, triaged(to)));
    await until(() => triaging.settled, "the triage");
    expect(triaging.error).toBeUndefined();
    expect(shownList(project), "no project list takes it").toEqual({ total: 1, groups: [["", 1, ["i-z"]]] });
    const listed = (filters: IntakeFilters) =>
      tab.store.getList("p-web", filters)?.data.map((row) => [row.id, row.intake.status]);
    const stays = (filters: IntakeFilters) => filters.statuses.includes(to.status);
    expect(listed(open)).toEqual(
      stays(open)
        ? [
            ["i-x", to.status],
            ["i-y", "pending"],
          ]
        : [["i-y", "pending"]]
    );
    expect(listed(snoozedToo)).toEqual(
      stays(snoozedToo)
        ? [
            ["i-x", to.status],
            ["i-y", "pending"],
          ]
        : [["i-y", "pending"]]
    );
    expect(listed(closed), "a list that lacks it gets none").toEqual([]);
    expect(activity).toHaveBeenCalledWith(acme.slug, "p-web", "i-x");
    await until(() => tab.nerve.to(countsPath(acme)).length === 1, "the counts again");
  }
);

it.each(["snoozed", "accepted"] as const)(
  "%s: a read again that gives nothing takes the work item out of every intake list, puts it in no project list",
  async (status) => {
    holdDetail(tab);
    const project = await loadProjectList(tab, {}, issuePageOf([]));
    await loadList(tab.nerve, tab.store, "p-web", every, pageOf([x, y]));
    tab.nerve.calls.length = 0;
    const activity = vi.spyOn(tab.root.issueDetail, "refreshAfterChange");
    const body: IntakeTriage = status === "snoozed" ? { status, snoozed_till: TOMORROW } : { status };
    const triaging = triage(body);
    await inTurn(tab.nerve, 0, ["POST", TRIAGE], json(200, triaged(status === "snoozed" ? snoozed : accepted)));
    await inTurn(tab.nerve, 1, ["GET", ISSUE], problem(404, "issue.not_found"));
    await until(() => triaging.settled, "the triage");
    expect(triaging.error).toBeUndefined();
    expect(tab.store.getList("p-web", every)).toEqual(pageOf([y], null, 1));
    expect(shownList(project)).toEqual({ total: 0, groups: [["", 0, []]] });
    expect(activity).not.toHaveBeenCalled();
  }
);

it.each(["declined", "accepted"] as const)(
  "%s: a read again that fails leaves the error the work item's own, fails no triage, confirms no project list",
  async (status) => {
    holdDetail(tab);
    const project = await loadProjectList(tab, {}, issuePageOf([]));
    tab.nerve.calls.length = 0;
    const triaging = triage({ status });
    await inTurn(tab.nerve, 0, ["POST", TRIAGE], json(200, triaged(intake(status))));
    await inTurn(tab.nerve, 1, ["GET", ISSUE], problem(500, "internal"));
    await until(() => triaging.settled, "the triage");
    expect(triaging.error).toBeUndefined();
    expect(tab.root.verification.getError("i-x")).toMatchObject({ status: 500 });
    expect(shownList(project)).toEqual({ total: 0, groups: [["", 0, []]] });
  }
);

it("hands on a change of session during a triage's read again, the counts not fetched again", async () => {
  holdDetail(tab);
  await loadCounts(tab.nerve, tab.store, acme, countsOf({ "p-web": 2 }));
  tab.nerve.calls.length = 0;
  const triaging = triage({ status: "declined" });
  await inTurn(tab.nerve, 0, ["POST", TRIAGE], json(200, triaged(declined)));
  await until(() => tab.nerve.to(ISSUE).length === 1, "the read again");
  tab.root.retire();
  tab.nerve.to(ISSUE)[0]!.answer(json(200, triaged(declined)));
  await until(() => triaging.settled, "the triage");
  expect(triaging.error).toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.to(countsPath(acme))).toEqual([]);
});

it("deletes a work item of an intake through its intake, reads it again in its turn: found none, canonical deletes it", async () => {
  holdDetail(tab);
  const project = await loadProjectList(tab, {}, issuePageOf([]));
  await intakeHeld();
  const change = track(tab.root.writes.deleteIntake("i-x", tab.scope));
  await inTurn(tab.nerve, 0, ["DELETE", DELETE_PATH], noContent());
  await inTurn(tab.nerve, 1, ["GET", ISSUE], problem(404, "issue.not_found"));
  await until(() => change.settled, "the deletion");
  expect(change.error).toBeUndefined();
  expect(tab.root.issues.held("i-x")).toBeNull();
  expect(tab.store.getList("p-web", open)).toEqual(pageOf([y], null, 1));
  expect(shownList(project), "no project list is touched").toEqual({ total: 0, groups: [["", 0, []]] });
  await until(() => tab.nerve.to(countsPath(acme)).length === 1, "the counts again");
});

it("deletes the intake row alone of a work item another accepted meanwhile: canonical takes it as read again", async () => {
  holdDetail(tab);
  const project = await loadProjectList(tab, {}, issuePageOf([]));
  await loadList(tab.nerve, tab.store, "p-web", open, pageOf([x, y]));
  await loadList(tab.nerve, tab.store, "p-web", closed, pageOf([{ ...x, intake: accepted }]));
  tab.nerve.calls.length = 0;
  const theirs = detailOf("x", { intake: null, state_id: "s-default", updated_at: at(1) });
  const change = track(tab.root.writes.deleteIntake("i-x", tab.scope));
  await inTurn(tab.nerve, 0, ["DELETE", DELETE_PATH], noContent());
  await inTurn(tab.nerve, 1, ["GET", ISSUE], json(200, theirs));
  await until(() => change.settled, "the deletion");
  expect(change.error).toBeUndefined();
  expect(tab.root.issues.getIssueDetailById("i-x")).toEqual(theirs);
  expect(tab.store.getList("p-web", open)).toEqual(pageOf([y], null, 1));
  expect(tab.store.getList("p-web", closed)).toEqual(pageOf([], null, 0));
  expect(shownList(project), "no project list is touched").toEqual({ total: 0, groups: [["", 0, []]] });
});

it("deletes nothing from canonical on a deletion's read again the permission scope overtook: only one that finds none", async () => {
  holdDetail(tab);
  await intakeHeld();
  const change = track(tab.root.writes.deleteIntake("i-x", tab.scope));
  await inTurn(tab.nerve, 0, ["DELETE", DELETE_PATH], noContent());
  await until(() => tab.nerve.to(ISSUE).length === 1, "the read again");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [projectOf("WEB", acme.id, { member_role: 20 })]);
  tab.nerve.to(ISSUE)[0]!.answer(json(200, detailOf("x", { intake: null, updated_at: at(1) })));
  await until(() => change.settled, "the deletion");
  expect(change.error).toBeUndefined();
  expect(tab.root.issues.held("i-x")).not.toBeNull();
  expect(tab.store.getList("p-web", open)).toEqual(pageOf([y], null, 1));
});
