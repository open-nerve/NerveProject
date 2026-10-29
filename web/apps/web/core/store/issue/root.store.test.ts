/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { configure, runInAction } from "mobx";
import { expect, it, vi } from "vitest";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { RootStore } from "@/store/root.store";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

it("computedScopeClearsEmptyCatalogs and the signed-out user", () => {
  const app = new RootStore(new FakeNerve().client());
  runInAction(() => {
    app.user.data = { id: "user" } as NonNullable<typeof app.user.data>;
    app.memberRoot.memberMap = { user: { id: "user" } as never };
    app.router.setQuery({ workspaceSlug: "ws", projectId: "A", userId: "user" });
  });
  expect(app.issue.currentUserId).toBe("user");
  expect(app.issue.memberMap?.user).toBeDefined();
  expect(app.issue.projectId).toBe("A");
  runInAction(() => {
    app.user.data = undefined;
    app.memberRoot.memberMap = {};
    app.router.setQuery({});
  });
  expect(app.issue.currentUserId).toBeUndefined();
  expect(app.issue.memberMap).toEqual({});
  expect(app.issue.projectId).toBeUndefined();
  expect(app.issue.stateMap).toEqual({});
  expect(app.issue.labelMap).toEqual({});
  app.dispose();
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
