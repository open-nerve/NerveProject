/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import type { Mock } from "vitest";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { ProfileSettingsLanguageAndTimezonePreferencesList } from "./language-and-timezone-list";

// What the profile settings' preferences say of how nerve settled a change (M2 design 7.3, M3 design 7.1): the time
// zone, the language and the start of the week (StartOfWeekPreference, which the list renders). The list renders on
// the server with stand-ins for its selects, which keep what each gives the page; the test picks a value as a select
// would, and nerve answers when the test says. Its session is fake-tab.ts's.

const page = vi.hoisted(
  (): {
    timezone: ((value: string) => unknown) | undefined;
    updateCurrentUser: Mock;
    updateUserProfile: Mock;
  } => ({ timezone: undefined, updateCurrentUser: vi.fn(), updateUserProfile: vi.fn() })
);
vi.mock("@/components/global", () => ({
  TimezoneSelect: ({ onChange }: { onChange: (value: string) => unknown }) => {
    page.timezone = onChange;
    return null;
  },
}));
vi.mock("@/components/settings/control-item", () => ({
  SettingsControlItem: ({ control }: { control: ReactNode }) => control,
}));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: undefined, updateCurrentUser: page.updateCurrentUser, userProfile: { data: undefined } }),
  useUserProfile: () => ({ data: undefined, updateUserProfile: page.updateUserProfile }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  ...(await import("@/lib/fake-i18n")),
}));

/** The list's three preferences: how a select picks a value, what the page sends, and what it says of success. */
const preferences = [
  {
    preference: "the time zone",
    pick: () => page.timezone?.("Asia/Shanghai"),
    sent: () => page.updateCurrentUser,
    body: { user_timezone: "Asia/Shanghai" },
    said: "power_k.preferences_actions.toast.timezone.success",
  },
  {
    preference: "the language",
    pick: () => shown.selects[0]?.onChange("zh-CN"),
    sent: () => page.updateUserProfile,
    body: { language: "zh-CN" },
    said: "power_k.preferences_actions.toast.generic.success",
  },
  {
    preference: "the start of the week",
    pick: () => shown.selects[1]?.onChange(1),
    sent: () => page.updateUserProfile,
    body: { start_of_the_week: 1 },
    said: "power_k.preferences_actions.toast.generic.success",
  },
];

beforeEach(() => {
  signedIn();
  emptyShown();
  page.timezone = undefined;
  page.updateCurrentUser.mockReset();
  page.updateCurrentUser.mockResolvedValue(undefined);
  page.updateUserProfile.mockReset();
  page.updateUserProfile.mockResolvedValue(undefined);
  toasts.length = 0;
  renderToStaticMarkup(<ProfileSettingsLanguageAndTimezonePreferencesList />);
});

describe("ProfileSettingsLanguageAndTimezonePreferencesList", () => {
  it.each(preferences)("sends $preference picked, and says nerve made the change", async (row) => {
    row.pick();
    await pageSettled();
    expect(row.sent().mock.calls).toEqual([[row.body]]);
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: row.said }]);
  });

  it.each(preferences)("says nerve's refusal of $preference in a toast", async (row) => {
    row.sent().mockRejectedValueOnce(refusal(403, "forbidden"));
    row.pick();
    await pageSettled();
    expect(toasts.map((toast) => toast.type)).toEqual(["error"]);
  });

  it.each(preferences.flatMap((row) => lateSettlings.map((late) => Object.assign({}, row, late))))(
    "says nothing when the change of $preference $settles after another tab moved this one to another account",
    async (row) => {
      const change = heldChange<undefined>();
      row.sent().mockReturnValueOnce(change.sent);
      row.pick();
      switchAccount();
      row.settle(change);
      await pageSettled();
      expect([row.sent().mock.calls.length, toasts]).toEqual([1, []]);
    }
  );
});
