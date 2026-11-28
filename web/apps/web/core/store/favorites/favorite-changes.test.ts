/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError, problemCode } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sent } from "@/store/fake-queue";
import {
  acme,
  at,
  beta,
  favoriteFolder,
  favoriteOfTarget,
  favoritesTab,
  listOf,
  loadFavorites,
} from "@/store/favorites/fake-favorites";
import type { FavoriteStore } from "@/store/favorites/favorite.store";

// The changes of one's favorites (M7/P7 spec 2.1, R3, re-98 m2, o3), against a fake nerve that answers each request
// when the test says: one queue for each workspace; each change an intent whose body is reckoned in its turn from the
// list nerve answered last, so that the same intent twice sends once; nothing written before nerve confirms, nothing
// when it refuses, but a favorite nerve says is not there, which goes; and what is queued or out, by both its keys.

const web = favoriteOfTarget("project", "p-web", "p-web", { sequence: 90000 });
const sprint = favoriteOfTarget("cycle", "c-sprint", "p-web", { sequence: 80000 });
const later = favoriteFolder("Later", { sequence: 70000 });
const bugs = favoriteOfTarget("view", "v-bugs", "p-web", { parent_id: later.id, sequence: 60000 });
const auth = favoriteOfTarget("module", "m-auth", "p-web", { parent_id: later.id, sequence: 50000 });
const soon = favoriteFolder("Soon", { sequence: 40000 });
/** acme's favorites as nerve lists them: web's project and cycle, Later holding a view and a module, Soon empty. */
const acmeFavorites = [web, sprint, later, bugs, auth, soon];
/** A favorite nerve makes: ops's project, 10000 past the greatest. */
const ops = favoriteOfTarget("project", "p-ops", "p-ops", { sequence: 100000 });
/** nerve's answer that a favorite is not there (each answer its own: a body is read once). */
const notFound = () => problem(404, "navigation.favorite_not_found");

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

/** A tab whose store holds acme's favorites; nerve has been asked that alone. */
async function loaded() {
  const tab = favoritesTab();
  await loadFavorites(tab.nerve, tab.store, acmeFavorites);
  return tab;
}

/** The requests nerve received, as "METHOD path". */
const requests = (nerve: FakeNerve) => nerve.calls.map(({ method, path }) => `${method} ${path}`);

/** The k-th request arrives, and nothing after it for a second. */
async function out(nerve: FakeNerve, k: number) {
  await until(() => nerve.calls.length === k + 1, `request ${k}`);
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls).toHaveLength(k + 1);
}

/** Each change, as a page makes it, and the request it sends to acme's held list. */
const changes: { name: string; send: (store: FavoriteStore) => Promise<unknown>; request: Endpoint }[] = [
  { name: "add", send: (store) => store.add(acme, "project", "p-ops"), request: ["POST", listOf(acme)] },
  { name: "remove", send: (store) => store.remove(acme, "cycle", "c-sprint"), request: ["DELETE", at(sprint)] },
  { name: "removeFavorite", send: (store) => store.removeFavorite(acme, later.id), request: ["DELETE", at(later)] },
  { name: "createFolder", send: (store) => store.createFolder(acme, "Someday"), request: ["POST", listOf(acme)] },
  {
    name: "renameFolder",
    send: (store) => store.renameFolder(acme, later.id, "Someday"),
    request: ["PATCH", at(later)],
  },
  {
    name: "move",
    send: (store) => store.move(acme, web.id, { at: "below", rowId: sprint.id }),
    request: ["PATCH", at(web)],
  },
];

