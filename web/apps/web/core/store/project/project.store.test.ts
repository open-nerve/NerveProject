/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, sent } from "@/store/fake-queue";
import { loadArchivedProjects, loadProject, loadProjects, projectOf, projectTab } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The projects the caller sees (M3 design 3.19, 7.3), their lists and reads, against a fake nerve that answers each
// request when the test says; the changes are project.store.changes.test.ts. The address names acme, where he is an
// admin; he is a member of web and docs, not of the public ops; beta is his other workspace.

const acme = workspaceOf("acme", { role: 20 });
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id, { sort_order: 2000 });
const ops = projectOf("OPS", acme.id, { member_role: null, sort_order: null });
const docs = projectOf("DOCS", acme.id, { sort_order: 1000 });
const old = projectOf("OLD", acme.id, { archived_at: "2026-10-02T09:00:00Z" });
const lab = projectOf("LAB", beta.id);
/** web as nerve answers its change of name */
const renamed = { ...web, name: "Web App", updated_at: "2026-10-07T09:00:00Z" };
const LIST = "/api/v0/workspaces/acme/projects";
const ids = (projects: Project[]) => projects.map((project) => project.id);

/** The project store of a tab at acme's address, whose caller's workspaces are acme and beta; projects of neither fetched. */
async function unloaded() {
  const tab = await projectTab({ workspace: acme }, { workspace: beta });
  return { ...tab, store: tab.projectRoot.project, filters: tab.projectRoot.projectFilter };
}

