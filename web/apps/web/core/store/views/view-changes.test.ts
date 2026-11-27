/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, View, ViewCreate, ViewUpdate } from "@nerve/api-client";
import { ApiError, problemCode } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { listTab, ops, web } from "@/store/fake-project-lists";
import { inTurn, sent } from "@/store/fake-queue";
import { preferencesOf } from "@/store/project/fake-projects";
import { ViewStore } from "@/store/views/view.store";

// The changes of the views (P6a spec 2.1, R3; v0 design 7.7), against a fake nerve that answers each request when the
// test says. A view's changes go one at a time, each body reckoned in its turn, and a project's creations too; another
// view's do not wait. Only nerve's answer is written, into its project's views and the view's own read: nothing
// before nerve answers, nothing when it refuses, nothing once the session has changed. The tab's address is web's, a
// project of acme; ops is acme's other.

const { display_filters, display_properties } = preferencesOf();

/** A view of the project as nerve gives it: the name names it in its project; ann's, of no filter, unless fields say. */
const viewOf = (project: Project, name: string, fields: Partial<View> = {}): View => ({
  id: `v-${project.identifier}-${name}`,
  workspace_id: project.workspace_id,
  project_id: project.id,
  name,
  description: "",
  logo_props: {},
  filters: {},
  display_filters,
  display_properties,
  owned_by_id: "u-ann",
  created_at: "2026-11-01T09:00:00Z",
  updated_at: "2026-11-01T09:00:00Z",
  ...fields,
});

/** Times after the fixtures', of nerve's answers to the changes. */
const LATER = "2026-11-10T09:00:00Z";
const LATEST = "2026-11-10T10:00:00Z";
const bugs = viewOf(web, "Bugs", { filters: { priority: ["urgent", "high"] }, created_at: "2026-11-03T09:00:00Z" });
const mine = viewOf(web, "Mine", { filters: { assignee_id: ["u-ann"] }, created_at: "2026-11-02T09:00:00Z" });
const triage = viewOf(web, "Triage", { owned_by_id: "u-bob" });
/** web's views as nerve lists them: the newest first. */
const webViews = [bugs, mine, triage];
/** A view created in web, as nerve answers it: the newest. */
const release = viewOf(web, "Release", { created_at: LATER, updated_at: LATER });

/** The requests of the views: a project's, where it creates its views; a view's, changed and deleted there. */
const list = (project: Project) => `/api/v0/projects/${project.id}/views`;
const at = (view: View) => `/api/v0/views/${view.id}`;

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

/** A tab whose store holds web's views, and Bugs's own read. */
async function loadedViews() {
  const tab = await listTab(ViewStore);
  const { nerve, store } = tab;
  await answered(nerve, () => store.fetchProjectViews(web.id), ["GET", list(web)], { data: webViews }, "the views");
  await answered(nerve, () => store.fetchView(bugs.id), ["GET", at(bugs)], bugs, "the view");
  return tab;
}

/** Every value the store holds of web's views: its list, and each view's own read. */
const held = (store: ViewStore) => ({
  list: store.lists.get(web.id),
  details: [...webViews, release].map((view) => store.details.get(view.id)),
});