describe("FavoriteStore, the changes", () => {
  it("favors a target with its kind and id alone; nerve's answer, new (201) or as it was (200), held once", async () => {
    const { nerve, store } = await loaded();
    const adding = await sent(nerve, () => store.add(acme, "project", "p-ops"), ["POST", listOf(acme)], json(201, ops));
    expect(nerve.calls[1]?.body).toEqual({ entity_type: "project", entity_id: "p-ops" });
    expect(adding.value).toEqual(ops);
    expect(store.getFavorites(acme.id)).toEqual([...acmeFavorites, ops]);

    // another tab had favored the module: nerve answers that one as it is, in Later
    const deploys = favoriteOfTarget("module", "m-deploys", "p-web", { parent_id: later.id, sequence: 30000 });
    const had = await sent(
      nerve,
      () => store.add(acme, "module", "m-deploys"),
      ["POST", listOf(acme)],
      json(200, deploys)
    );
    expect(nerve.calls[2]?.body).toEqual({ entity_type: "module", entity_id: "m-deploys" });
    expect(had.value).toEqual(deploys);
    expect(store.getFavorites(acme.id)).toEqual([...acmeFavorites, ops, deploys]);
    expect(store.favoriteOf(acme.id, "module", "m-deploys")).toEqual(deploys);
  });

  it("favors a target once: the second add finds the first's favorite in its turn and sends nothing", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.add(acme, "project", "p-ops"));
    const second = track(store.add(acme, "project", "p-ops"));
    await out(nerve, 1);
    nerve.calls[1]?.answer(json(201, ops));
    await until(() => first.settled && second.settled, "both");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(requests(nerve)).toEqual([`GET ${listOf(acme)}`, `POST ${listOf(acme)}`]);
    expect([first.value, second.value]).toEqual([ops, ops]);
    expect(store.getFavorites(acme.id)).toEqual([...acmeFavorites, ops]);
  });

  it("unfavors a target; its body reckoned in its turn: added back once removed, nothing once gone", async () => {
    const { nerve, store } = await loaded();
    await sent(nerve, () => store.remove(acme, "cycle", "c-sprint"), ["DELETE", at(sprint)], noContent());
    expect(store.getFavorites(acme.id)).toEqual([web, later, bugs, auth, soon]);
    // none held: nothing sent
    const none = track(store.remove(acme, "cycle", "c-sprint"));
    await until(() => none.settled, "nothing");
    expect(nerve.calls).toHaveLength(2);

    // the star removes web while it is out, and adds it again: held when pressed, gone in its turn, so favored anew
    const removing = track(store.remove(acme, "project", "p-web"));
    await out(nerve, 2);
    const adding = track(store.add(acme, "project", "p-web"));
    nerve.calls[2]?.answer(noContent());
    const again = { ...web, id: "f-project-p-web-2", sequence: 100000 };
    await out(nerve, 3);
    expect(nerve.calls[3]).toMatchObject({ method: "POST", body: { entity_type: "project", entity_id: "p-web" } });
    nerve.calls[3]?.answer(json(201, again));
    await until(() => removing.settled && adding.settled, "both");
    expect(store.getFavorites(acme.id)).toEqual([later, bugs, auth, soon, again]);

    // the row's menu removes again while it is out; the star's removal finds none in its turn and sends nothing
    const fromMenu = track(store.removeFavorite(acme, again.id));
    await out(nerve, 4);
    const fromStar = track(store.remove(acme, "project", "p-web"));
    nerve.calls[4]?.answer(noContent());
    await until(() => fromMenu.settled && fromStar.settled, "both");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(5);
    expect([fromMenu.error, fromStar.error]).toEqual([undefined, undefined]);
    expect(store.getFavorites(acme.id)).toEqual([later, bugs, auth, soon]);
  });

  it.each(changes)("$name: nothing written before nerve answers, nor when it refuses", async ({ send, request }) => {
    const { nerve, store } = await loaded();
    const before = store.getFavorites(acme.id);
    const change = track(send(store));
    await out(nerve, 1);
    expect(nerve.calls[1]).toMatchObject({ method: request[0], path: request[1] });
    expect(store.getFavorites(acme.id)).toBe(before);
    nerve.calls[1]?.answer(problem(403, "forbidden"));
    await until(() => change.settled, "the refusal");
    expect(change.error).toBeInstanceOf(ApiError);
    expect(store.getFavorites(acme.id)).toBe(before);
  });

  it("changes nothing once the session has changed: each change fails before it is sent", async () => {
    const { nerve, api, store } = await loaded();
    const before = store.getFavorites(acme.id);
    FakeNerve.replaceSession(api);
    const cut = changes.map(({ send }) => track(send(store)));
    await until(() => cut.every((change) => change.settled), "the changes");
    expect(cut.map((change) => change.error)).toEqual(changes.map(() => expect.any(SessionChangedError)));
    expect(nerve.calls).toHaveLength(1);
    expect(store.getFavorites(acme.id)).toBe(before);
  });

  it("removes a folder with every favorite it holds, and a target's favorite from its row", async () => {
    const { nerve, store } = await loaded();
    await sent(nerve, () => store.removeFavorite(acme, later.id), ["DELETE", at(later)], noContent());
    expect(store.getFavorites(acme.id)).toEqual([web, sprint, soon]);
    await sent(nerve, () => store.removeFavorite(acme, sprint.id), ["DELETE", at(sprint)], noContent());
    expect(store.getFavorites(acme.id)).toEqual([web, soon]);
    // one not held: nothing sent
    const none = track(store.removeFavorite(acme, later.id));
    await until(() => none.settled, "nothing");
    expect(nerve.calls).toHaveLength(3);
  });

  it("makes a folder of the name given; nerve's refusal of a name taken fails it, writing nothing", async () => {
    const { nerve, store } = await loaded();
    const someday = favoriteFolder("Someday", { sequence: 100000 });
    const making = await sent(
      nerve,
      () => store.createFolder(acme, "Someday"),
      ["POST", listOf(acme)],
      json(201, someday)
    );
    expect(nerve.calls[1]?.body).toEqual({ entity_type: "folder", name: "Someday" });
    expect(making.value).toEqual(someday);
    expect(store.getFavorites(acme.id)).toEqual([...acmeFavorites, someday]);

    const taken = await sent(
      nerve,
      () => store.createFolder(acme, "LATER"),
      ["POST", listOf(acme)],
      problem(409, "navigation.favorite_folder_name_taken")
    );
    expect(problemCode(taken.error)).toBe("navigation.favorite_folder_name_taken");
    expect(store.getFavorites(acme.id)).toEqual([...acmeFavorites, someday]);
  });

  it("renames a folder with its name alone; nothing sent for the name it has in its turn, or once it is gone", async () => {
    const { nerve, store } = await loaded();
    const someday = { ...later, name: "Someday" };
    await sent(nerve, () => store.renameFolder(acme, later.id, "Someday"), ["PATCH", at(later)], json(200, someday));
    expect(nerve.calls[1]?.body).toEqual({ name: "Someday" });
    expect(store.getFavorites(acme.id)).toEqual([web, sprint, someday, bugs, auth, soon]);
    const same = track(store.renameFolder(acme, later.id, "Someday"));
    await until(() => same.settled, "nothing");

    const removing = track(store.removeFavorite(acme, soon.id));
    await out(nerve, 2);
    const renaming = track(store.renameFolder(acme, soon.id, "Never"));
    nerve.calls[2]?.answer(noContent());
    await until(() => removing.settled && renaming.settled, "both");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(requests(nerve)).toEqual([`GET ${listOf(acme)}`, `PATCH ${at(later)}`, `DELETE ${at(soon)}`]);
    expect(store.getFavorites(acme.id)).toEqual([web, sprint, someday, bugs, auth]);
  });

  it("moves with one PATCH of what changes, reckoned in its turn; nothing once it would not move", async () => {
    const { nerve, store } = await loaded();
    const moved = { ...web, sequence: 75000 };
    await sent(
      nerve,
      () => store.move(acme, web.id, { at: "below", rowId: sprint.id }),
      ["PATCH", at(web)],
      json(200, moved)
    );
    expect(nerve.calls[1]?.body).toEqual({ sequence: 75000 });
    expect(store.getFavorites(acme.id)).toEqual([moved, sprint, later, bugs, auth, soon]);

    // the cycle into Later while Soon is dropped below it: in Soon's turn the cycle is in a folder, which holds none
    const into = track(store.move(acme, sprint.id, { at: "into", folderId: later.id }));
    await out(nerve, 2);
    expect(nerve.calls[2]?.body).toEqual({ parent_id: later.id });
    const folder = track(store.move(acme, soon.id, { at: "below", rowId: sprint.id }));
    nerve.calls[2]?.answer(json(200, { ...sprint, parent_id: later.id }));
    await until(() => into.settled && folder.settled, "both");

    // the module, removed while its move waits: gone in its turn
    const removing = track(store.removeFavorite(acme, auth.id));
    await out(nerve, 3);
    const gone = track(store.move(acme, auth.id, { at: "out" }));
    nerve.calls[3]?.answer(noContent());
    await until(() => removing.settled && gone.settled, "both");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(requests(nerve)).toEqual([
      `GET ${listOf(acme)}`,
      `PATCH ${at(web)}`,
      `PATCH ${at(sprint)}`,
      `DELETE ${at(auth)}`,
    ]);
    expect([folder.error, gone.error]).toEqual([undefined, undefined]);
    expect(store.getFavorites(acme.id)).toEqual([moved, { ...sprint, parent_id: later.id }, later, bugs, soon]);
  });

  it("drops a favorite nerve says is not there: a removal so done, a rename or a move still failed", async () => {
    const { nerve, store } = await loaded();
    const removed = await sent(
      nerve,
      () => store.remove(acme, "cycle", "c-sprint"),
      ["DELETE", at(sprint)],
      notFound()
    );
    expect(removed.error).toBeUndefined();
    expect(store.getFavorites(acme.id)).toEqual([web, later, bugs, auth, soon]);
    const renamed = await sent(nerve, () => store.renameFolder(acme, later.id, "X"), ["PATCH", at(later)], notFound());
    expect(problemCode(renamed.error)).toBe("navigation.favorite_not_found");
    // the folder goes with what it holds
    expect(store.getFavorites(acme.id)).toEqual([web, soon]);
    const moving = await sent(
      nerve,
      () => store.move(acme, web.id, { at: "below", rowId: soon.id }),
      ["PATCH", at(web)],
      notFound()
    );
    expect(problemCode(moving.error)).toBe("navigation.favorite_not_found");
    expect(store.getFavorites(acme.id)).toEqual([soon]);
    const fromMenu = await sent(nerve, () => store.removeFavorite(acme, soon.id), ["DELETE", at(soon)], notFound());
    expect(fromMenu.error).toBeUndefined();
    expect(store.getFavorites(acme.id)).toEqual([]);
  });

  it("holds one copy of a favorite made while a read is out whose answer has it already", async () => {
    const { nerve, store } = await loaded();
    const reading = track(store.fetchFavorites(acme));
    await out(nerve, 1);
    await sent(nerve, () => store.add(acme, "project", "p-ops"), ["POST", listOf(acme)], json(201, ops));
    expect(store.getFavorites(acme.id)).toEqual([...acmeFavorites, ops]);
    nerve.calls[1]?.answer(json(200, { data: [ops, ...acmeFavorites] }));
    await until(() => reading.settled, "the read");
    expect(store.getFavorites(acme.id)).toEqual([ops, ...acmeFavorites]);
  });
});

