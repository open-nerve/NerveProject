/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueDetail } from "@nerve/api-client";
import type { AssigneeSetChange, IssueChange } from "@nerve/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { inTurn } from "@/store/fake-queue";
import { issueDetailOf, issueStoreTab, requestedIssue } from "./fake-issues";
import { acme, detailOf, holdDetail, writesTab } from "./intake/fake-intake";
import { changeBody } from "./issue-write-fields";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

// P5b spec 2.6, R20 (Q34): a change of a work item's assignees from a dropdown is additions and removals, made into
// the whole set nerve replaces from the set the work item holds when its turn comes (v0 design 7.7), as its modules
// are (P7a spec 2.2); the request is still the whole set. So a second pick while the first one's change is out keeps
// the first, on a project's work item and on one of an intake alike, and after a refusal the set is as it was. A whole
// set, worked out in the turn (a drop, the palette) or chosen in a form, still goes as given.

/** The body as it goes on the wire: the fields given, none of those left undefined. */
const wire = (body: unknown) => JSON.parse(JSON.stringify(body)) as unknown;
/** The work item held with the assignees of ids, and no module. */
const holding = (assignee_ids: string[]) => ({ assignee_ids, module_ids: [] });
/** The n-th version of a work item, as nerve stamps it. */
const version = (n: number) => `2026-11-24T00:00:00.00000000${n}Z`;
/** The set a change of the assignees makes of from, as its request names it. */
const after = (from: string[], assignees: AssigneeSetChange) => changeBody({ assignees }, holding(from)).assignee_ids;

it("a change of the assignees removes and adds to the set held, keeping its order and the ids it does not name", () => {
  const from = ["A", "Z"];
  expect(after(from, { add: ["B"] })).toEqual(["A", "Z", "B"]);
  expect(after(from, { remove: ["A"] })).toEqual(["Z"]);
  expect(after(from, { add: ["A"] }), "adding one it holds").toEqual(["A", "Z"]);
  expect(after(from, { remove: ["B"] }), "removing one it does not hold").toEqual(["A", "Z"]);
  expect(after(["G", "A", "C"], { add: ["B", "D"], remove: ["A"] })).toEqual(["G", "C", "B", "D"]);
  expect(after([], { add: ["B"] })).toEqual(["B"]);
  expect(from, "the set held is not changed").toEqual(["A", "Z"]);
});

it("a change's request names the assignees' whole set as given, or the set its additions and removals make", () => {
  expect(wire(changeBody({ assignee_ids: ["B"] }, holding(["A"])))).toStrictEqual({ assignee_ids: ["B"] });
  expect(wire(changeBody({ assignee_ids: [] }, holding(["A"])))).toStrictEqual({ assignee_ids: [] });
  expect(wire(changeBody({ name: "Renamed", assignees: { add: ["B"] } }, holding(["A"])))).toStrictEqual({
    name: "Renamed",
    assignee_ids: ["A", "B"],
  });
  const both = { assignee_ids: ["Z"], assignees: { add: ["B"] } } as unknown as IssueChange;
  expect(wire(changeBody(both, holding(["A"]))), "a whole set slipped beside them").toStrictEqual({
    assignee_ids: ["A", "B"],
  });
  const planned = changeBody({ assignees: { remove: ["A"] }, modules: { add: ["M"] } }, holding(["A"]));
  expect(wire(planned)).toStrictEqual({ assignee_ids: [], module_ids: ["M"] });
  expect(wire(changeBody({ name: "Renamed" }, holding(["A"]))), "none named, none sent").toStrictEqual({
    name: "Renamed",
  });
});

it("a change cannot carry the assignees both as a whole set and as additions and removals", () => {
  // @ts-expect-error the whole set and its additions and removals are one or the other
  const both: IssueChange = { assignee_ids: ["A"], assignees: { add: ["B"] } };
  const whole: IssueChange = { assignee_ids: ["A"] };
  const change: IssueChange = { name: "Renamed", assignees: { add: ["B"], remove: ["A"] } };
  expect([both, whole, change]).toHaveLength(3);
});

