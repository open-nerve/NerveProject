/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import type { Profile } from "@nerve/api-client";
import { ProfileStore } from "@/store/user/profile.store";
import { UserStore } from "@/store/user";
import { fakeRoot } from "@/store/fake-root";
import { accountOf } from "@/store/member/fake-users";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import { usePowerKPreferencesCommands } from "./preferences-commands";
const confirmedProfile: Profile = {
  theme: "dark",
  language: "en",
  start_of_the_week: 0,
  is_onboarded: true,
  is_tour_completed: false,
  last_workspace_id: null,
  updated_at: "2026-10-31T00:00:00Z",
  onboarding_step: { profile_complete: true, workspace_create: true, workspace_invite: true, workspace_join: true },
};
const ui = vi.hoisted(() => ({ setTheme: vi.fn(), reload: vi.fn() }));
vi.mock("next-themes", () => ({ useTheme: () => ({ setTheme: ui.setTheme }) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => user,
  useUserProfile: () => profile,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
let profile: ProfileStore, nerve: FakeNerve, command: TPowerKCommandConfig;
let user: UserStore, commands: TPowerKCommandConfig[];
const ctx: TPowerKContext = {
  params: {},
  activeCommand: null,
  activeContext: null,
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: () => {},
  navigate: () => {},
  closePalette: () => {},
  setActiveCommand: () => {},
  setActivePage: () => {},
};
function Consumer() {
  commands = usePowerKPreferencesCommands();
  command = commands.find((item) => item.id === "update_interface_theme")!;
  return null;
}
beforeEach(() => {
  vi.useFakeTimers();
  signedIn();
  ui.setTheme.mockReset();
  ui.reload.mockReset();
  toasts.length = 0;
  vi.stubGlobal("window", { location: { reload: ui.reload } });
  nerve = new FakeNerve();
  user = new UserStore(fakeRoot({}), nerve.client());
  profile = new ProfileStore(nerve.client());
  renderToStaticMarkup(<Consumer />);
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const choose = async () => {
  if (command.type !== "change-page") throw new Error("theme command must select a value");
  void command.onSelect("dark", ctx);
  await until(() => nerve.calls.length === 1, "theme request");
  expect(nerve.calls[0]).toMatchObject({ method: "PATCH", path: "/api/v0/me/profile", body: { theme: "dark" } });
};
const flush = () => vi.advanceTimersByTimeAsync(0);
it("themeChangesOnlyAfterSameSessionConfirmation", async () => {
  await choose();
  expect(ui.setTheme).not.toHaveBeenCalled();
  expect(ui.reload).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
  nerve.calls[0]!.answer(json(200, confirmedProfile));
  await flush();
  expect(profile.data?.theme).toBe("dark");
  expect(ui.setTheme).toHaveBeenCalledExactlyOnceWith("dark");
  expect(ui.reload).toHaveBeenCalledOnce();
  expect(toasts).toHaveLength(1);
  expect(toasts[0]?.type).toBe("success");
});
it.each([403, 409, 422, 503])(
  "themeRefusalAndLateSuccessDoNotReload: current refusal %s keeps theme and reports code",
  async (status) => {
    await choose();
    nerve.calls[0]!.answer(problem(status, status === 503 ? "server_busy" : "forbidden"));
    await flush();
    expect(ui.setTheme).not.toHaveBeenCalled();
    expect(ui.reload).not.toHaveBeenCalled();
    expect(toasts).toEqual([
      { type: "error", title: "toast.error", message: status === 503 ? "errors.server_busy" : "errors.forbidden" },
    ]);
  }
);
it.each(["success", "refusal"])(
  "a different login generation including same-account ABA ignores old %s",
  async (outcome) => {
    await choose();
    switchAccount();
    switchAccount();
    nerve.calls[0]!.answer(outcome === "success" ? json(200, confirmedProfile) : problem(403, "forbidden"));
    await flush();
    expect(ui.setTheme).not.toHaveBeenCalled();
    expect(ui.reload).not.toHaveBeenCalled();
    expect(toasts).toEqual([]);
  }
);

const preferences = [
  { id: "update_timezone", value: "Asia/Shanghai", path: "/api/v0/me", body: { user_timezone: "Asia/Shanghai" } },
  { id: "update_start_of_week", value: 1, path: "/api/v0/me/profile", body: { start_of_the_week: 1 } },
  { id: "update_interface_language", value: "zh-CN", path: "/api/v0/me/profile", body: { language: "zh-CN" } },
] as const;
for (const preference of preferences) {
  const select = async () => {
    const item = commands.find((candidate) => candidate.id === preference.id)!;
    if (item.type !== "change-page") throw new Error("preference must select a value");
    void item.onSelect(preference.value, ctx);
    await until(() => nerve.calls.length === 1, preference.id);
    expect(nerve.calls[0]).toMatchObject({ method: "PATCH", path: preference.path, body: preference.body });
  };
  const confirmed = () =>
    json(
      200,
      preference.path === "/api/v0/me"
        ? { ...accountOf("Ada"), ...preference.body }
        : { ...confirmedProfile, ...preference.body }
    );
  it(`${preference.id} reports success only after its real store confirms`, async () => {
    await select();
    expect(toasts).toEqual([]);
    nerve.calls[0]!.answer(confirmed());
    await flush();
    expect(toasts).toHaveLength(1);
    expect(toasts[0]?.type).toBe("success");
    expect(ui.reload).not.toHaveBeenCalled();
  });
  it(`${preference.id} preserves the native refusal code`, async () => {
    await select();
    nerve.calls[0]!.answer(problem(503, "server_busy"));
    await flush();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
  });
  it.each(["success", "refusal"])(`${preference.id} suppresses old %s after login ABA`, async (outcome) => {
    await select();
    switchAccount();
    switchAccount();
    nerve.calls[0]!.answer(outcome === "success" ? confirmed() : problem(403, "forbidden"));
    await flush();
    expect(toasts).toEqual([]);
    expect(ui.reload).not.toHaveBeenCalled();
  });
}
