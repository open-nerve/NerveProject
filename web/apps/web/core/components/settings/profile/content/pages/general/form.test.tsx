/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import type { Mock } from "vitest";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { GeneralProfileSettingsForm } from "./form";

// What the profile settings' general form says of how nerve settled the change of the names (M2 design 7.3, M3 design
// 7.1): the form renders on the server with a stand-in for its form (react-hook-form's useForm, fake-form.ts), which
// keeps the form's submit and what it says under its fields; the test submits the names as the form would, and nerve
// answers them when the test says. Its session is fake-tab.ts's.

/** What the form's fields hold, as it gives its submit. */
type Names = { first_name: string; last_name: string; display_name: string; email: string };

const form = vi.hoisted(
  (): {
    submit: ((names: Names) => Promise<void>) | undefined;
    setError: Mock;
    updateCurrentUser: Mock;
  } => ({ submit: undefined, setError: vi.fn(), updateCurrentUser: vi.fn() })
);
vi.mock("react-hook-form", async () => (await import("@/lib/fake-form")).fakeForm(form));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: undefined, updateCurrentUser: form.updateCurrentUser }),
}));
vi.mock("@/components/account/deactivate-account-modal", () => ({ DeactivateAccountModal: () => null }));
vi.mock("@/components/common/cover-image", () => ({ CoverImage: () => null }));
vi.mock("@/components/settings/boxed-control-item", () => ({ SettingsBoxedControlItem: () => null }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const ada: User = {
  id: "u-ada",
  email: "ada@example.com",
  first_name: "Ada",
  last_name: "Lovelace",
  display_name: "ada",
  user_timezone: "UTC",
  avatar_url: null,
  cover_image_url: null,
  created_at: "2026-01-01T00:00:00Z",
};

/** Renders the form, and submits new names as it would; settles once the form has. */
async function renamed() {
  renderToStaticMarkup(<GeneralProfileSettingsForm user={ada} />);
  if (!form.submit) throw new Error("the page showed no form");
  await form.submit({ first_name: "Augusta", last_name: "King", display_name: "augusta", email: ada.email });
}

beforeEach(() => {
  signedIn();
  form.submit = undefined;
  form.setError.mockReset();
  form.updateCurrentUser.mockReset();
  form.updateCurrentUser.mockResolvedValue(ada);
  toasts.length = 0;
});

describe("GeneralProfileSettingsForm", () => {
  it("sends the names, the address not among them, and says nerve made the change", async () => {
    await renamed();
    expect(form.updateCurrentUser.mock.calls).toEqual([
      [{ first_name: "Augusta", last_name: "King", display_name: "augusta" }],
    ]);
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "profile_updated" }]);
  });

  it("says a refusal under the field it names, and no toast", async () => {
    form.updateCurrentUser.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "display_name", code: "too_long" }])
    );
    await renamed();
    expect([underFields(form.setError), toasts]).toEqual([[["display_name", "errors.field.too_long"]], []]);
  });

  it("says in a toast a refusal that names no field of the form", async () => {
    form.updateCurrentUser.mockRejectedValueOnce(refusal(403, "forbidden"));
    await renamed();
    expect([underFields(form.setError), toasts.map((toast) => toast.type)]).toEqual([[], ["error"]]);
  });

  it.each(lateSettlings)(
    "says nothing when the change $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      form.updateCurrentUser.mockReturnValueOnce(change.sent);
      const submitted = renamed();
      switchAccount();
      settle(change);
      await submitted;
      expect([form.setError.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});