describe("a work item of a project", () => {
  let tab: Awaited<ReturnType<typeof issueStoreTab>>;
  beforeEach(async () => {
    vi.useFakeTimers();
    tab = await issueStoreTab();
  });
  afterEach(() => {
    tab.app.dispose();
    vi.useRealTimers();
  });
  const senders: [string, (change: IssueChange) => Promise<unknown>][] = [
    [
      "the write unit's command",
      (change) => tab.app.issue.writes.changeIssue("x", change, tab.app.issue.scope("ws", "A")),
    ],
    ["a list's update", (change) => tab.app.issue.projectIssues.updateIssue("ws", "A", "x", change)],
    ["the detail's update", (change) => tab.app.issue.issueDetail.updateIssue("ws", "A", "x", change)],
  ];
  const answerOf = (assignee_ids: string[], n: number): IssueDetail => ({
    ...issueDetailOf("x"),
    assignee_ids,
    updated_at: version(n),
  });
  const PATCH = ["PATCH", "/api/v0/issues/x"] as const;
  /** The work item x held with the assignees A and Z, Z a member the page does not know. */
  const held = () => tab.app.issue.issues.acceptRead(answerOf(["A", "Z"], 1), tab.app.issue.captureRead());

  it.each(senders)(
    "%s adds an assignee in its turn after one still out, to the set that one's answer holds",
    async (_, send) => {
      const { nerve, app } = tab;
      held();
      const first = await requestedIssue(nerve, () => send({ assignees: { add: ["B"] } }), [...PATCH]);
      expect(first.call.body).toEqual({ assignee_ids: ["A", "Z", "B"] });
      const second = track(send({ assignees: { add: ["O"] } }));
      await vi.advanceTimersByTimeAsync(1_000);
      expect(nerve.calls, "the second change waits for the first").toHaveLength(1);
      first.call.answer(json(200, answerOf(["A", "Z", "B"], 2)));
      await first.promise;
      await until(() => nerve.calls.length === 2, "the second change");
      expect(nerve.calls[1]?.body, "worked out from the first one's answer").toEqual({
        assignee_ids: ["A", "Z", "B", "O"],
      });
      nerve.calls[1]?.answer(json(200, answerOf(["A", "Z", "B", "O"], 3)));
      await until(() => second.settled, "the second change");
      expect(second.error).toBeUndefined();
      expect(app.issue.issues.getIssueById("x")?.assignee_ids).toEqual(["A", "Z", "B", "O"]);
    }
  );

  it.each(senders)(
    "%s works a change out from the set held as it was when nerve refused the one before",
    async (_, send) => {
      const { nerve, app } = tab;
      held();
      const first = await requestedIssue(nerve, () => send({ assignees: { add: ["B"] } }), [...PATCH]);
      const second = track(send({ assignees: { add: ["O"], remove: ["A"] } }));
      const refused = expect(first.promise).rejects.toMatchObject({ problem: { code: "validation_failed" } });
      first.call.answer(problem(422, "validation_failed"));
      await refused;
      await until(() => nerve.calls.length === 2, "the second change");
      expect(nerve.calls[1]?.body, "the refused addition is not in it").toEqual({ assignee_ids: ["Z", "O"] });
      nerve.calls[1]?.answer(json(200, answerOf(["Z", "O"], 2)));
      await until(() => second.settled, "the second change");
      expect(second.error).toBeUndefined();
      expect(app.issue.issues.getIssueById("x")?.assignee_ids).toEqual(["Z", "O"]);
    }
  );
});

describe("a work item of a project's intake", () => {
  let tab: Awaited<ReturnType<typeof writesTab>>;
  const INTAKE = ["PATCH", "/api/v0/intake-issues/i-x"] as const;
  beforeEach(async () => {
    vi.useFakeTimers({ now: new Date("2026-11-24T09:00:00Z") });
    tab = await writesTab();
  });
  afterEach(() => {
    tab.app.dispose();
    vi.useRealTimers();
  });

  it("adds an assignee in its turn after one still out, through the intake's change, to the set that one's answer holds", async () => {
    holdDetail(tab, { assignee_ids: [] });
    const update = (change: IssueChange) => tab.root.issueDetail.updateIssue(acme.slug, "p-web", "i-x", change);
    const first = track(update({ assignees: { add: ["u-bob"] } }));
    await until(() => tab.nerve.calls.length === 1, "the first change");
    expect(tab.nerve.calls[0]?.body).toEqual({ assignee_ids: ["u-bob"] });
    const second = track(update({ assignees: { add: ["u-odo"] } }));
    const bob = detailOf("x", { assignee_ids: ["u-bob"], updated_at: "2026-11-24T09:00:01Z" });
    await inTurn(tab.nerve, 0, [...INTAKE], json(200, bob));
    const both = detailOf("x", { assignee_ids: ["u-bob", "u-odo"], updated_at: "2026-11-24T09:00:02Z" });
    await inTurn(tab.nerve, 1, [...INTAKE], json(200, both));
    expect(tab.nerve.calls[1]?.body, "worked out from the first one's answer").toEqual({
      assignee_ids: ["u-bob", "u-odo"],
    });
    await until(() => first.settled && second.settled, "both changes");
    expect([first.error, second.error]).toEqual([undefined, undefined]);
    expect(tab.root.issues.getIssueDetailById("i-x")?.assignee_ids).toEqual(["u-bob", "u-odo"]);
  });
});
