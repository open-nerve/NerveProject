/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Favorite } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import {
  acme,
  beta,
  favoriteFolder,
  favoriteOfTarget,
  favoritesTab,
  listOf,
  loadFavorites,
} from "@/store/favorites/fake-favorites";
import type { FavoriteStore } from "@/store/favorites/favorite.store";

// The favorites store (M7/P7 spec 2.1, R1): the caller's favorites in each workspace, read by its slug and held by its
// id, as nerve last answered them; the tree and the favorite of a target derived from them; and what follows a target
// nerve confirmed changed here (spec 2.7, R12, R7: forgotten, renamed), which sends nothing and holds against a read
// out meanwhile.

const web = favoriteOfTarget("project", "p-web", "p-web", { sequence: 90000 });
const ops = favoriteOfTarget("project", "p-ops", "p-ops", { sequence: 80000 });
const sprint = favoriteOfTarget("cycle", "c-sprint", "p-web", { sequence: 70000 });
const auth = favoriteOfTarget("module", "m-auth", "p-web", { sequence: 60000 });
const later = favoriteFolder("Later", { sequence: 50000 });
const bugs = favoriteOfTarget("view", "v-bugs", "p-web", { parent_id: later.id, sequence: 40000 });
const triage = favoriteOfTarget("view", "v-triage", null, { sequence: 30000 });
const deploys = favoriteOfTarget("cycle", "c-deploys", "p-ops", { parent_id: later.id, sequence: 20000 });
/** acme's favorites as nerve lists them: web's project, cycle, module and view (in Later), ops's, a workspace's view. */
const acmeFavorites = [web, ops, sprint, auth, later, bugs, triage, deploys];
/** beta's: one project. */
const lab = favoriteOfTarget("project", "p-lab", "p-lab", { workspace_id: beta.id });

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

/**
 * A read of acme's favorites out while follow runs, which nerve then answers with what it read before: the store shows
 * that answer with what follow confirmed made again.
 */
async function whileReadIsOut(nerve: FakeNerve, store: FavoriteStore, follow: () => void, read: Favorite[]) {
  const k = nerve.calls.length;
  const fetching = track(store.fetchFavorites(acme));
  await until(() => nerve.calls.length === k + 1, "the read");
  follow();
  nerve.calls[k]?.answer(json(200, { data: read }));
  await until(() => fetching.settled, "the read's answer");
}

describe("FavoriteStore, its reads", () => {
  it("reads a workspace's favorites by its slug and holds them by its id, each workspace its own", async () => {
    const { nerve, store } = favoritesTab();
    expect(store.getFavorites(acme.id)).toBeUndefined();
    expect(store.getFavoriteTree(acme.id)).toBeUndefined();
    expect(store.answeredAt(acme.id)).toBe(0);

    expect((await loadFavorites(nerve, store, acmeFavorites)).value).toEqual(acmeFavorites);
    expect(store.getFavorites(acme.id)).toEqual(acmeFavorites);
    expect(store.getFavorites(beta.id)).toBeUndefined();
    const acmeAnswered = store.answeredAt(acme.id);
    expect(acmeAnswered).toBeGreaterThan(0);

    await loadFavorites(nerve, store, [lab], beta);
    expect(nerve.calls.map((call) => `${call.method} ${call.path}`)).toEqual([
      `GET ${listOf(acme)}`,
      `GET ${listOf(beta)}`,
    ]);
    expect(store.getFavorites(beta.id)).toEqual([lab]);
    expect(store.getFavorites(acme.id)).toEqual(acmeFavorites);
    expect(store.answeredAt(beta.id)).toBeGreaterThan(acmeAnswered);
    expect(store.answeredAt(acme.id)).toBe(acmeAnswered);

    // its tree: the top in order, Later's favorites under it
    expect(store.getFavoriteTree(acme.id)?.map(({ favorite, children }) => [favorite.id, children.length])).toEqual([
      [web.id, 0],
      [ops.id, 0],
      [sprint.id, 0],
      [auth.id, 0],
      [later.id, 2],
      [triage.id, 0],
    ]);

    // a read again: the newer answer, answered later
    await loadFavorites(nerve, store, [web, ops]);
    expect(store.getFavorites(acme.id)).toEqual([web, ops]);
    expect(store.answeredAt(acme.id)).toBeGreaterThan(store.answeredAt(beta.id));
  });

  it("fails when nerve refuses a read, keeping what it held, and gives nothing once the session has changed", async () => {
    const { nerve, api, store } = favoritesTab();
    await loadFavorites(nerve, store, acmeFavorites);
    const answered = store.answeredAt(acme.id);
    const refused = track(store.fetchFavorites(acme));
    await until(() => nerve.calls.length === 2, "the read again");
    nerve.calls[1]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getFavorites(acme.id)).toEqual(acmeFavorites);
    expect(store.answeredAt(acme.id)).toBe(answered);

    FakeNerve.replaceSession(api);
    expect(await settle(store.fetchFavorites(acme), "the read once more")).toEqual({ settled: true, value: undefined });
    expect(await settle(store.fetchFavorites(beta), "beta's read")).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(2);
    expect(store.getFavorites(acme.id)).toEqual(acmeFavorites);
    expect(store.getFavorites(beta.id)).toBeUndefined();
  });

  it("finds a target's favorite by its kind and its id: a view's of either container, never another kind's", async () => {
    const { nerve, store } = favoritesTab();
    expect(store.favoriteOf(acme.id, "project", "p-web")).toBeUndefined();
    await loadFavorites(nerve, store, acmeFavorites);
    expect(store.favoriteOf(acme.id, "project", "p-web")).toEqual(web);
    expect(store.favoriteOf(acme.id, "cycle", "c-sprint")).toEqual(sprint);
    expect(store.favoriteOf(acme.id, "module", "m-auth")).toEqual(auth);
    // a project's view, in a folder, and a workspace's
    expect(store.favoriteOf(acme.id, "view", "v-bugs")).toEqual(bugs);
    expect(store.favoriteOf(acme.id, "view", "v-triage")).toEqual(triage);
    expect(store.favoriteOf(acme.id, "cycle", "c-deploys")).toEqual(deploys);
    // none of the target, of another kind with the same id, or in another workspace
    expect(store.favoriteOf(acme.id, "cycle", "c-none")).toBeUndefined();
    expect(store.favoriteOf(beta.id, "project", "p-web")).toBeUndefined();

    const shared = "00000000-0000-4000-8000-000000000001";
    const cycleOf = favoriteOfTarget("cycle", shared, "p-web");
    await loadFavorites(nerve, store, [cycleOf]);
    expect(store.favoriteOf(acme.id, "cycle", shared)).toEqual(cycleOf);
    expect(store.favoriteOf(acme.id, "module", shared)).toBeUndefined();
    expect(store.favoriteOf(acme.id, "project", shared)).toBeUndefined();
    expect(store.favoriteOf(acme.id, "view", shared)).toBeUndefined();
  });
});

