/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueDetail, IssueIntake, IssueListPage } from "@nerve/api-client";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { ProjectIssueQuery } from "@/services/issue/issue-list.service";
import { inTurn, sent } from "@/store/fake-queue";
import { issueCommentOf, issueOf, issuePageOf, requestedIssue } from "@/store/issue/fake-issues";
import { IssueListBinding } from "@/store/issue/helpers/issue-list-binding";
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
  writesTab,
} from "@/store/issue/intake/fake-intake";
import type { IntakeFilters } from "@/store/issue/intake/filters";
import { loadProjects, projectOf, stateOf } from "@/store/project/fake-projects";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

// The writes of a work item of an intake (P5a spec 2.4, R3, R28, R33; re-94 S1, m1, m2): each goes through the work
// item's one queue of M4's writes, in turn with its comments' and the others'; where a content change goes is decided
// in its turn, by the detail then held. A work item in an intake is written beside write(): no directory of a project
// list is read, no project list is confirmed or marked unavailable; its intake lists take nerve's answer; the counts
// are fetched again after a triage or a deletion, never a content change. A refusal leaves every value as it was; one
// that says the work item left its intake, or that another triaged it first, is followed in the refused write's turn.

const at = (n: number) => `2026-11-10T00:00:0${n}Z`;
const pending: IssueIntake = { status: "pending", snoozed_till: null, duplicate_to_id: null };
const accepted: IssueIntake = { status: "accepted", snoozed_till: null, duplicate_to_id: null };
const declined: IssueIntake = { status: "declined", snoozed_till: null, duplicate_to_id: null };
const closed: IntakeFilters = { ...open, tab: "closed", statuses: ["accepted", "declined", "duplicate"] };
const [x, y] = [intakeIssueOf("x"), intakeIssueOf("y")];
const ISSUE = "/api/v0/issues/i-x";
const INTAKE = "/api/v0/intake-issues/i-x";

let tab: Awaited<ReturnType<typeof writesTab>>;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date("2026-11-10T09:00:00Z") });
  tab = await writesTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

const held = (fields: Partial<IssueDetail> = {}) => holdDetail(tab, fields);
const projectList = (query: ProjectIssueQuery, page: IssueListPage) => loadProjectList(tab, query, page);

/** p-web's open tab (x, y) and closed tab (none) held, and acme's counts; the requests so far forgotten. */
async function intakeHeld() {
  await loadList(tab.nerve, tab.store, "p-web", open, pageOf([x, y]));
  await loadList(tab.nerve, tab.store, "p-web", closed, pageOf([]));
  await loadCounts(tab.nerve, tab.store, acme, countsOf({ "p-web": 2 }));
  tab.nerve.calls.length = 0;
}

/** Answers the k-th request to method path with response, once it is out. */
async function reply([method, path]: Endpoint, response: Response, k = 0) {
  const calls = () => tab.nerve.calls.filter((call) => call.method === method && call.path === path);
  await until(() => calls().length > k, `${method} ${path}`);
  calls()[k]!.answer(response);
}

type Kind = "edit" | "decline" | "delete";
const send = (kind: Kind) => (): Promise<unknown> =>
  kind === "edit"
    ? tab.root.writes.updateIssue("i-x", () => ({ name: "renamed" }), undefined, tab.scope)
    : kind === "decline"
      ? tab.root.writes.triageIntake("i-x", { status: "declined" }, tab.scope)
      : tab.root.writes.deleteIntake("i-x", tab.scope);
const request = (kind: Kind): Endpoint =>
  kind === "edit" ? ["PATCH", INTAKE] : kind === "decline" ? ["POST", `${INTAKE}/triage`] : ["DELETE", INTAKE];
const done = (kind: Kind): Response =>
  kind === "edit"
    ? json(200, detailOf("x", { name: "renamed", updated_at: at(1) }))
    : kind === "decline"
      ? json(200, detailOf("x", { intake: declined }))
      : noContent();

