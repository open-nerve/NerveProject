// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { profilePage } from "@/components/assets/fake-profile-page";
import { problem, json, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { ProfileSetupStep } from "@/components/onboarding/steps/profile/root";
import type { RootStore } from "@/store/root.store";
import { GeneralProfileSettingsForm } from "./form";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "profile-page" },
    subscribe: () => () => {},
  },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
let tab: Awaited<ReturnType<typeof profilePage>>;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await profilePage();
  toasts.length = 0;
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const settings = () => tab.render(<GeneralProfileSettingsForm user={tab.account()} />);

it("independently uploads and removes avatar and cover; neither write contains the other's field or unsaved names", async () => {
  await settings();
  const upload = async (
    kind: "avatar" | "cover",
    field: "avatar_asset_id" | "cover_image_asset_id",
    purpose: string
  ) => {
    await tab.click(`image_upload.change_${kind}`);
    await tab.uploaded();
    await tab.closed();
    const id = `asset-${tab.puts.length}`;
    expect(tab.writes().at(-1)?.body).toEqual({ [field]: id });
    expect(tab.nerve.to("/api/v0/assets").at(-1)?.body).toMatchObject({ purpose });
    expect(tab.app.user.data?.[field]).toBe(id);
    await act(async () => until(() => !!tab.container.querySelector(`img[src="/static/${id}"]`), "new picture"));
  };
  await upload("avatar", "avatar_asset_id", "USER_AVATAR");
  await upload("cover", "cover_image_asset_id", "USER_COVER");
  const remove = async (kind: "avatar" | "cover", field: "avatar_asset_id" | "cover_image_asset_id") => {
    await tab.click(`image_upload.change_${kind}`);
    await tab.click("image_upload.remove");
    await tab.closed();
    expect(tab.writes().at(-1)?.body).toEqual({ [field]: null });
    expect(tab.app.user.data?.[field]).toBeNull();
  };
  await remove("avatar", "avatar_asset_id");
  await remove("cover", "cover_image_asset_id");
  expect(tab.puts).toHaveLength(2);
  expect(tab.container.querySelector("img[src^='/static/']")).toBeNull();
});

it("a refused owner write retains the completed asset and retries only PATCH, with no names saved", async () => {
  let refusals = 1;
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/me" && call.method === "PATCH" && refusals-- > 0)
      call.answer(problem(500, "internal_error"));
    else tab.answer(call);
  };
  await settings();
  await tab.click("image_upload.change_avatar");
  await tab.uploaded();
  await act(async () => until(() => !!document.querySelector('[role="alert"]'), "bind refusal"));
  expect(tab.app.user.data?.avatar_asset_id).toBeNull();
  await tab.click("image_upload.save");
  await tab.closed();
  expect(tab.writes().map((call) => call.body)).toEqual([
    { avatar_asset_id: "asset-1" },
    { avatar_asset_id: "asset-1" },
  ]);
  expect(tab.puts).toHaveLength(1);
  expect(tab.nerve.to("/api/v0/assets")).toHaveLength(1);
  expect(tab.nerve.to("/api/v0/assets/asset-1/complete")).toHaveLength(1);
});

it("onboarding image upload never submits the names or advances the step", async () => {
  const done = vi.fn();
  await tab.render(<ProfileSetupStep onDone={done} />);
  await tab.click("image_upload.change_avatar");
  await tab.uploaded();
  await tab.closed();
  expect(tab.writes().map((call) => call.body)).toEqual([{ avatar_asset_id: "asset-1" }]);
  expect(done).not.toHaveBeenCalled();
  await act(async () =>
    tab.container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  await act(async () => until(() => done.mock.calls.length === 1, "names complete"));
  expect(tab.writes().at(-1)?.body).toEqual({
    first_name: tab.account().first_name,
    last_name: tab.account().last_name,
  });
  expect(tab.app.user.data?.avatar_asset_id).toBe("asset-1");
});

it("explicit picture retry after a failed refresh is only a current-user read, never another upload or bind", async () => {
  await settings();
  await tab.click("image_upload.change_avatar");
  await tab.uploaded();
  await tab.closed();
  let refuse = true;
  tab.nerve.respond = (call) => {
    if (refuse && call.path === "/api/v0/me" && call.method === "GET") call.answer(problem(403, "forbidden"));
    else tab.answer(call);
  };
  const image = tab.container.querySelector('img[src="/static/asset-1"]');
  expect(image).not.toBeNull();
  await act(async () => image!.dispatchEvent(new Event("error")));
  expect(tab.container.querySelector('img[src="/static/asset-1"]')).toBeNull();
  const before = tab.nerve.to("/api/v0/me").length;
  refuse = false;
  await tab.click("image_upload.retry_image");
  await act(async () => until(() => !!tab.container.querySelector('img[src="/static/asset-1"]'), "retried picture"));
  expect(tab.nerve.to("/api/v0/me")).toHaveLength(before + 1);
  expect(tab.puts).toHaveLength(1);
  expect(tab.writes().map((call) => call.body)).toEqual([{ avatar_asset_id: "asset-1" }]);
});

it.each([false, true])(
  "a retired account's late image bind cannot toast, advance or move focus; refused=%s",
  async (refused) => {
    const done = vi.fn();
    let held: Call | undefined;
    tab.nerve.respond = (call) => {
      if (call.path === "/api/v0/me" && call.method === "PATCH") held = call;
      else tab.answer(call);
    };
    await tab.render(<ProfileSetupStep onDone={done} />);
    await tab.click("image_upload.change_avatar");
    await tab.uploaded();
    await act(async () => until(() => !!held, "binding"));
    await act(async () => tab.app.dispose());
    await tab.render(<button type="button">New account</button>);
    const next = tab.button("New account");
    next.focus();
    await act(async () =>
      held!.answer(
        refused ? problem(500, "internal_error") : json(200, { ...tab.account(), avatar_asset_id: "asset-1" })
      )
    );
    expect(done).not.toHaveBeenCalled();
    expect(toasts).toEqual([]);
    expect(document.activeElement).toBe(next);
    expect(tab.revoke).toHaveBeenCalledExactlyOnceWith("blob:selection-1");
  }
);
