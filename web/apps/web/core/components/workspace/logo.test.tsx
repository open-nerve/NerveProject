// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { profilePage } from "@/components/assets/fake-profile-page";
import { json } from "@/lib/auth/fake-nerve";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import type { RootStore } from "@/store/root.store";
import { WorkspaceLogo } from "./logo";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { start: async () => {}, state: { status: "signed-in", loginId: "logos" }, subscribe: () => () => {} },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof profilePage>>;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await profilePage();
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

it("multiple workspace identities share one directory read and one expiry refresh, each retaining its own proof", async () => {
  const expiry = new Date(Date.now() + 60_000).toISOString();
  const first = { ...tab.workspace, logo_url: "/static/one", logo_url_expires_at: expiry };
  const second = workspaceOf("second", { logo_url: "/static/two", logo_url_expires_at: expiry });
  tab.nerve.respond = undefined;
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [first, second]);
  tab.nerve.calls.length = 0;
  tab.nerve.respond = (call) => call.answer(json(200, { data: [first, second] }));
  await tab.render(
    <>
      <WorkspaceLogo source={{ kind: "workspace", workspaceID: first.id }} name={first.name} />
      <WorkspaceLogo source={{ kind: "workspace", workspaceID: second.id }} name={second.name} />
      <WorkspaceLogo source={{ kind: "workspace", workspaceID: first.id }} name={first.name} />
    </>
  );
  expect(tab.nerve.to("/api/v0/workspaces")).toHaveLength(1);
  expect([...tab.container.querySelectorAll("img")].map((img) => img.getAttribute("src"))).toEqual([
    "/static/one",
    "/static/two",
    "/static/one",
  ]);
  tab.nerve.respond = undefined;
  await act(async () => vi.advanceTimersByTimeAsync(60_001));
  expect(tab.container.querySelector("img")).toBeNull();
  expect(tab.nerve.to("/api/v0/workspaces")).toHaveLength(2);
  await tab.render(<WorkspaceLogo source={{ kind: "workspace", workspaceID: undefined }} name="Unknown" />);
  expect(tab.container.querySelector("img")).toBeNull();
  expect(tab.nerve.to("/api/v0/workspaces")).toHaveLength(2);
});

it("explicit public preview works without a session store and removes its image at expiry or network error", async () => {
  const expiresAt = new Date(Date.now() + 60_000).toISOString();
  const render = (url: string, expiry = expiresAt) =>
    act(async () =>
      tab.renderer.render(<WorkspaceLogo source={{ kind: "public-preview", url, expiresAt: expiry }} name="Public" />)
    );
  await render("/static/public");
  const image = tab.container.querySelector("img");
  expect(image?.getAttribute("src")).toBe("/static/public");
  await act(async () => vi.advanceTimersByTimeAsync(60_001));
  expect(tab.container.querySelector("img")).toBeNull();
  await render("/static/fresh", new Date(Date.now() + 60_000).toISOString());
  const fresh = tab.container.querySelector("img");
  expect(fresh).not.toBeNull();
  await act(async () => fresh?.dispatchEvent(new Event("error")));
  expect(tab.container.querySelector("img")).toBeNull();
  expect(tab.nerve.calls).toHaveLength(0);
});
