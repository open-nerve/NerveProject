/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { View, WorkspaceRole } from "@nerve/api-client";
import { EUserWorkspaceRoles } from "@nerve/types";
import { answered, json, noContent } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { fakeRoot } from "@/store/fake-root";
import { inTurn, sent } from "@/store/fake-queue";
import { projectOf, projectTab } from "@/store/project/fake-projects";
import { viewOf, workspaceViewOf } from "@/store/views/fake-views";
import type { ViewContainer } from "@/store/views/view-container";
import { ViewStore } from "@/store/views/view.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// A workspace's own views in the views' store (M7/P6b spec 2.6, 2.7; R9, R13), against a fake nerve that answers each
// request when the test says: held by the workspace's id as a project's are by the project's, created into that list,
// changed and deleted there and in the view's own read; shown to the workspace's admins and members only
// (workspaceViewsOn), never to its guests, nor once the caller has left the workspace. Each container's creations go
// one at a time, apart from another's. A project's views are as ever (view.store.test.ts, view-changes.test.ts).

const { ADMIN, MEMBER, GUEST } = EUserWorkspaceRoles;
/** The caller's workspaces: acme, of the role each test gives him, with its project web; beta, his as a member's. */
const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id);

/** Times after the fixtures', of nerve's answers to the changes. */
const LATER = "2026-11-10T09:00:00Z";
const roadmap = workspaceViewOf(acme, "Roadmap", "u-me", { created_at: "2026-11-03T09:00:00Z" });
const triage = workspaceViewOf(acme, "Triage", "u-bob");
/** acme's views as nerve lists them: the newest first. */
const acmeViews = [roadmap, triage];
/** A view created in acme, as nerve answers it: the newest. */
const release = workspaceViewOf(acme, "Release", "u-me", { created_at: LATER, updated_at: LATER });
const plans = workspaceViewOf(beta, "Plans", "u-me");
const bugs = viewOf(web, "Bugs", "u-me");

/** The containers of the views: acme's, beta's, and web's. */
const inAcme: ViewContainer = { kind: "workspace", workspaceId: acme.id, workspaceSlug: acme.slug };
const inBeta: ViewContainer = { kind: "workspace", workspaceId: beta.id, workspaceSlug: beta.slug };
const inWeb: ViewContainer = { kind: "project", projectId: web.id };
/** The requests of the views: a workspace's, where it creates its views; a view's, read, changed and deleted there. */
const list = (slug: string) => `/api/v0/workspaces/${slug}/views`;
const at = (view: View) => `/api/v0/views/${view.id}`;

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

/**
 * A views' store of a tab whose caller has role in acme and is beta's member; acme's views held as nerve listed them
 * (the store's list, as its fetch writes it), and beta's.
 */
async function workspaceTab(role: WorkspaceRole) {
  const { nerve, workspaceRoot, projectRoot } = await projectTab(
    { workspace: { ...acme, role }, projects: [web] },
    { workspace: beta }
  );
  const store = new ViewStore(fakeRoot({ workspaceRoot, projectRoot }), nerve.client());
  store.lists.receive(acme.id, acmeViews, (_held, answer) => answer);
  store.lists.receive(beta.id, [plans], (_held, answer) => answer);
  return { nerve, workspaceRoot, store };
}

/** nerve answers the view's own read with read. */
const loadView = (tab: Awaited<ReturnType<typeof workspaceTab>>, read: View) =>
  answered(tab.nerve, () => tab.store.fetchView(read.id), ["GET", at(read)], read, "the view");

/** What the store gives of acme's and beta's views, and of web's Bugs. */
const shown = (store: ViewStore) => [roadmap, triage, plans, bugs].map((view) => store.getViewById(view.id)?.name);

