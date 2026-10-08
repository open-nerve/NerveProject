/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { inTurn, sent } from "@/store/fake-queue";
import { loadArchivedProjects, loadProject, preferencesOf, projectOf, projectTab } from "@/store/project/fake-projects";
import type { IProjectStore } from "@/store/project/project.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The changes of the projects the caller sees (M3 design 3.19, 7.3), against a fake nerve that answers each request
// when the test says; the lists and the reads are project.store.test.ts. The address names acme, where he is an
// admin; he is a member of web and docs, not of the public ops; beta is his other workspace.

const acme = workspaceOf("acme", { role: 20 });
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id, { sort_order: 2000 });
const ops = projectOf("OPS", acme.id, { member_role: null, sort_order: null });
const docs = projectOf("DOCS", acme.id, { sort_order: 1000 });
const old = projectOf("OLD", acme.id, { archived_at: "2026-10-02T09:00:00Z" });
/** web as nerve answers its change of name */
const renamed = { ...web, name: "Web App", updated_at: "2026-10-07T09:00:00Z" };
/** web as nerve answers its unarchiving */
const restored = { ...web, updated_at: "2026-10-08T10:00:00Z" };
const LIST = "/api/v0/workspaces/acme/projects";
const ids = (projects: Project[]) => projects.map((project) => project.id);
const placeOf = (project: Project) => `/api/v0/me/projects/${project.id}/preferences`;

