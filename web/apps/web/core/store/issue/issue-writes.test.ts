/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueDetail } from "@nerve/api-client";
import { issueDetailOf, issueOf, issueStoreTab, requestedIssue } from "./fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { loadModules, moduleRowOf } from "@/store/planning/fake-modules";
import { IssueListBinding } from "./helpers/issue-list-binding";
import type { IssueChange } from "@nerve/types";
import { loadProjects, moduleOf, projectOf } from "@/store/project/fake-projects";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const version = (n: number) => `2026-10-28T00:00:00.00000000${n}Z`;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

// The write unit takes the scope its caller gives (P7a spec 2.1): it never works one out from the held copy.
it("confirms a write into the lists covering the scope it is given, whatever project the held copy names", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  const given = new IssueListBinding(tab.app.issue, {
    scope: "project",
    workspaceSlug: "ws",
    workspaceID: tab.workspace.id,
    projectID: "B",
    query: { group_by: "state_id", order_by: "sort_order", limit: 10 },
  });
  const listed = { ...issueOf("x", "B"), state_id: "A", sort_order: 5, updated_at: version(1) };
  const read = await requestedIssue(tab.nerve, () => given.fetchFirst(), ["GET", "/api/v0/projects/B/issues"]);
  read.call.answer(
    json(200, {
      total: 1,
      groups: [
        { key: "A", total: 1, data: [listed], next_cursor: null },
        { key: "B", total: 0, data: [], next_cursor: null },
      ],
    })
  );
  await read.promise;
  tab.app.issue.acceptListIssues([{ ...listed, project_id: "A", updated_at: version(2) }], tab.app.issue.captureRead());
  expect(tab.app.issue.issues.getIssueById("x")?.project_id).toBe("A");
  const writing = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.writes.updateIssue("x", () => ({ state_id: "B" }), undefined, tab.app.issue.scope("ws", "B")),
    ["PATCH", "/api/v0/issues/x"]
  );
  writing.call.answer(json(200, { ...issueDetailOf("x", "B"), state_id: "B", sort_order: 5, updated_at: version(3) }));
  await writing.promise;
  await vi.advanceTimersByTimeAsync(1);
  expect(given.group("A")?.rows.map((row) => row.id)).toEqual([]);
  expect(given.group("B")?.rows.map((row) => row.id)).toEqual(["x"]);
});

// M6 P7a spec 2.2, R3, R19: a change of the work item names its modules as additions and removals, applied in the
// work item's turn to the set it then holds; its other fields, the cycle too, go as given. Each way the lists and the
// detail send one takes the write unit's command.
const MA = "m-alpha";
const MB = "m-beta";
/** An archived module the work item is in, which the page's caller does not know. */
const MG = "m-gamma";
type Send = (change: IssueChange) => Promise<unknown>;
const senders: [string, Send][] = [
  [
    "the write unit's command",
    (change) => tab.app.issue.writes.changeIssue("x", change, tab.app.issue.scope("ws", "A")),
  ],
  ["a list's update", (change) => tab.app.issue.projectIssues.updateIssue("ws", "A", "x", change)],
  ["the detail's update", (change) => tab.app.issue.issueDetail.updateIssue("ws", "A", "x", change)],
];
const answerOf = (fields: Partial<IssueDetail>, n: number): IssueDetail => ({
  ...issueDetailOf("x"),
  ...fields,
  updated_at: version(n),
});
/** The work item held with these fields; the project's open modules, alpha and beta, known to the page. */
async function held(fields: Partial<IssueDetail>) {
  const { nerve, app, project } = tab;
  await loadModules(
    nerve,
    app.planning.modules,
    [moduleRowOf(moduleOf(project, "alpha", { id: MA })), moduleRowOf(moduleOf(project, "beta", { id: MB }))],
    project
  );
  app.issue.issues.acceptRead(answerOf(fields, 1), app.issue.captureRead());
  nerve.calls.length = 0;
}
const PATCH = ["PATCH", "/api/v0/issues/x"] as const;