describe("ViewStore, a workspace's own views", () => {
  it.each([
    ["admin", ADMIN],
    ["member", MEMBER],
  ])(
    "shows its %s a workspace's views, each as nerve last answered it, its list's row or its read",
    async (_, role) => {
      const tab = await workspaceTab(role);
      const { store } = tab;
      expect(store.getViewById(roadmap.id)).toEqual(roadmap);
      expect(store.getViewById(triage.id)).toEqual(triage);
      // another tab renames it, and its read, answered after the list, shows the name; the list keeps its answer
      const renamed: View = { ...roadmap, name: "Plan", updated_at: LATER };
      await loadView(tab, renamed);
      expect(store.getViewById(roadmap.id)).toEqual(renamed);
      expect(store.lists.get(acme.id)).toEqual(acmeViews);
      // the list, answered after the read, shows the view as it has it
      const relisted: View = { ...roadmap, name: "The plan", updated_at: "2026-11-10T10:00:00Z" };
      store.lists.receive(acme.id, [relisted, triage], (_held, answer) => answer);
      expect(store.getViewById(roadmap.id)).toEqual(relisted);
    }
  );

  it("shows a workspace's guest none of its views, held or read; a project's view by the project as ever", async () => {
    const tab = await workspaceTab(GUEST);
    await loadView(tab, roadmap);
    await loadView(tab, bugs);
    expect(shown(tab.store)).toEqual([undefined, undefined, "Plans", "Bugs"]);
  });

  it("shows none once the caller is the workspace's guest, or has left it; another workspace's as ever", async () => {
    const tab = await workspaceTab(MEMBER);
    const { nerve, workspaceRoot, store } = tab;
    await loadView(tab, bugs);
    expect(shown(store)).toEqual(["Roadmap", "Triage", "Plans", "Bugs"]);
    await loadWorkspaces(nerve, workspaceRoot, [{ ...acme, role: GUEST }, beta]);
    expect(shown(store)).toEqual([undefined, undefined, "Plans", "Bugs"]);
    await loadWorkspaces(nerve, workspaceRoot, [{ ...acme, role: ADMIN }, beta]);
    expect(shown(store)).toEqual(["Roadmap", "Triage", "Plans", "Bugs"]);

    await sent(nerve, () => workspaceRoot.leaveWorkspace(acme), ["POST", "/api/v0/workspaces/acme/leave"], noContent());
    expect(shown(store)).toEqual([undefined, undefined, "Plans", undefined]);
  });
});

