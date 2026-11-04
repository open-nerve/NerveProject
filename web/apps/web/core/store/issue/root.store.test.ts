/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { configure, runInAction } from "mobx";
import { expect, it, vi } from "vitest";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { RootStore } from "@/store/root.store";
import { accountOf } from "@/store/member/fake-users";
import { seedAccount } from "@/store/user/fake-account";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

it("computedScopeClearsEmptyCatalogs and the signed-out user", async () => {
  vi.useFakeTimers();
  const nerve = new FakeNerve();
  let app = new RootStore(nerve.client());
  await seedAccount(nerve, app.user, { ...accountOf("user"), id: "user" });
  runInAction(() => {
    app.memberRoot.memberMap = { user: { id: "user" } as never };
    app.router.setQuery({ workspaceSlug: "ws", projectId: "A", userId: "user" });
  });
  expect(app.issue.currentUserId).toBe("user");
  expect(app.issue.memberMap?.user).toBeDefined();
  expect(app.issue.projectId).toBe("A");
  // Signing out creates a new root; the retired account is never mutated into another session's state.
  app.dispose();
  app = new RootStore(nerve.client(), app);
  runInAction(() => {
    app.memberRoot.memberMap = {};
    app.router.setQuery({});
  });
  expect(app.issue.currentUserId).toBeUndefined();
  expect(app.issue.memberMap).toEqual({});
  expect(app.issue.projectId).toBeUndefined();
  expect(app.issue.stateMap).toEqual({});
  expect(app.issue.labelMap).toEqual({});
  app.dispose();
  vi.useRealTimers();
});

it("rootConstructionMakesNoStrictModeWarning", () => {
  configure({ enforceActions: "always" });
  const warn = vi.spyOn(console, "warn");
  const app = new RootStore(new FakeNerve().client());
  app.router.setQuery({ workspaceSlug: "ws", projectId: "A" });
  expect(warn.mock.calls.filter(([message]) => String(message).includes("[MobX]"))).toEqual([]);
  app.dispose();
  warn.mockRestore();
});

it("retire releases permission observation and does not advance old identities", async () => {
  const { issueStoreTab } = await import("./fake-issues");
  const { loadProjects } = await import("@/store/project/fake-projects");
  vi.useFakeTimers();
  const tab = await issueStoreTab();
  try {
    const before = tab.app.issue.permissionReads.identity(tab.workspace.id, [tab.project.id]);
    tab.app.dispose();
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    expect(tab.app.issue.permissionReads.identity(tab.workspace.id, [tab.project.id])).toBe(before);
  } finally {
    tab.app.dispose();
    vi.useRealTimers();
  }
});

it("updates the live calendar preference but stops observing it when the issue root retires", () => {
  const app = new RootStore(new FakeNerve().client());
  const calendar = app.issue.issueCalendarView;
  try {
    const before = calendar.calendarPayload;
    runInAction(() => {
      app.user.userProfile.data = { start_of_the_week: 1 } as NonNullable<typeof app.user.userProfile.data>;
    });
    expect(calendar.calendarPayload).not.toBe(before);
    const start = calendar.getStartAndEndDate("week")!.startDate;
    expect(new Date(`${start}T12:00:00`).getDay()).toBe(1);
    app.issue.retire();
    const retired = calendar.calendarPayload;
    runInAction(() => {
      app.user.userProfile.data = { start_of_the_week: 0 } as NonNullable<typeof app.user.userProfile.data>;
    });
    expect(calendar.calendarPayload).toBe(retired);
  } finally {
    app.dispose();
  }
});
