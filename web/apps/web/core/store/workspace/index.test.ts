/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceCreate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { RouterStore } from "@/store/router.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The caller's workspaces (M3 design 7.3), against a fake nerve that answers each request when the test says. That
// the store sends as the session of its RootStore is root.store.test.ts.

const LIST = "/api/v0/workspaces";
const acme = workspaceOf("acme", { role: 20 });
const beta = workspaceOf("beta");
const gamma = workspaceOf("gamma", { role: 20 });
/** acme as nerve answers its change of name. */
const renamed = { ...acme, name: "Acme Inc", updated_at: "2026-10-07T09:00:00Z" };

function setUp() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  const store = new WorkspaceRootStore(fakeRoot({ router }), api);
  return { nerve, api, router, store };
}

/** A store whose list nerve gave as acme and beta. */
async function loaded() {
  const tab = setUp();
  await loadWorkspaces(tab.nerve, tab.store, [acme, beta]);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("WorkspaceRootStore, the list", () => {
  it("lists the caller's workspaces as nerve gives them, and finds them by slug and by the address", async () => {
    const { nerve, router, store } = setUp();
    expect(store.workspaces).toBeUndefined();

    const fetched = await loadWorkspaces(nerve, store, [acme, beta]);
    expect(fetched.value).toEqual([acme, beta]);
    expect(store.workspaces).toEqual([acme, beta]);
    expect(store.getWorkspaceBySlug("beta")).toEqual(beta);
    expect(store.getWorkspaceBySlug("gamma")).toBeNull();

    expect(store.currentWorkspace).toBeNull();
    router.setQuery({ workspaceSlug: "acme" });
    expect(store.currentWorkspace).toEqual(acme);
    router.setQuery({ workspaceSlug: "gamma" });
    expect(store.currentWorkspace).toBeNull();
  });

  it("fails when nerve cannot list them, keeping the list it had", async () => {
    const { nerve, store } = setUp();
    const first = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 1, "the list");
    nerve.calls[0]?.answer(problem(503, "server_busy"));
    await until(() => first.settled, "the failure");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.workspaces).toBeUndefined();

    await loadWorkspaces(nerve, store, [acme, beta]);
    const again = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.fail();
    await until(() => again.settled, "the failure");
    // The failure is the caller's to handle (SWR's error), not an unhandled rejection.
    expect(again.error).toBeInstanceOf(TypeError);
    expect(store.workspaces).toEqual([acme, beta]);
  });

  it("fetches the list while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    // nerve lists one more than was loaded: the store holds the fetch's answer, which the refused change leaves alone
    const listed = [acme, beta, gamma];
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateWorkspace("acme", { name: "Acme Inc" }), request: ["PATCH", `${LIST}/acme`] },
      { send: () => store.fetchWorkspaces(), request: ["GET", LIST], body: { data: listed } }
    );
    expect(store.workspaces).toEqual(listed);
  });

  it("keeps the list it had, gives nothing and does not fail, when the session changes as it fetches it again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchWorkspaces(), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.workspaces).toEqual([acme, beta]);
  });

  it("asks nerve whether a slug is free, and fails when nerve cannot say", async () => {
    const { nerve, store } = setUp();
    const checked = track(store.checkWorkspaceSlug("acme"));
    await until(() => nerve.calls.length === 1, "the check");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: "/api/v0/workspace-slugs/acme" });
    nerve.calls[0]?.answer(json(200, { available: false, reason: "taken" }));
    await until(() => checked.settled, "the answer");
    expect(checked.value).toEqual({ available: false, reason: "taken" });

    const failed = track(store.checkWorkspaceSlug("beta"));
    await until(() => nerve.calls.length === 2, "the second check");
    nerve.calls[1]?.answer(problem(500, "internal_error"));
    await until(() => failed.settled, "the failure");
    expect(failed.error).toBeInstanceOf(ApiError);
  });

  it("checks a slug while a change is out: a check, a read, does not wait for it", async () => {
    const { nerve, store } = setUp();
    const checked: unknown[] = [];
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateWorkspace("acme", { name: "Acme Inc" }), request: ["PATCH", `${LIST}/acme`] },
      {
        send: async () => checked.push(await store.checkWorkspaceSlug("gamma")),
        request: ["GET", "/api/v0/workspace-slugs/gamma"],
        body: { available: true },
      }
    );
    expect(checked).toEqual([{ available: true }]);
  });
});