describe("ViewStore, the changes of a workspace's own views", () => {
  it("creates a view in the workspace by its slug: first of its views, as nerve lists them, the newest first", async () => {
    const { nerve, store } = await workspaceTab(MEMBER);
    const body = { name: "Release" };
    const creating = await sent(nerve, () => store.create(inAcme, body), ["POST", list("acme")], json(201, release));
    expect(nerve.calls.at(-1)?.body).toEqual(body);
    expect(creating.value).toEqual(release);
    expect(store.lists.get(acme.id)).toEqual([release, ...acmeViews]);
    expect(store.getViewById(release.id)).toEqual(release);
    expect(store.lists.get(beta.id)).toEqual([plans]);
  });

  it("changes a workspace's view: nerve's answer replaces it in the workspace's views and in its read", async () => {
    const tab = await workspaceTab(MEMBER);
    const { nerve, store } = tab;
    await loadView(tab, roadmap);
    const changed: View = { ...roadmap, name: "Plan", filters: { priority: ["urgent"] }, updated_at: LATER };
    const body = { name: "Plan", filters: { priority: ["urgent"] } };
    await sent(nerve, () => store.update(roadmap.id, () => body), ["PATCH", at(roadmap)], json(200, changed));
    expect(nerve.calls.at(-1)?.body).toEqual(body);
    expect(store.lists.get(acme.id)).toEqual([changed, triage]);
    expect(store.details.get(roadmap.id)).toEqual(changed);
    expect(store.getViewById(roadmap.id)).toEqual(changed);
  });

  it("deletes a workspace's view: out of the workspace's views, its read null; it shows no more", async () => {
    const tab = await workspaceTab(ADMIN);
    const { nerve, store } = tab;
    await loadView(tab, triage);
    await sent(nerve, () => store.remove(triage.id), ["DELETE", at(triage)], noContent());
    expect(store.lists.get(acme.id)).toEqual([roadmap]);
    expect(store.details.get(triage.id)).toBeNull();
    expect(store.getViewById(triage.id)).toBeUndefined();
    // one the store has no read of: its row goes
    await sent(nerve, () => store.remove(roadmap.id), ["DELETE", at(roadmap)], noContent());
    expect(store.lists.get(acme.id)).toEqual([]);
    expect(store.getViewById(roadmap.id)).toBeUndefined();
  });

  it("creates each container's views one at a time, apart: a workspace's do not wait for a project's", async () => {
    const { nerve, store } = await workspaceTab(MEMBER);
    const sentNow = () => nerve.calls.map(({ method, path, body }) => [method, path, body]);
    const inProject = track(store.create(inWeb, { name: "Bugs" }));
    await until(() => nerve.calls.length === 1, "the project's creation");
    // while the project's is out, the workspaces' go: one for each
    const creating = [track(store.create(inAcme, { name: "Release" })), track(store.create(inAcme, { name: "Next" }))];
    const elsewhere = track(store.create(inBeta, { name: "Later" }));
    await vi.advanceTimersByTimeAsync(1_000);
    expect(sentNow()).toEqual([
      ["POST", `/api/v0/projects/${web.id}/views`, { name: "Bugs" }],
      ["POST", list("acme"), { name: "Release" }],
      ["POST", list("beta"), { name: "Later" }],
    ]);
    nerve.calls[1]?.answer(json(201, release));
    const next = workspaceViewOf(acme, "Next", "u-me", { created_at: "2026-11-10T10:00:00Z" });
    await inTurn(nerve, 3, ["POST", list("acme")], json(201, next));
    nerve.calls[2]?.answer(json(201, workspaceViewOf(beta, "Later", "u-me", { created_at: LATER })));
    nerve.calls[0]?.answer(json(201, bugs));
    await until(() => [...creating, elsewhere, inProject].every((change) => change.settled), "the creations");
    expect(store.lists.get(acme.id)).toEqual([next, release, ...acmeViews]);
    expect(store.lists.get(beta.id)?.map((view) => view.name)).toEqual(["Later", "Plans"]);
  });
});

describe("ViewStore, a workspace's own views as its pages list them", () => {
  it.each([
    ["admin", ADMIN],
    ["member", MEMBER],
  ])(
    "reads them by the workspace's slug, holds them by its id, and gives its %s each as nerve last answered it",
    async (_, role) => {
      const tab = await workspaceTab(role);
      const { nerve, store } = tab;
      const listed = await answered(
        nerve,
        () => store.fetchWorkspaceViews({ id: acme.id, slug: acme.slug }),
        ["GET", list("acme")],
        { data: [release, ...acmeViews] },
        "acme's views"
      );
      expect(listed.value).toEqual([release, ...acmeViews]);
      expect(store.getWorkspaceViews("acme")).toEqual([release, ...acmeViews]);
      expect(store.getWorkspaceViews("beta")).toEqual([plans]);
      // a read of one view answered after the list shows in the list
      const renamed: View = { ...triage, name: "Inbox", updated_at: LATER };
      await loadView(tab, renamed);
      expect(store.getWorkspaceViews("acme")?.map((view) => view.name)).toEqual(["Release", "Roadmap", "Inbox"]);
    }
  );

  it("gives its guest none, held or not, nor one who has no role in the workspace", async () => {
    const { store } = await workspaceTab(GUEST);
    expect(store.lists.get(acme.id), "acme's views as nerve listed them before").toEqual(acmeViews);
    expect(store.getWorkspaceViews("acme")).toBeUndefined();
    expect(store.getWorkspaceViews("gamma")).toBeUndefined();
  });
});