/** The same, where nerve listed acme's web, ops and docs. */
async function loaded() {
  const tab = await projectTab({ workspace: acme, projects: [web, ops, docs] }, { workspace: beta });
  return { ...tab, store: tab.projectRoot.project, filters: tab.projectRoot.projectFilter };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProjectStore, the lists and the reads", () => {
  it("lists the address's projects as nerve gives them, and finds them by id and identifier", async () => {
    const { nerve, router, store } = await unloaded();
    expect(store.workspaceProjectIds).toBeUndefined();
    expect(store.loader).toBe("init-loader");

    const fetched = await loadProjects(nerve, store, acme, [web, ops, docs]);
    expect(nerve.calls[0]?.query).toEqual({ archived: "false" });
    expect(fetched.value).toEqual([web, ops, docs]);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
    expect(store.loader).toBe("loaded");
    // his own, by their place in his sidebar
    expect(store.joinedProjectIds).toEqual(ids([docs, web]));
    expect(store.getProjectById(ops.id)).toEqual(ops);
    expect(store.getProjectIdentifierById(web.id)).toBe("WEB");
    expect(store.getProjectByIdentifier("DOCS")).toEqual(docs);
    router.setQuery({ workspaceSlug: "acme", projectId: web.id });
    expect(store.currentProjectDetails).toEqual(web);

    // another of his workspaces: its projects are found by id, not by the address's identifiers
    await loadProjects(nerve, store, beta, [lab]);
    expect(store.getProjectById(lab.id)).toEqual(lab);
    expect(store.getProjectByIdentifier("LAB")).toBeUndefined();
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
  });

  it("lists the archived projects apart, and gives the projects page's once both lists are there", async () => {
    const { nerve, filters, store } = await loaded();
    expect(store.filteredProjectIds).toBeUndefined();
    const archived = await loadArchivedProjects(nerve, store, acme, [old]);
    expect(nerve.calls[0]?.query).toEqual({ archived: "true" });
    expect(archived.value).toEqual([old]);
    expect(store.totalProjectIds).toEqual(ids([web, ops, docs, old]));
    // the page shows the projects that are not archived, unless it asks for the archived ones
    expect(store.filteredProjectIds).toEqual(ids([web, ops, docs]));
    filters.updateDisplayFilters("acme", { archived_projects: true });
    expect(store.filteredProjectIds).toEqual([old.id]);
    filters.updateDisplayFilters("acme", { archived_projects: false });
    filters.updateSearchQuery("do");
    expect(store.filteredProjectIds).toEqual([docs.id]);
  });

  it("gives a project's own read nerve answered after its list, and holds nothing when nerve refuses it", async () => {
    const { nerve, store } = await loaded();
    const read = await loadProject(nerve, store, web, renamed);
    expect(read.value).toEqual(renamed);
    expect(store.getProjectById(web.id)).toEqual(renamed);

    const missing = track(store.fetchProject("p-gone"));
    await until(() => nerve.calls.length === 2, "the read");
    nerve.calls[1]?.answer(problem(404, "project.not_found"));
    await until(() => missing.settled, "the refusal");
    expect(missing.error).toBeInstanceOf(ApiError);
    expect(store.getProjectById("p-gone")).toBeUndefined();
  });

  it("gives the copy of a list nerve answered after the project's own read: a change made meanwhile shows (F-3)", async () => {
    const { nerve, store } = await loaded();
    await loadProject(nerve, store, web);
    const demoted: Project = { ...web, member_role: 5 };
    await loadProjects(nerve, store, acme, [demoted, ops, docs]);
    expect(store.getProjectById(web.id)).toEqual(demoted);
    // and the project's own read, answered after that list, again
    await loadProject(nerve, store, web, renamed);
    expect(store.getProjectById(web.id)).toEqual(renamed);
  });

  it("gives a project's own read nerve answered after its list, fetched twice before: one count orders every copy's answers", async () => {
    const { nerve, store } = await loaded();
    await loadProjects(nerve, store, acme, [web, ops, docs]);
    await loadProject(nerve, store, web, renamed);
    expect(store.getProjectById(web.id)).toEqual(renamed);
  });

  it("gives the copy of the archived list nerve answered after the project's own read: archived meanwhile, it shows so", async () => {
    const { nerve, store } = await loaded();
    await loadProject(nerve, store, web);
    const archivedWeb: Project = { ...web, archived_at: "2026-10-08T09:00:00Z" };
    await loadArchivedProjects(nerve, store, acme, [archivedWeb]);
    expect(store.getProjectById(web.id)).toEqual(archivedWeb);
  });

  it("fails when nerve refuses to read a project again, keeping the read it had", async () => {
    const { nerve, store } = await loaded();
    await loadProject(nerve, store, web, renamed);
    const again = track(store.fetchProject(web.id));
    await until(() => nerve.calls.length === 2, "the second read");
    nerve.calls[1]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    // its own read, not the list's web
    expect(store.getProjectById(web.id)).toEqual(renamed);
  });

  it("fails when nerve refuses a list, keeping none, and again, keeping the list it had", async () => {
    const { nerve, store } = await loaded();
    const archived = track(store.fetchArchivedProjects(acme));
    await until(() => nerve.calls.length === 1, "the archived list");
    nerve.calls[0]?.answer(problem(503, "server_busy"));
    await until(() => archived.settled, "the refusal");
    expect(archived.error).toBeInstanceOf(ApiError);
    expect(store.filteredProjectIds).toBeUndefined();
    const again = track(store.fetchProjects(acme));
    await until(() => nerve.calls.length === 2, "the refetch");
    nerve.calls[1]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
  });

  it("gives nothing of a workspace the caller left, the lists and reads it holds being by id (v0 design 7.7)", async () => {
    const { nerve, workspaceRoot, store } = await loaded();
    await loadProject(nerve, store, web);
    await loadProjects(nerve, store, beta, [lab]);
    await sent(nerve, () => workspaceRoot.leaveWorkspace(acme), ["POST", "/api/v0/workspaces/acme/leave"], noContent());
    expect(store.getProjectById(web.id)).toBeUndefined();
    expect(store.getProjectById(ops.id)).toBeUndefined();
    expect(store.workspaceProjectIds).toBeUndefined();
    expect(store.joinedProjectIds).toEqual([]);
    expect(store.getProjectById(lab.id)).toEqual(lab);
  });

  it("gives nothing of a workspace deleted and made again under its slug: its lists are by the workspace's id", async () => {
    const { nerve, workspaceRoot, store } = await loaded();
    await loadProject(nerve, store, web);
    await loadWorkspaces(nerve, workspaceRoot, [workspaceOf("acme", { id: "id-acme-again", role: 20 }), beta]);
    expect(store.workspaceProjectIds).toBeUndefined();
    expect(store.getProjectById(web.id)).toBeUndefined();
    expect(store.getProjectById(ops.id)).toBeUndefined();
  });

  it("keeps the lists it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchProjects(acme), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(0);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
  });

  it("asks nerve whether an identifier is free, and fails when nerve refuses", async () => {
    const { nerve, store } = await unloaded();
    const check = () => store.checkProjectIdentifier("acme", "web");
    const request: Endpoint = ["GET", "/api/v0/workspaces/acme/project-identifiers/web"];
    const checked = await answered(nerve, check, request, { available: false }, "the check");
    expect(checked).toEqual({ settled: true, value: { available: false } });
    const refused = track(check());
    await until(() => nerve.calls.length === 2, "the second check");
    nerve.calls[1]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it). Of two fetches, only the newer writes.
describe("ProjectStore, while a fetch is out", () => {
  it("fetches the list while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    const listed = [web, ops, docs, projectOf("API", acme.id)];
    await fetchedWhileChangeIsOut(
      nerve,
      {
        send: () => store.updateProject(web.id, { name: "Web App" }),
        request: ["PATCH", `/api/v0/projects/${web.id}`],
      },
      { send: () => store.fetchProjects(acme), request: ["GET", LIST], body: { data: listed } }
    );
    expect(store.workspaceProjectIds).toEqual(ids(listed));
  });

  it("keeps the changes nerve confirmed during a refetch on the list the refetch shows, once each", async () => {
    const { nerve, store } = await loaded();
    const api = projectOf("API", acme.id);
    const refetched = track(store.fetchProjects(acme));
    await until(() => nerve.calls.length === 1, "the refetch");
    await sent(
      nerve,
      () => store.createProject("acme", { name: "API", identifier: "API" }),
      ["POST", LIST],
      json(201, api)
    );
    await sent(
      nerve,
      () => store.updateProject(web.id, { name: "Web App" }),
      ["PATCH", `/api/v0/projects/${web.id}`],
      json(200, renamed)
    );
    await sent(nerve, () => store.deleteProject(ops), ["DELETE", `/api/v0/projects/${ops.id}`], noContent());
    // the list was read after the creation, which it has where nerve lists it, and before the other two
    nerve.calls[0]?.answer(json(200, { data: [web, ops, docs, api] }));
    await until(() => refetched.settled, "the refetch");

    expect(store.workspaceProjectIds).toEqual(ids([web, docs, api]));
    expect(store.getProjectById(web.id)).toEqual(renamed);
  });

  // web as nerve answers its archiving
  const archived: Project = { ...web, archived_at: "2026-10-08T09:00:00Z" };
  it.each([
    { read: "before", listed: [old] },
    { read: "after", listed: [old, archived] },
  ])(
    "keeps a project archived during a refetch of the archived list, read $read the archiving, on it once",
    async ({ listed }) => {
      const { nerve, store } = await loaded();
      const refetched = track(store.fetchArchivedProjects(acme));
      await until(() => nerve.calls.length === 1, "the archived list");
      await sent(
        nerve,
        () => store.archiveProject(web.id),
        ["POST", `/api/v0/projects/${web.id}/archive`],
        json(200, archived)
      );
      nerve.calls[0]?.answer(json(200, { data: listed }));
      await until(() => refetched.settled, "the archived list");
      expect(store.totalProjectIds).toEqual(ids([ops, docs, old, archived]));
    }
  );

  it("shows a change nerve confirmed during a read of the project, not the read's older project", async () => {
    const { nerve, store } = await loaded();
    const read = track(store.fetchProject(web.id));
    await until(() => nerve.calls.length === 1, "the read");
    const change = () => store.updateProject(web.id, { name: "Web App" });
    await sent(nerve, change, ["PATCH", `/api/v0/projects/${web.id}`], json(200, renamed));
    // read before the change
    nerve.calls[0]?.answer(json(200, web));
    await until(() => read.settled, "the read");
    expect(store.getProjectById(web.id)).toEqual(renamed);
  });

  // The newer fetch decides: an older one answering last gives nothing, whatever nerve answered it, a failure too.
  it.each([
    { answer: "a list", reply: () => json(200, { data: [web] }) },
    { answer: "a failure", reply: () => problem(503, "server_busy") },
  ])("lets the newer of two fetches write: the older gives nothing when $answer comes last", async ({ reply }) => {
    const { nerve, store } = await loaded();
    const older = track(store.fetchProjects(acme));
    await until(() => nerve.calls.length === 1, "the older list");
    const newer = track(store.fetchProjects(acme));
    await until(() => nerve.calls.length === 2, "the newer list");
    nerve.calls[1]?.answer(json(200, { data: [docs, web] }));
    await until(() => newer.settled, "the newer list");
    nerve.calls[0]?.answer(reply());
    await until(() => older.settled, "the older list");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.workspaceProjectIds).toEqual(ids([docs, web]));
  });
});