it("sends a change of a work item held as a list's row alone to /issues: no list's row is of an intake", async () => {
  tab.root.acceptListIssues([issueOf("i-x", "p-web", acme.id)], tab.root.captureRead());
  const change = () => tab.root.writes.updateIssue("i-x", () => ({ name: "renamed" }), undefined, tab.scope);
  const out = await requestedIssue(tab.nerve, change, ["PATCH", ISSUE]);
  out.call.answer(json(200, detailOf("x", { intake: null, name: "renamed", updated_at: at(1) })));
  await out.promise;
});

it("sends a content change of a work item held in an intake to its intake, confirmed to canonical and its intake lists alone", async () => {
  held();
  const project = await projectList({}, issuePageOf([]));
  await intakeHeld();
  const before = project.value;
  const renamed = detailOf("x", { name: "renamed", priority: "high", updated_at: at(1) });
  const change = await sent(
    tab.nerve,
    () => tab.root.issueDetail.updateIssue(acme.slug, "p-web", "i-x", { name: "renamed", priority: "high" }),
    ["PATCH", INTAKE],
    json(200, renamed)
  );
  expect(change.error).toBeUndefined();
  expect(tab.nerve.calls[0]?.body).toEqual({ name: "renamed", priority: "high" });
  expect(tab.root.issues.getIssueDetailById("i-x")).toEqual(renamed);
  expect(tab.store.getList("p-web", open)?.data.map((row) => [row.id, row.name])).toEqual([
    ["i-x", "renamed"],
    ["i-y", "y"],
  ]);
  expect(project.value, "no project list takes it").toBe(before);
  await vi.advanceTimersByTimeAsync(1_000);
  expect(
    tab.nerve.calls.map((call) => [call.method, call.path]),
    "no counts again"
  ).toEqual([["PATCH", INTAKE]]);
});

it("restores a description version of a work item held in an intake through its intake", async () => {
  held();
  const restore = () => tab.root.issueDetail.versions.restore(acme.slug, "p-web", "i-x", "<p>old</p>");
  const out = await requestedIssue(tab.nerve, restore, ["PATCH", INTAKE]);
  expect(out.call.body).toEqual({ description_html: "<p>old</p>" });
  out.call.answer(json(200, detailOf("x", { description_html: "<p>old</p>", updated_at: at(1) })));
  expect(await out.promise).toBe("<p>old</p>");
});

it("decides where a change goes in its turn: one queued behind an acceptance goes to /issues", async () => {
  held();
  const accepting = track(tab.root.writes.triageIntake("i-x", { status: "accepted" }, tab.scope));
  await until(() => tab.nerve.calls.length === 1, "the acceptance");
  const change = track(tab.root.writes.updateIssue("i-x", () => ({ name: "renamed" }), undefined, tab.scope));
  const answer = detailOf("x", { intake: accepted, updated_at: at(1) });
  await inTurn(tab.nerve, 0, ["POST", `${INTAKE}/triage`], json(200, answer));
  await inTurn(tab.nerve, 1, ["GET", ISSUE], json(200, answer));
  await inTurn(tab.nerve, 2, ["PATCH", ISSUE], json(200, { ...answer, name: "renamed", updated_at: at(2) }));
  await until(() => accepting.settled && change.settled, "both");
  expect([accepting.error, change.error]).toEqual([undefined, undefined]);
});