describe("FavoriteStore, following its targets", () => {
  it("forgets a target's favorite, of its kind alone, asking nerve nothing, and holds it against a read out", async () => {
    const { nerve, store } = favoritesTab();
    await loadFavorites(nerve, store, acmeFavorites);
    const answered = store.answeredAt(acme.id);
    store.forgetTarget(acme.id, "cycle", "c-sprint");
    expect(store.getFavorites(acme.id)).toEqual([web, ops, auth, later, bugs, triage, deploys]);
    store.forgetTarget(acme.id, "view", "v-bugs");
    expect(store.getFavorites(acme.id)).toEqual([web, ops, auth, later, triage, deploys]);
    // another kind's id, another workspace's, or none held: nothing
    store.forgetTarget(acme.id, "module", "p-web");
    store.forgetTarget(beta.id, "project", "p-web");
    expect(store.getFavorites(acme.id)).toEqual([web, ops, auth, later, triage, deploys]);
    expect(store.answeredAt(acme.id)).toBe(answered);
    expect(nerve.calls).toHaveLength(1);

    // a read out while the module is forgotten, answered with what nerve read before: still without it
    await whileReadIsOut(nerve, store, () => store.forgetTarget(acme.id, "module", "m-auth"), acmeFavorites);
    expect(store.getFavorites(acme.id)).toEqual([web, ops, sprint, later, bugs, triage, deploys]);
  });

  it("forgets every favorite of a project's, its own and its cycles', modules' and views', and no other", async () => {
    const { nerve, store } = favoritesTab();
    await loadFavorites(nerve, store, acmeFavorites);
    store.forgetProject(acme.id, "p-web");
    // the folder, ops's (in the folder too) and the workspace's view stay
    expect(store.getFavorites(acme.id)).toEqual([ops, later, triage, deploys]);
    store.forgetProject(beta.id, "p-ops");
    expect(store.getFavorites(acme.id)).toEqual([ops, later, triage, deploys]);
    expect(nerve.calls).toHaveLength(1);

    await whileReadIsOut(nerve, store, () => store.forgetProject(acme.id, "p-ops"), acmeFavorites);
    expect(store.getFavorites(acme.id)).toEqual([web, sprint, auth, later, bugs, triage]);
  });

  it("renames a target's favorite with the name nerve confirmed, its icon too when given, and holds it", async () => {
    const { nerve, store } = favoritesTab();
    await loadFavorites(nerve, store, acmeFavorites);
    const answered = store.answeredAt(acme.id);
    const icon = { in_use: "emoji" as const, emoji: { value: "128640" } };
    store.renameTarget(acme.id, "cycle", "c-sprint", { name: "Sprint 2" });
    store.renameTarget(acme.id, "view", "v-triage", { name: "Inbox", logo_props: icon });
    store.renameTarget(acme.id, "view", "v-bugs", { name: "Defects" });
    const renamed = [
      web,
      ops,
      { ...sprint, name: "Sprint 2" },
      auth,
      later,
      { ...bugs, name: "Defects" },
      { ...triage, name: "Inbox", logo_props: icon },
      deploys,
    ];
    expect(store.getFavorites(acme.id)).toEqual(renamed);
    // of another kind with the same id, or none held: nothing; nerve asked nothing, the list answered no later
    const held = store.getFavorites(acme.id);
    store.renameTarget(acme.id, "module", "c-sprint", { name: "Not a module" });
    store.renameTarget(acme.id, "cycle", "c-none", { name: "None" });
    expect(store.getFavorites(acme.id)).toBe(held);
    expect(store.answeredAt(acme.id)).toBe(answered);
    expect(nerve.calls).toHaveLength(1);

    // a read out while the module is renamed, answered with the name nerve read before: the new name
    await whileReadIsOut(nerve, store, () => store.renameTarget(acme.id, "module", "m-auth", { name: "Login" }), [
      web,
      auth,
    ]);
    expect(store.getFavorites(acme.id)).toEqual([web, { ...auth, name: "Login" }]);
  });
});