describe("ViewStore, the changes", () => {
  it("creates a view: first of its project's views, as nerve lists them, the newest first", async () => {
    const { nerve, store } = await loadedViews();
    const body: ViewCreate = { name: "Release", display_filters, display_properties };
    const creating = await sent(
      nerve,
      () => store.create({ kind: "project", projectId: web.id }, body),
      ["POST", list(web)],
      json(201, release)
    );
    expect(nerve.calls.at(-1)?.body).toEqual(body);
    expect(creating.value).toEqual(release);
    expect(store.lists.get(web.id)).toEqual([release, ...webViews]);
    expect(store.getViewById(release.id)).toEqual(release);

    // of a project whose views the store has not listed: nothing is written, its fetch brings them
    const infra = viewOf(ops, "Infra");
    await sent(
      nerve,
      () => store.create({ kind: "project", projectId: ops.id }, { name: "Infra" }),
      ["POST", list(ops)],
      json(201, infra)
    );
    expect(store.getProjectViews(ops.id)).toBeUndefined();
  });

  it("changes a view: nerve's answer replaces it in its project's views and in its read", async () => {
    const { nerve, store } = await loadedViews();
    const body: ViewUpdate = { name: "Urgent", filters: { priority: ["urgent"] } };
    const changed: View = { ...bugs, ...body, updated_at: LATER };
    const changing = await sent(
      nerve,
      () => store.update(bugs.id, () => body),
      ["PATCH", at(bugs)],
      json(200, changed)
    );
    expect(nerve.calls.at(-1)?.body).toEqual(body);
    expect(changing.value).toEqual(changed);
    expect(store.lists.get(web.id)).toEqual([changed, mine, triage]);
    expect(store.details.get(bugs.id)).toEqual(changed);
    expect(store.getViewById(bugs.id)).toEqual(changed);

    // a view the store has no read of: its row alone
    const described: View = { ...mine, description: "Mine to do", updated_at: LATER };
    await sent(
      nerve,
      () => store.update(mine.id, () => ({ description: "Mine to do" })),
      ["PATCH", at(mine)],
      json(200, described)
    );
    expect(store.lists.get(web.id)).toEqual([changed, described, triage]);
    expect(store.details.get(mine.id)).toBeUndefined();
  });

  it("writes nerve's answer to a change of nothing: the view as nerve holds it, which another tab changed", async () => {
    const { nerve, store } = await loadedViews();
    // another tab described it; this one sends its name as it is: nerve writes nothing, and answers the view it holds
    const elsewhere: View = { ...bugs, description: "Urgent and high", updated_at: LATER };
    const unchanged = { name: bugs.name };
    await sent(nerve, () => store.update(bugs.id, () => unchanged), ["PATCH", at(bugs)], json(200, elsewhere));
    expect(nerve.calls.at(-1)?.body).toEqual(unchanged);
    expect(store.lists.get(web.id)).toEqual([elsewhere, mine, triage]);
    expect(store.details.get(bugs.id)).toEqual(elsewhere);
  });

  it("deletes a view: out of its project's views, its read null; neither shows, nor once the fetches out meanwhile answer", async () => {
    const { nerve, store } = await loadedViews();
    const k = nerve.calls.length;
    const listing = track(store.fetchProjectViews(web.id));
    await until(() => nerve.calls.length === k + 1, "the list");
    const reading = track(store.fetchView(bugs.id));
    await until(() => nerve.calls.length === k + 2, "the read");
    const removing = await sent(nerve, () => store.remove(bugs.id), ["DELETE", at(bugs)], noContent());
    expect(removing).toEqual({ settled: true, value: undefined });
    const gone = () => {
      expect(store.lists.get(web.id)).toEqual([mine, triage]);
      expect(store.details.get(bugs.id)).toBeNull();
      expect(store.getViewById(bugs.id)).toBeUndefined();
      expect(store.getProjectViews(web.id)).toEqual([mine, triage]);
    };
    gone();
    // the list and the read, which nerve answered before the deletion
    nerve.calls[k]?.answer(json(200, { data: webViews }));
    nerve.calls[k + 1]?.answer(json(200, bugs));
    await until(() => listing.settled && reading.settled, "the fetches");
    gone();
    expect(reading.value).toBeNull();

    // a view the store has no read of: its row goes, and it has no read
    await sent(nerve, () => store.remove(mine.id), ["DELETE", at(mine)], noContent());
    expect(store.lists.get(web.id)).toEqual([triage]);
    expect(store.getViewById(mine.id)).toBeUndefined();
  });
});