it.each<Kind>(["edit", "decline", "delete"])(
  "a %s of a work item in an intake reads no directory of the project lists held, fails none, puts no row in them",
  async (kind) => {
    held();
    const started = stateOf(tab.web, "started", "started", 1);
    tab.nerve.respond = (call) => {
      if (call.method === "GET" && call.path === "/api/v0/projects/p-web/states")
        call.answer(json(200, { data: [started] }));
      if (call.method === "GET" && call.path === ISSUE)
        call.answer(
          kind === "delete" ? problem(404, "issue.not_found") : json(200, detailOf("x", { intake: declined }))
        );
    };
    const other = { ...issueOf("i-y", "p-web", acme.id), state_id: started.id };
    const filtered = await projectList({ state_group: ["started"] }, issuePageOf([other]));
    const grouped = new IssueListBinding(tab.root, {
      scope: "member",
      workspaceSlug: acme.slug,
      workspaceID: acme.id,
      userID: "actor",
      query: { relation: "created", order_by: "sort_order", group_by: "state_group" },
    });
    const read = await requestedIssue(tab.nerve, () => grouped.fetchFirst(), [
      "GET",
      "/api/v0/workspaces/acme/members/actor/issues",
    ]);
    read.call.answer(json(200, { total: 1, groups: [{ key: "started", total: 1, data: [other], next_cursor: null }] }));
    await read.promise;
    const lists = [filtered, grouped];
    expect(lists.map(shownList)).toEqual([
      { total: 1, groups: [["", 1, ["i-y"]]] },
      { total: 1, groups: [["started", 1, ["i-y"]]] },
    ]);
    tab.nerve.calls.length = 0;
    const change = await sent(tab.nerve, send(kind), request(kind), done(kind));
    expect(change.error).toBeUndefined();
    expect(tab.nerve.to("/api/v0/projects/p-web/states")).toEqual([]);
    expect(lists.map(shownList)).toEqual([
      { total: 1, groups: [["", 1, ["i-y"]]] },
      { total: 1, groups: [["started", 1, ["i-y"]]] },
    ]);
    expect(lists.map((list) => list.error)).toEqual([undefined, undefined]);
  }
);

it("refuses an intake write whose role dropped while it waited behind another write of the work item", async () => {
  held();
  await intakeHeld();
  const first = track(send("edit")());
  await until(() => tab.nerve.calls.length === 1, "the first write");
  const queued = track(send("decline")());
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [projectOf("WEB", acme.id, { member_role: 20 })]);
  tab.nerve.calls[0]!.answer(done("edit"));
  await until(() => first.settled && queued.settled, "both writes");
  expect(queued.error).toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.to(`${INTAKE}/triage`)).toEqual([]);
});

it.each<Kind>(["edit", "decline", "delete"])(
  "a %s nerve refuses leaves every value as it was and is handed on, nothing read again",
  async (kind) => {
    held();
    await intakeHeld();
    const change = await sent(tab.nerve, send(kind), request(kind), problem(403, "forbidden"));
    expect(change.error).toMatchObject({ status: 403, problem: { code: "forbidden" } });
    expect(tab.root.issues.getIssueDetailById("i-x")).toEqual(detailOf("x"));
    expect(tab.store.getList("p-web", open)).toEqual(pageOf([x, y]));
    await vi.advanceTimersByTimeAsync(1_000);
    expect(tab.nerve.calls).toHaveLength(1);
  }
);

it.each<Kind>(["edit", "decline", "delete"])(
  "a %s nerve refuses as out of its intake takes it out of every intake list and counts again before it is handed on",
  async (kind) => {
    held();
    await intakeHeld();
    await loadList(tab.nerve, tab.store, "p-web", { ...open, statuses: ["pending", "snoozed"] }, pageOf([x]));
    tab.nerve.calls.length = 0;
    tab.nerve.respond = (call) => {
      if (call.method === "GET" && call.path === ISSUE) call.answer(problem(404, "issue.not_found"));
    };
    const change = await sent(tab.nerve, send(kind), request(kind), problem(404, "issue.intake_not_found"));
    expect(change.error).toMatchObject({ problem: { code: "issue.intake_not_found" } });
    expect(tab.store.getList("p-web", open)).toEqual(pageOf([y], null, 1));
    expect(tab.store.getList("p-web", { ...open, statuses: ["pending", "snoozed"] })).toEqual(pageOf([], null, 0));
    await until(() => tab.nerve.to(countsPath(acme)).length === 1, "the counts again");
    // a triage's page leaves the work item itself (the action model's conflict); a content change's or a deletion's
    // follows canonical, which the work item read again in the refused write's turn tells
    expect(tab.nerve.to(ISSUE)).toHaveLength(kind === "decline" ? 0 : 1);
  }
);