/** The project store of a tab at acme's address, whose caller's workspaces are acme and beta; nerve listed acme's. */
async function loaded() {
  const tab = await projectTab({ workspace: acme, projects: [web, ops, docs] }, { workspace: beta });
  return { ...tab, store: tab.projectRoot.project };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProjectStore, the changes", () => {
  it("puts a created project first in its workspace's list, and in none not fetched", async () => {
    const { nerve, store } = await loaded();
    const created = projectOf("API", acme.id, { member_role: 20 });
    const body = { name: "API", identifier: "API" };
    const creating = await sent(nerve, () => store.createProject("acme", body), ["POST", LIST], json(201, created));
    expect(nerve.calls[0]?.body).toEqual(body);
    expect(creating.value).toEqual(created);
    // first, as nerve lists it: it is first in his sidebar
    expect(store.workspaceProjectIds).toEqual(ids([created, web, ops, docs]));

    // beta's projects were never fetched: a list of the new project alone would show as the whole list
    const elsewhere = projectOf("NEW", beta.id);
    const request: Endpoint = ["POST", "/api/v0/workspaces/beta/projects"];
    await sent(
      nerve,
      () => store.createProject("beta", { name: "New", identifier: "NEW" }),
      request,
      json(201, elsewhere)
    );
    expect(store.getProjectById(elsewhere.id)).toBeUndefined();
  });

  it("shows nerve's answer to a change in the list and in the project's own read", async () => {
    const { nerve, store } = await loaded();
    await loadProject(nerve, store, web);
    const request: Endpoint = ["PATCH", `/api/v0/projects/${web.id}`];
    const updated = await sent(
      nerve,
      () => store.updateProject(web.id, { name: "Web App" }),
      request,
      json(200, renamed)
    );
    expect(nerve.calls[1]?.body).toEqual({ name: "Web App" });
    expect(updated.value).toEqual(renamed);
    expect(store.getProjectById(web.id)).toEqual(renamed);
    expect(store.getProjectByIdentifier("WEB")).toEqual(renamed);
  });

  it("moves a project to the archived list and back as nerve archives and unarchives it, its own read too", async () => {
    const { nerve, store } = await loaded();
    await loadArchivedProjects(nerve, store, acme, [old]);
    await loadProject(nerve, store, web);
    const archived: Project = { ...web, archived_at: "2026-10-08T09:00:00Z" };
    await sent(
      nerve,
      () => store.archiveProject(web.id),
      ["POST", `/api/v0/projects/${web.id}/archive`],
      json(200, archived)
    );
    expect(store.totalProjectIds).toEqual(ids([ops, docs, old, archived]));
    expect(store.joinedProjectIds).toEqual([docs.id]);
    expect(store.getProjectById(web.id)).toEqual(archived);

    const request: Endpoint = ["POST", `/api/v0/projects/${web.id}/unarchive`];
    await sent(nerve, () => store.restoreProject(web.id), request, json(200, restored));
    expect(store.totalProjectIds).toEqual(ids([ops, docs, web, old]));
    expect(store.getProjectById(web.id)).toEqual(restored);
  });

  it("forgets a deleted project: neither the lists nor its own read give it", async () => {
    const { nerve, store } = await loaded();
    await loadArchivedProjects(nerve, store, acme, [old]);
    const shelved = await sent(
      nerve,
      () => store.deleteProject(old),
      ["DELETE", `/api/v0/projects/${old.id}`],
      noContent()
    );
    expect(shelved.error).toBeUndefined();
    expect(store.totalProjectIds).toEqual(ids([web, ops, docs]));

    await loadProject(nerve, store, web);
    const deleted = await sent(
      nerve,
      () => store.deleteProject(web),
      ["DELETE", `/api/v0/projects/${web.id}`],
      noContent()
    );
    expect(deleted.error).toBeUndefined();
    expect(store.getProjectById(web.id)).toBeUndefined();
    expect(store.workspaceProjectIds).toEqual(ids([ops, docs]));
    expect(store.joinedProjectIds).toEqual([docs.id]);
  });

  it("moves a project in the caller's sidebar before the one he dropped it on, to the place nerve gives it", async () => {
    const { nerve, store } = await loaded();
    // dropped on web: halfway between docs, before it, and web; nerve's answer places docs after web
    const moving = () => store.updateProjectSortOrder(docs, web.id, false);
    const moved = await sent(nerve, moving, ["PATCH", placeOf(docs)], json(200, preferencesOf({ sort_order: 2500 })));
    expect(nerve.calls[0]?.body).toEqual({ sort_order: 1500 });
    expect(moved.error).toBeUndefined();
    expect(store.joinedProjectIds).toEqual(ids([web, docs]));
    expect(store.getProjectById(docs.id)?.sort_order).toBe(2500);
  });

  it("reckons a move's place in its turn, from the places nerve gave: two moves in a row", async () => {
    const { nerve, store } = await loaded();
    // web before docs, his first: a step before it; then docs last, which is after web once nerve placed web first
    const first = track(store.updateProjectSortOrder(web, docs.id, false));
    const second = track(store.updateProjectSortOrder(docs, undefined, true));
    await inTurn(nerve, 0, ["PATCH", placeOf(web)], json(200, preferencesOf({ sort_order: 500 })));
    await inTurn(nerve, 1, ["PATCH", placeOf(docs)], json(200, preferencesOf({ sort_order: 11000 })));
    await until(() => second.settled, "the second move");
    expect(first.error).toBeUndefined();
    expect(nerve.calls.map((call) => call.body)).toEqual([{ sort_order: -9000 }, { sort_order: 11000 }]);
    expect(store.joinedProjectIds).toEqual(ids([web, docs]));
  });

  const changes: { change: string; send: (store: IProjectStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a creation",
      send: (store) => store.createProject("acme", { name: "Web", identifier: "WEB" }),
      refusal: problem(409, "project.identifier_taken"),
    },
    {
      change: "a change",
      send: (store) => store.updateProject(web.id, { name: "Ops" }),
      refusal: problem(409, "project.name_taken"),
    },
    { change: "a deletion", send: (store) => store.deleteProject(web), refusal: problem(403, "forbidden") },
    { change: "an archiving", send: (store) => store.archiveProject(web.id), refusal: problem(403, "forbidden") },
    { change: "an unarchiving", send: (store) => store.restoreProject(web.id), refusal: problem(403, "forbidden") },
    {
      change: "a move in the sidebar",
      send: (store) => store.updateProjectSortOrder(web, docs.id, false),
      refusal: problem(403, "forbidden"),
    },
  ];
  it.each(changes)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const refused = track(send(store));
    await until(() => nerve.calls.length === 1, "the change");
    nerve.calls[0]?.answer(refusal);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
    expect([web, ops, docs].map((project) => store.getProjectById(project.id))).toEqual([web, ops, docs]);
  });

  it("sends each change once nerve has answered the one before it, refused or not: two moves in a row among them", async () => {
    const { nerve, store } = await loaded();
    const sending = changes.map(({ send }) => track(send(store)));
    // the first move is refused; the second is sent after it, as each change after the one before
    await inTurn(nerve, 0, ["POST", LIST], json(201, projectOf("API", acme.id)));
    await inTurn(nerve, 1, ["PATCH", `/api/v0/projects/${web.id}`], json(200, renamed));
    await inTurn(nerve, 2, ["DELETE", `/api/v0/projects/${web.id}`], problem(403, "forbidden"));
    await inTurn(nerve, 3, ["POST", `/api/v0/projects/${web.id}/archive`], problem(403, "forbidden"));
    await inTurn(nerve, 4, ["POST", `/api/v0/projects/${web.id}/unarchive`], json(200, renamed));
    await inTurn(nerve, 5, ["PATCH", placeOf(web)], json(200, preferencesOf({ sort_order: 500 })));
    await until(() => sending.every((change) => change.settled), "the last change");
    expect(sending.map((change) => change.error === undefined)).toEqual([true, true, false, false, true, true]);
    expect(store.getProjectById(web.id)).toEqual({ ...renamed, sort_order: 500 });
  });
});