it.each(senders)("%s adds and removes modules in the work item's turn, from the set it then holds", async (_, send) => {
  const { nerve, app } = tab;
  await held({ module_ids: [MA, MG] });
  const first = await requestedIssue(nerve, () => send({ modules: { add: [MB] } }), [...PATCH]);
  expect(first.call.body, "the archived module the page does not know stays").toEqual({ module_ids: [MA, MG, MB] });
  const second = track(send({ modules: { remove: [MA] } }));
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls, "the second change waits for the first").toHaveLength(1);
  first.call.answer(json(200, answerOf({ module_ids: [MA, MG, MB] }, 2)));
  await first.promise;
  await until(() => nerve.calls.length === 2, "the second change");
  expect(nerve.calls[1]?.body, "worked out from the first one's answer").toEqual({ module_ids: [MG, MB] });
  nerve.calls[1]?.answer(json(200, answerOf({ module_ids: [MG, MB] }, 3)));
  await until(() => second.settled, "the second change");
  expect(second.error).toBeUndefined();
  expect(app.issue.issues.getIssueById("x")?.module_ids).toEqual([MG, MB]);
});

it.each(senders)(
  "%s works a change out from the set held as it was when nerve refused the one before",
  async (_, send) => {
    const { nerve, app } = tab;
    await held({ module_ids: [MA, MG] });
    const first = await requestedIssue(nerve, () => send({ modules: { add: [MB] } }), [...PATCH]);
    expect(first.call.body).toEqual({ module_ids: [MA, MG, MB] });
    const second = track(send({ modules: { remove: [MA] } }));
    const refused = expect(first.promise).rejects.toMatchObject({ problem: { code: "planning.archived" } });
    first.call.answer(problem(409, "planning.archived"));
    await refused;
    await until(() => nerve.calls.length === 2, "the second change");
    expect(nerve.calls[1]?.body, "the refused addition is not in it").toEqual({ module_ids: [MG] });
    nerve.calls[1]?.answer(json(200, answerOf({ module_ids: [MG] }, 2)));
    await until(() => second.settled, "the second change");
    expect(second.error).toBeUndefined();
    expect(app.issue.issues.getIssueById("x")?.module_ids).toEqual([MG]);
  }
);

it.each(senders)(
  "%s sends a change's other fields as given beside the modules, and nothing it does not name",
  async (_, send) => {
    const { nerve } = tab;
    await held({ module_ids: [MA, MG] });
    const named = await requestedIssue(nerve, () => send({ name: "Renamed", modules: { add: [MB] } }), [...PATCH]);
    expect(named.call.body).toEqual({ name: "Renamed", module_ids: [MA, MG, MB] });
    named.call.answer(json(200, answerOf({ name: "Renamed", module_ids: [MA, MG, MB] }, 2)));
    await named.promise;
    const slipped = { id: "x", project_id: "A", name: "Again", module_ids: [MB] } as unknown as IssueChange;
    const other = await requestedIssue(nerve, () => send(slipped), [...PATCH]);
    expect(other.call.body, "a whole set slipped in is not sent").toEqual({ name: "Again" });
    other.call.answer(json(200, answerOf({ name: "Again", module_ids: [MA, MG, MB] }, 3)));
    await other.promise;
  }
);

it.each(senders)(
  "%s sends each cycle as given, null to leave, whatever cycle the work item then holds",
  async (_, send) => {
    const { nerve, app } = tab;
    await held({ cycle_id: "C0", module_ids: [MA, MG] });
    const first = await requestedIssue(nerve, () => send({ cycle_id: "C1" }), [...PATCH]);
    expect(first.call.body).toEqual({ cycle_id: "C1" });
    const second = track(send({ cycle_id: null }));
    first.call.answer(json(200, answerOf({ cycle_id: "C1", module_ids: [MA, MG] }, 2)));
    await first.promise;
    await until(() => nerve.calls.length === 2, "the second change");
    expect(nerve.calls[1]?.body).toEqual({ cycle_id: null });
    nerve.calls[1]?.answer(json(200, answerOf({ cycle_id: null, module_ids: [MA, MG] }, 3)));
    await until(() => second.settled, "the second change");
    const again = await requestedIssue(nerve, () => send({ cycle_id: null }), [...PATCH]);
    expect(again.call.body, "the cycle held is no reason to leave it out").toEqual({ cycle_id: null });
    again.call.answer(json(200, answerOf({ cycle_id: null, module_ids: [MA, MG] }, 4)));
    await again.promise;
    expect(app.issue.issues.getIssueById("x")).toMatchObject({ cycle_id: null, module_ids: [MA, MG] });
  }
);