it.each([
  ["edit", json(200, detailOf("x", { intake: null, updated_at: at(1) })), null],
  ["delete", problem(404, "issue.not_found"), undefined],
] as const)(
  "a %s refused as out of its intake reads the work item again in its turn: canonical then holds nerve's read",
  async (kind, read, intake) => {
    held();
    await intakeHeld();
    const refused = track(send(kind)());
    await inTurn(tab.nerve, 0, request(kind), problem(404, "issue.intake_not_found"));
    await inTurn(tab.nerve, 1, ["GET", ISSUE], read);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toMatchObject({ problem: { code: "issue.intake_not_found" } });
    expect(tab.root.verification.isMissing("i-x")).toBe(intake === undefined);
    expect(tab.root.issues.getIssueDetailById("i-x")?.intake).toBe(intake);
    expect(tab.store.getList("p-web", open)).toEqual(pageOf([y], null, 1));
    await until(() => tab.nerve.to(countsPath(acme)).length === 1, "the counts again");
  }
);

it.each([
  ["decline", "issue.intake_transition_not_allowed"],
  ["edit", "issue.intake_closed"],
] as const)(
  "a %s refused as %s reads the work item again in its turn; a change queued behind it then goes to /issues",
  async (kind, code) => {
    held();
    await intakeHeld();
    const refused = track(send(kind)());
    const queued = track(tab.root.writes.updateIssue("i-x", () => ({ priority: "high" }), undefined, tab.scope));
    await inTurn(tab.nerve, 0, request(kind), problem(409, code));
    const theirs = detailOf("x", { intake: accepted, updated_at: at(1) });
    await inTurn(tab.nerve, 1, ["GET", ISSUE], json(200, theirs));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toMatchObject({ problem: { code } });
    expect(tab.store.getList("p-web", open)).toEqual(pageOf([y], null, 1));
    expect(tab.store.getList("p-web", closed), "a list that lacks it gets none").toEqual(pageOf([]));
    await until(() => tab.nerve.to(countsPath(acme)).length === 1, "the counts again");
    await reply(["PATCH", ISSUE], json(200, { ...theirs, priority: "high", updated_at: at(2) }));
    await until(() => queued.settled, "the change queued");
    expect(queued.error).toBeUndefined();
    expect(tab.nerve.to(INTAKE)).toHaveLength(kind === "edit" ? 1 : 0);
  }
);

it("takes a work item read again as gone after a conflict out of every intake list", async () => {
  held();
  await intakeHeld();
  const refused = track(send("decline")());
  await inTurn(tab.nerve, 0, request("decline"), problem(409, "issue.intake_transition_not_allowed"));
  await inTurn(tab.nerve, 1, ["GET", ISSUE], problem(404, "issue.not_found"));
  await until(() => refused.settled, "the refusal");
  expect(refused.error).toMatchObject({ problem: { code: "issue.intake_transition_not_allowed" } });
  expect(tab.store.getList("p-web", open)).toEqual(pageOf([y], null, 1));
});

it.each<IssueIntake>([
  pending,
  { status: "snoozed", snoozed_till: "2026-11-11T08:00:00Z", duplicate_to_id: null },
  declined,
  { status: "duplicate", snoozed_till: null, duplicate_to_id: "i-y" },
])(
  "takes a list's row landing over a detail held in an intake (%o) as accepted: its next change goes to /issues",
  async (intake) => {
    held({ intake });
    tab.root.acceptListIssues([{ ...issueOf("i-x", "p-web", acme.id), updated_at: at(1) }], tab.root.captureRead());
    expect(tab.root.issues.getIssueDetailById("i-x")).toMatchObject({ intake: accepted, description_html: "<p>x</p>" });
    const change = () => tab.root.writes.updateIssue("i-x", () => ({ name: "renamed" }), undefined, tab.scope);
    const out = await requestedIssue(tab.nerve, change, ["PATCH", ISSUE]);
    out.call.answer(json(200, detailOf("x", { intake: accepted, name: "renamed", updated_at: at(2) })));
    await out.promise;
  }
);

