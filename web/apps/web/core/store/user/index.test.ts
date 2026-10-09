/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { fakeRoot } from "@/store/fake-root";
import { projectOf, projectTab } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { UserStore } from "@/store/user";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The account's changes (names, time zone) against a fake nerve that answers each when the test says: a change goes
// out once the one before it is answered or has failed, so nerve applies them in the order they were made and the
// last answer is what nerve holds. The store gets its session's client from RootStore; of the token manager it
// imports, the deactivation reads the tab's session and ends it (tab).
const tab = vi.hoisted(() => ({
  state: { loginId: "x" },
  endSession: vi.fn<(loginId: string | undefined) => Promise<boolean>>(),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: tab, publicClient: {} }));

const ME = "/api/v0/me";

/** nerve's answer to a change of the time zone: the account in it. */
const accountIn = (user_timezone: string) => ({ first_name: "Ada", user_timezone }) as User;

/** A store, and two changes made one after the other, to Shanghai and then to Berlin: only the first is out. */
async function twoChanges() {
  const nerve = new FakeNerve();
  const store = new UserStore({} as RootStore, nerve.client());
  const first = track(store.updateCurrentUser({ user_timezone: "Asia/Shanghai" }));
  const second = track(store.updateCurrentUser({ user_timezone: "Europe/Berlin" }));
  await until(() => nerve.calls.length === 1, "the first change");
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls.map((call) => [call.method, call.path, call.body])).toEqual([
    ["PATCH", ME, { user_timezone: "Asia/Shanghai" }],
  ]);
  return { nerve, store, first, second };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("UserStore.updateCurrentUser", () => {
  it("sends a change once nerve has answered the one before it", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(json(200, accountIn("Asia/Shanghai")));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.value).toEqual(accountIn("Asia/Shanghai"));
    expect(store.data).toEqual(accountIn("Asia/Shanghai"));
    expect(nerve.calls[1]?.body).toEqual({ user_timezone: "Europe/Berlin" });
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(second.value).toEqual(accountIn("Europe/Berlin"));
    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });

  it("sends the next change when nerve refuses the one before it, which leaves the account as it was", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.answer(problem(500, "internal_error"));
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.data).toBeUndefined();
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });

  it("sends the next change when the one before it gets no answer", async () => {
    const { nerve, store, first, second } = await twoChanges();

    nerve.calls[0]?.fail();
    await until(() => nerve.calls.length === 2, "the second change");
    expect(first.error).toBeInstanceOf(TypeError);
    nerve.calls[1]?.answer(json(200, accountIn("Europe/Berlin")));
    await until(() => second.settled, "the second answer");

    expect(store.data).toEqual(accountIn("Europe/Berlin"));
  });
});

// The deactivation ends the session it was sent in, read as it is sent: the tab may follow another tab's sign-in while
// it is out (M3 design 7.1). Its answer says whether it ended it, which the dialog follows.
describe("UserStore.deactivateAccount", () => {
  it.each([
    { answeredIn: "x", ended: true, when: "the tab's record is still that session's" },
    { answeredIn: "y", ended: false, when: "another tab has moved this one to another account" },
  ])("ends the session it was sent in, and resolves $ended when $when", async ({ answeredIn, ended }) => {
    tab.state.loginId = "x";
    tab.endSession.mockReset();
    tab.endSession.mockResolvedValue(ended);
    const nerve = new FakeNerve();
    const deactivated = track(new UserStore(fakeRoot({}), nerve.client()).deactivateAccount());
    await until(() => nerve.calls.length === 1, "the deactivation");
    expect([nerve.calls[0]?.method, nerve.calls[0]?.path]).toEqual(["POST", "/api/v0/me/deactivate"]);
    // the session the tab is in when nerve answers: still x, or y once another tab has moved it; the store ends x
    tab.state.loginId = answeredIn;
    nerve.calls[0]?.answer(noContent());
    await until(() => deactivated.settled, "the deactivation's answer");

    expect([tab.endSession.mock.calls, deactivated.value]).toEqual([[["x"]], ended]);
  });
});

// The projects the pages offer for a new work item, cycle or module: those of the address's workspace in which the
// caller's role, as the permission store gives it from the project store, is a member's or an admin's.
describe("UserStore, the projects the caller may create in", () => {
  it("gives the address's workspace's projects in which he is a member or an admin, each with his role", async () => {
    const acme = workspaceOf("acme", { role: 15 });
    const beta = workspaceOf("beta", { role: 20 });
    const pa = projectOf("PA", acme.id, { member_role: 20 });
    const pm = projectOf("PM", acme.id, { member_role: 15 });
    const pg = projectOf("PG", acme.id, { member_role: 5 });
    const seen = projectOf("SEEN", acme.id, { member_role: null });
    const lab = projectOf("LAB", beta.id, { member_role: 15 });
    const { api, router, workspaceRoot, projectRoot } = await projectTab(
      { workspace: acme, projects: [pa, pm, pg, seen] },
      { workspace: beta, projects: [lab] }
    );
    const user = new UserStore(fakeRoot({ router, workspaceRoot, projectRoot }), api);
    expect(user.projectsWithCreatePermissions).toEqual({ [pa.id]: 20, [pm.id]: 15 });
    expect(user.canPerformAnyCreateAction).toBe(true);

    // at beta's address, its project, where his role is the admin's (beta's admin); none at an address of no workspace
    router.setQuery({ workspaceSlug: beta.slug });
    expect(user.projectsWithCreatePermissions).toEqual({ [lab.id]: 20 });
    router.setQuery({});
    expect(user.projectsWithCreatePermissions).toEqual({});
    expect(user.canPerformAnyCreateAction).toBe(false);
  });
});