describe("FavoriteStore, the queues", () => {
  it("queues each workspace's changes on their own: beta's is sent while acme's is out", async () => {
    const { nerve, store } = await loaded();
    const lab = favoriteOfTarget("project", "p-lab", "p-lab", { workspace_id: beta.id });
    await loadFavorites(nerve, store, [lab], beta);
    const acmeFirst = track(store.add(acme, "project", "p-ops"));
    await out(nerve, 2);
    const acmeNext = track(store.remove(acme, "project", "p-web"));
    const betaOwn = track(store.remove(beta, "project", "p-lab"));
    await out(nerve, 3);
    expect(nerve.calls[3]).toMatchObject({ method: "DELETE", path: at(lab) });
    nerve.calls[3]?.answer(noContent());
    await until(() => betaOwn.settled, "beta's");
    expect([acmeNext.settled, store.getFavorites(beta.id)]).toEqual([false, []]);
    nerve.calls[2]?.answer(json(201, ops));
    await out(nerve, 4);
    expect(nerve.calls[4]).toMatchObject({ method: "DELETE", path: at(web) });
    nerve.calls[4]?.answer(noContent());
    await until(() => acmeFirst.settled && acmeNext.settled, "acme's");
  });

  it("tells a change queued or out by its target's key and its favorite's id, both whichever entry made it", async () => {
    const { nerve, store } = await loaded();
    const pending = (...keys: string[]) => keys.map((key) => store.isPending(acme.id, key));
    expect(pending("cycle:c-sprint", sprint.id)).toEqual([false, false]);
    // the row's menu removes the cycle: its star's key too
    const fromMenu = track(store.removeFavorite(acme, sprint.id));
    await out(nerve, 1);
    expect(pending("cycle:c-sprint", sprint.id)).toEqual([true, true]);
    expect(store.isPending(beta.id, "cycle:c-sprint")).toBe(false);
    // the star removes the project, queued behind it: its favorite's id too; a target never favored, its key alone
    const fromStar = track(store.remove(acme, "project", "p-web"));
    const adding = track(store.add(acme, "project", "p-ops"));
    const renaming = track(store.renameFolder(acme, later.id, "Someday"));
    expect(pending("project:p-web", web.id, "project:p-ops", later.id)).toEqual([true, true, true, true]);
    nerve.calls[1]?.answer(noContent());
    await out(nerve, 2);
    expect(pending("cycle:c-sprint", sprint.id, "project:p-web")).toEqual([false, false, true]);
    nerve.calls[2]?.answer(noContent());
    await out(nerve, 3);
    nerve.calls[3]?.answer(json(201, ops));
    await out(nerve, 4);
    nerve.calls[4]?.answer(problem(403, "forbidden"));
    await until(() => [fromMenu, fromStar, adding, renaming].every((change) => change.settled), "all");
    expect(pending("project:p-web", web.id, "project:p-ops", ops.id, later.id)).toEqual([
      false,
      false,
      false,
      false,
      false,
    ]);
  });
});