it.each([null, accepted])("keeps the intake row %o of a detail a list's row lands over", (intake) => {
  held({ intake });
  const row = { ...issueOf("i-x", "p-web", acme.id), name: "listed", updated_at: at(1) };
  tab.root.acceptListIssues([row], tab.root.captureRead());
  expect(tab.root.issues.getIssueDetailById("i-x")).toEqual({ ...detailOf("x", { intake }), ...row });
});

it("triages a work item only once a comment's write to it is done: one queue for each work item", async () => {
  held();
  const comment = track(tab.root.issueDetail.comment.createComment(acme.slug, "p-web", "i-x", { comment_html: "c" }));
  await until(() => tab.nerve.calls.length === 1, "the comment");
  const triage = track(tab.root.writes.triageIntake("i-x", { status: "declined" }, tab.scope));
  await vi.advanceTimersByTimeAsync(1_000);
  expect(tab.nerve.calls.map((call) => [call.method, call.path])).toEqual([["POST", `${ISSUE}/comments`]]);
  tab.nerve.respond = (call) => {
    const triaged = tab.nerve.to(`${INTAKE}/triage`).length > 0;
    if (call.method === "GET" && call.path === ISSUE)
      call.answer(json(200, detailOf("x", { intake: triaged ? declined : pending })));
    if (call.method === "GET" && call.path === `${ISSUE}/comments`)
      call.answer(json(200, { data: [issueCommentOf("c-1", "i-x")], next_cursor: null }));
    if (call.method === "POST" && call.path === `${INTAKE}/triage`)
      call.answer(json(200, detailOf("x", { intake: declined })));
  };
  tab.nerve.calls[0]!.answer(json(201, issueCommentOf("c-1", "i-x")));
  await until(() => comment.settled && triage.settled, "both");
  expect([comment.error, triage.error]).toEqual([undefined, undefined]);
  expect(tab.nerve.calls.map((call) => `${call.method} ${call.path}`)).toEqual([
    `POST ${ISSUE}/comments`,
    `GET ${ISSUE}`,
    `GET ${ISSUE}/comments`,
    `POST ${INTAKE}/triage`,
    `GET ${ISSUE}`,
    `GET ${ISSUE}/comments`,
  ]);
});

it("confirms nothing of a change whose permission scope changed while it was out: write()'s marks", async () => {
  held();
  await intakeHeld();
  const change = track(send("edit")());
  await until(() => tab.nerve.calls.length === 1, "the change");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, acme, [projectOf("WEB", acme.id, { member_role: 20 })]);
  tab.nerve.calls[0]!.answer(done("edit"));
  await until(() => change.settled, "the change");
  expect(change.error).toBeUndefined();
  expect(tab.root.issues.held("i-x")?.name).toBe("x");
  expect(tab.store.getList("p-web", open)).toEqual(pageOf([x, y]));
});

it.each<Kind>(["edit", "decline", "delete"])(
  "a %s whose answer comes after the session changed writes nothing",
  async (kind) => {
    held();
    await intakeHeld();
    const change = track(send(kind)());
    await until(() => tab.nerve.calls.length === 1, "the write");
    tab.root.retire();
    tab.nerve.calls[0]!.answer(done(kind));
    await until(() => change.settled, "the write");
    expect(change.error).toBeInstanceOf(SessionChangedError);
    expect(tab.root.issues.held("i-x")).toEqual(detailOf("x"));
    expect(tab.store.getList("p-web", open)).toEqual(pageOf([x, y]));
    expect(tab.nerve.calls).toHaveLength(1);
  }
);
