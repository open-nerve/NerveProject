// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { profilePage } from "@/components/assets/fake-profile-page";
import { ProfileSettingsSidebarHeader } from "@/components/settings/profile/sidebar/header";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { seedAccount } from "@/store/user/fake-account";
import type { RootStore } from "@/store/root.store";
import { UserMenuRoot } from "./user-menu-root";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "menu-images" },
    subscribe: () => () => {},
  },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  // Keep this consumer's menu content mounted; menu keyboard semantics have their own real-DOM tests.
  CustomMenu: Object.assign(
    ({ customButton, children }: { customButton: ReactNode; children: ReactNode }) => (
      <>
        {customButton}
        {children}
      </>
    ),
    { MenuItem: ({ children }: { children: ReactNode }) => <div>{children}</div> }
  ),
}));
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
it("the settings header and user menu share current-user proof, remove all expired pixels, and accept only a fresh read", async () => {
  const user = {
    ...tab.account(),
    avatar_asset_id: "avatar",
    cover_image_asset_id: "cover",
    avatar_url: "/static/avatar",
    cover_image_url: "/static/cover",
    avatar_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
    cover_image_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
  };
  await seedAccount(tab.nerve, tab.app.user, user);
  tab.nerve.respond = (call) => (call.path === "/api/v0/me" ? call.answer(json(200, user)) : tab.answer(call));
  await tab.render(
    <>
      <ProfileSettingsSidebarHeader />
      <UserMenuRoot />
    </>
  );
  expect(tab.nerve.to("/api/v0/me")).toHaveLength(1);
  await act(async () =>
    until(() => tab.container.querySelectorAll("img[src^='/static/']").length === 4, "all four images")
  );
  tab.nerve.respond = undefined;
  await act(async () => vi.advanceTimersByTimeAsync(60_001));
  expect(tab.container.querySelector("img[src^='/static/']")).toBeNull();
  expect(tab.nerve.to("/api/v0/me")).toHaveLength(2);
  await act(async () => tab.app.dispose());
  await act(async () =>
    tab.nerve.to("/api/v0/me")[1]!.answer(
      json(200, {
        ...user,
        avatar_url: "/static/late",
        avatar_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
      })
    )
  );
  expect(tab.container.querySelector("img[src^='/static/']")).toBeNull();
});