describe("ViewStore, a change in its turn", () => {
  it("sends a view's changes one at a time, each body reckoned in its turn; another view's change does not wait", async () => {
    const { nerve, store } = await loadedViews();
    const k = nerve.calls.length;
    let name = "Urgent";
    const first = track(store.update(bugs.id, () => ({ name })));
    const second = track(store.update(bugs.id, () => ({ name })));
    await until(() => nerve.calls.length > k, "the first change");
    // what the second sends changes while the first is out: it sends what it reads in its turn
    name = "Critical";
    const other = track(store.update(mine.id, () => ({ name: "Me" })));
    await until(() => nerve.calls.length > k + 1, "the other view's change");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls.slice(k).map(({ method, path, body }) => [method, path, body])).toEqual([
      ["PATCH", at(bugs), { name: "Urgent" }],
      ["PATCH", at(mine), { name: "Me" }],
    ]);
    nerve.calls[k]?.answer(json(200, { ...bugs, name: "Urgent", updated_at: LATER }));
    await inTurn(nerve, k + 2, ["PATCH", at(bugs)], json(200, { ...bugs, name: "Critical", updated_at: LATEST }));
    expect(nerve.calls[k + 2]?.body).toEqual({ name: "Critical" });
    nerve.calls[k + 1]?.answer(json(200, { ...mine, name: "Me", updated_at: LATER }));
    await until(() => first.settled && second.settled && other.settled, "the three changes");
    expect(store.getViewById(bugs.id)?.name).toBe("Critical");
    expect(store.getProjectViews(web.id)?.map((view) => view.name)).toEqual(["Critical", "Me", "Triage"]);
  });

  it("creates a project's views one at a time: each first of its views, the newest first", async () => {
    const { nerve, store } = await loadedViews();
    const k = nerve.calls.length;
    const notes = viewOf(web, "Notes", { created_at: LATEST, updated_at: LATEST });
    const creating = [
      track(store.create({ kind: "project", projectId: web.id }, { name: "Release" })),
      track(store.create({ kind: "project", projectId: web.id }, { name: "Notes" })),
    ];
    await inTurn(nerve, k, ["POST", list(web)], json(201, release));
    await inTurn(nerve, k + 1, ["POST", list(web)], json(201, notes));
    await until(() => creating.every((change) => change.settled), "the two creations");
    expect(nerve.calls.slice(k).map((call) => call.body)).toEqual([{ name: "Release" }, { name: "Notes" }]);
    expect(store.lists.get(web.id)).toEqual([notes, release, ...webViews]);
  });

  it("deletes a view it finds in its turn: of one deleted before it, nerve is asked nothing", async () => {
    const { nerve, store } = await loadedViews();
    const k = nerve.calls.length;
    const removed = track(store.remove(bugs.id));
    const again = track(store.remove(bugs.id));
    await inTurn(nerve, k, ["DELETE", at(bugs)], noContent());
    await until(() => removed.settled && again.settled, "both deletions");
    expect(removed).toEqual({ settled: true, value: undefined });
    expect(again.error).toEqual(new Error("View not found"));
    expect(nerve.calls).toHaveLength(k + 1);
  });
});

/** A change of a view, as the page sends it: its request, nerve's answer to it, and nerve's refusal of it. */
type Change = {
  change: string;
  send: (store: ViewStore) => Promise<unknown>;
  request: Endpoint;
  answer: () => Response;
  /** nerve's refusal: its status and its problem's code. */
  refusal: [status: number, code: string];
};
const changes: Change[] = [
  {
    change: "a view created",
    send: (store) => store.create({ kind: "project", projectId: web.id }, { name: "Release" }),
    request: ["POST", list(web)],
    answer: () => json(201, release),
    refusal: [409, "navigation.views_disabled"],
  },
  {
    change: "a view changed",
    send: (store) => store.update(bugs.id, () => ({ name: "Urgent", filters: {} })),
    request: ["PATCH", at(bugs)],
    answer: () => json(200, { ...bugs, name: "Urgent", filters: {}, updated_at: LATER }),
    refusal: [409, "navigation.views_disabled"],
  },
  {
    change: "a view deleted",
    send: (store) => store.remove(bugs.id),
    request: ["DELETE", at(bugs)],
    answer: noContent,
    refusal: [404, "navigation.view_not_found"],
  },
];

describe("ViewStore, a change before nerve confirms it", () => {
  it.each(changes)("changes nothing while nerve has not answered $change", async ({ send, request, answer }) => {
    const { nerve, store } = await loadedViews();
    const before = held(store);
    const k = nerve.calls.length;
    const changing = track(send(store));
    await until(() => nerve.calls.length === k + 1, "the change");
    expect(nerve.calls[k]).toMatchObject({ method: request[0], path: request[1] });
    await vi.advanceTimersByTimeAsync(1_000);
    expect(held(store)).toEqual(before);
    nerve.calls[k]?.answer(answer());
    await until(() => changing.settled, "the answer");
    expect(changing.error).toBeUndefined();
    expect(held(store)).not.toEqual(before);
  });

  it.each(changes)("fails with nerve's refusal, changing nothing, when nerve refuses $change", async (change) => {
    const { nerve, store } = await loadedViews();
    const before = held(store);
    const k = nerve.calls.length;
    const refused = track(change.send(store));
    await inTurn(nerve, k, change.request, problem(...change.refusal));
    await until(() => refused.settled, "the refusal");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(problemCode(refused.error)).toBe(change.refusal[1]);
    expect(held(store)).toEqual(before);
    expect(nerve.calls).toHaveLength(k + 1);
  });

  it("changes nothing of this generation once the session has changed: each change fails before it is sent", async () => {
    const { nerve, api, store } = await loadedViews();
    const before = held(store);
    const k = nerve.calls.length;
    FakeNerve.replaceSession(api);
    const cut = changes.map(({ send }) => track(send(store)));
    await until(() => cut.every((change) => change.settled), "the changes");
    expect(cut.map((change) => change.error)).toEqual(changes.map(() => expect.any(SessionChangedError)));
    expect(nerve.calls).toHaveLength(k);
    expect(held(store)).toEqual(before);
  });
});