describe("WorkspaceRootStore, the changes", () => {
  it("adds a created workspace last to the list it has, and to none it has not fetched", async () => {
    const { nerve, store } = await loaded();
    const body: WorkspaceCreate = { name: "Gamma", slug: "gamma", organization_size: "2-10" };
    const created = { ...gamma, name: "Gamma", organization_size: "2-10" };
    const creating = track(store.createWorkspace(body));
    await until(() => nerve.calls.length === 2, "the creation");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: LIST, body });
    // until nerve answers, the list is as it was
    expect(store.workspaces).toEqual([acme, beta]);
    nerve.calls[1]?.answer(json(201, created));
    await until(() => creating.settled, "the new workspace");
    expect(creating.value).toEqual(created);
    // last, not in nerve's order by name, until the next fetch
    expect(store.workspaces).toEqual([acme, beta, created]);

    const fresh = setUp();
    const alone = track(fresh.store.createWorkspace(body));
    await until(() => fresh.nerve.calls.length === 1, "the creation");
    fresh.nerve.calls[0]?.answer(json(201, created));
    await until(() => alone.settled, "the new workspace");
    // a list of the new workspace alone would show as the whole list
    expect(fresh.store.workspaces).toBeUndefined();
  });

  it("puts nerve's answer to a change in the list, in the changed workspace's place", async () => {
    const { nerve, store } = await loaded();
    const updated = track(store.updateWorkspace("acme", { name: "Acme Inc" }));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({ method: "PATCH", path: `${LIST}/acme`, body: { name: "Acme Inc" } });
    // until nerve answers, the list is as it was
    expect(store.workspaces).toEqual([acme, beta]);
    nerve.calls[1]?.answer(json(200, renamed));
    await until(() => updated.settled, "the answer");
    expect(updated.value).toEqual(renamed);
    expect(store.workspaces).toEqual([renamed, beta]);
  });

  it("takes a deleted workspace off the list, and one the caller left", async () => {
    const { nerve, store } = await loaded();
    const deleted = track(store.deleteWorkspace(acme));
    await until(() => nerve.calls.length === 2, "the deletion");
    expect(nerve.calls[1]).toMatchObject({ method: "DELETE", path: `${LIST}/acme` });
    // until nerve answers, the workspace is still the caller's
    expect(store.workspaces).toEqual([acme, beta]);
    nerve.calls[1]?.answer(noContent());
    await until(() => deleted.settled, "the deletion");
    expect(deleted.error).toBeUndefined();
    expect(store.workspaces).toEqual([beta]);

    const left = track(store.leaveWorkspace(beta));
    await until(() => nerve.calls.length === 3, "the leave");
    expect(nerve.calls[2]).toMatchObject({ method: "POST", path: `${LIST}/beta/leave` });
    nerve.calls[2]?.answer(noContent());
    await until(() => left.settled, "the leave");
    expect(left.error).toBeUndefined();
    expect(store.workspaces).toEqual([]);
  });

  it("adds an accepted invitation's workspace to the list it has, in its own place when listed, and to none it has not fetched", async () => {
    const { nerve, store } = await loaded();
    const joined = workspaceOf("gamma");
    const accepted = track(store.acceptInvitation("i-gamma", "nrv_inv_gamma"));
    await until(() => nerve.calls.length === 2, "the acceptance");
    expect(nerve.calls[1]).toMatchObject({
      method: "POST",
      path: "/api/v0/workspace-invitations/i-gamma/accept",
      body: { token: "nrv_inv_gamma" },
    });
    expect(store.workspaces).toEqual([acme, beta]);
    nerve.calls[1]?.answer(json(200, joined));
    await until(() => accepted.settled, "the workspace");
    expect(accepted.value).toEqual(joined);
    expect(store.workspaces).toEqual([acme, beta, joined]);

    // a member accepts too, his role as it was (M3 design 3.8)
    const betaInc = { ...beta, name: "Beta Inc" };
    const again = track(store.acceptInvitation("i-beta", "nrv_inv_beta"));
    await until(() => nerve.calls.length === 3, "the acceptance");
    nerve.calls[2]?.answer(json(200, betaInc));
    await until(() => again.settled, "the workspace");
    expect(store.workspaces).toEqual([acme, betaInc, joined]);

    const fresh = setUp();
    const alone = track(fresh.store.acceptInvitation("i-gamma", "nrv_inv_gamma"));
    await until(() => fresh.nerve.calls.length === 1, "the acceptance");
    fresh.nerve.calls[0]?.answer(json(200, joined));
    await until(() => alone.settled, "the workspace");
    expect(fresh.store.workspaces).toBeUndefined();
  });

  it("declines an invitation, the caller's workspaces staying as they are", async () => {
    const { nerve, store } = await loaded();
    const declined = track(store.declineInvitation("i-gamma", "nrv_inv_gamma"));
    await until(() => nerve.calls.length === 2, "the decline");
    expect(nerve.calls[1]).toMatchObject({
      method: "POST",
      path: "/api/v0/workspace-invitations/i-gamma/decline",
      body: { token: "nrv_inv_gamma" },
    });
    nerve.calls[1]?.answer(noContent());
    await until(() => declined.settled, "the decline");
    expect(declined.error).toBeUndefined();
    expect(store.workspaces).toEqual([acme, beta]);
  });

  const refusals: { change: string; send: (store: WorkspaceRootStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a creation",
      send: (store) => store.createWorkspace({ name: "A", slug: "acme" }),
      refusal: problem(409, "workspace.slug_taken"),
    },
    {
      change: "a change",
      send: (store) => store.updateWorkspace("beta", { name: "B" }),
      refusal: problem(403, "forbidden"),
    },
    { change: "a deletion", send: (store) => store.deleteWorkspace(beta), refusal: problem(403, "forbidden") },
    {
      change: "a leave",
      send: (store) => store.leaveWorkspace(acme),
      refusal: problem(409, "workspace.sole_admin"),
    },
    {
      change: "an acceptance",
      send: (store) => store.acceptInvitation("i-gamma", "nrv_inv_gamma"),
      refusal: problem(403, "workspace.invitation_email_mismatch"),
    },
    {
      change: "a decline",
      send: (store) => store.declineInvitation("i-gamma", "nrv_inv_gamma"),
      refusal: problem(409, "workspace.invitation_responded"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const sent = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => sent.settled, "the refusal");
    expect(sent.error).toBeInstanceOf(ApiError);
    expect(store.workspaces).toEqual([acme, beta]);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const delta = workspaceOf("delta");
    const updated = track(store.updateWorkspace("acme", { timezone: "Asia/Shanghai" }));
    const created = track(store.createWorkspace({ name: "gamma", slug: "gamma" }));
    const deleted = track(store.deleteWorkspace(beta));
    const left = track(store.leaveWorkspace(acme));
    const accepted = track(store.acceptInvitation("i-delta", "nrv_inv_delta"));
    const declined = track(store.declineInvitation("i-gamma", "nrv_inv_gamma"));
    await inTurn(nerve, 1, ["PATCH", `${LIST}/acme`], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["POST", LIST], json(201, gamma));
    await inTurn(nerve, 3, ["DELETE", `${LIST}/beta`], noContent());
    await inTurn(nerve, 4, ["POST", `${LIST}/acme/leave`], noContent());
    await inTurn(nerve, 5, ["POST", "/api/v0/workspace-invitations/i-delta/accept"], json(200, delta));
    await inTurn(nerve, 6, ["POST", "/api/v0/workspace-invitations/i-gamma/decline"], noContent());
    await until(() => declined.settled, "the last change");
    expect(updated.error).toBeInstanceOf(ApiError);
    expect(created.value).toEqual(gamma);
    expect(deleted.error).toBeUndefined();
    expect(left.error).toBeUndefined();
    expect(accepted.value).toEqual(delta);
    expect(declined.error).toBeUndefined();
    expect(store.workspaces).toEqual([gamma, delta]);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it, after the change has finished). Of two
// fetches, only the newer writes.
describe("WorkspaceRootStore, while a fetch is out", () => {
  it("lists a workspace created during the first load once, when the load lists it too", async () => {
    const { nerve, store } = setUp();
    const fetched = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 1, "the list");
    const created = track(store.createWorkspace({ name: "gamma", slug: "gamma" }));
    await inTurn(nerve, 1, ["POST", LIST], json(201, gamma));
    await until(() => created.settled, "the new workspace");
    // the list was read after gamma was made
    nerve.calls[0]?.answer(json(200, { data: [acme, beta, gamma] }));
    await until(() => fetched.settled, "the list");

    expect(store.workspaces).toEqual([acme, beta, gamma]);
    expect(fetched.value).toEqual(store.workspaces);
  });

  it("keeps a change and a deletion nerve confirmed during a refetch on the list the refetch shows", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 2, "the refetch");
    const updated = track(store.updateWorkspace("acme", { name: "Acme Inc" }));
    await inTurn(nerve, 2, ["PATCH", `${LIST}/acme`], json(200, renamed));
    const deleted = track(store.deleteWorkspace(beta));
    await inTurn(nerve, 3, ["DELETE", `${LIST}/beta`], noContent());
    await until(() => updated.settled && deleted.settled, "the changes");
    expect(store.workspaces).toEqual([renamed]);
    // the list was read before both
    nerve.calls[1]?.answer(json(200, { data: [acme, beta] }));
    await until(() => refetched.settled, "the refetch");

    expect(store.workspaces).toEqual([renamed]);
  });

  it("drops a workspace deleted during a refetch by its id: one made since under its slug stays listed", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 2, "the refetch");
    const deleted = track(store.deleteWorkspace(acme));
    await inTurn(nerve, 2, ["DELETE", `${LIST}/acme`], noContent());
    await until(() => deleted.settled, "the deletion");
    // the list was read once another tab of the caller's had made acme again
    const remade = workspaceOf("acme", { id: "id-acme-2", role: 20 });
    nerve.calls[1]?.answer(json(200, { data: [remade, beta] }));
    await until(() => refetched.settled, "the refetch");

    expect(store.workspaces).toEqual([remade, beta]);
  });

  it("lets the newer of two fetches write: the older writes nothing and gives nothing when it answers last", async () => {
    const { nerve, store } = setUp();
    const older = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 1, "the older list");
    const newer = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 2, "the newer list");
    nerve.calls[1]?.answer(json(200, { data: [acme, beta, gamma] }));
    await until(() => newer.settled, "the newer list");
    // read before gamma was made elsewhere
    nerve.calls[0]?.answer(json(200, { data: [acme, beta] }));
    await until(() => older.settled, "the older list");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.workspaces).toEqual([acme, beta, gamma]);
  });
});
