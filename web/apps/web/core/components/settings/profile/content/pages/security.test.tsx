/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import type { Mock } from "vitest";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import type { KeptForm } from "@/lib/fake-form";
import { focusedUnder, refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { SecurityProfileSettings } from "./security";

// What the security page says of how nerve settled the change of the password (M2 design 7.3, M3 design 7.1): the
// page renders on the server with a stand-in for its form (react-hook-form's useForm, fake-form.ts), which keeps the
// form's submit, what it says under its fields and its emptying; the test submits the passwords as the form would, and
// nerve answers them when the test says. Its session is fake-tab.ts's.

/** What the form's fields hold, as it gives its submit. */
type Passwords = { old_password: string; new_password: string; confirm_password: string };

const form = vi.hoisted((): KeptForm<Passwords> & { reset: Mock; changePassword: Mock } => ({
  submit: undefined,
  setError: vi.fn(),
  reset: vi.fn(),
  changePassword: vi.fn(),
}));
vi.mock("react-hook-form", async () => (await import("@/lib/fake-form")).fakeForm(form));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ changePassword: form.changePassword }) }));
vi.mock("@/components/api-token/token-list", () => ({ ApiTokenList: () => null }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => ({ PasswordStrengthIndicator: () => null }));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the page, and submits the passwords as its form would; settles once the page has. */
async function changed() {
  renderToStaticMarkup(<SecurityProfileSettings />);
  if (!form.submit) throw new Error("the page showed no form");
  await form.submit({ old_password: "old secret", new_password: "new secret", confirm_password: "new secret" });
}

beforeEach(() => {
  signedIn();
  form.submit = undefined;
  form.setError.mockReset();
  form.reset.mockReset();
  form.changePassword.mockReset();
  form.changePassword.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("SecurityProfileSettings", () => {
  it("sends the passwords; once nerve has changed it, empties the form and says so", async () => {
    await changed();
    expect(form.changePassword.mock.calls).toEqual([[{ current_password: "old secret", new_password: "new secret" }]]);
    expect([form.reset.mock.calls.length, toasts.map((toast) => toast.message)]).toEqual([
      1,
      ["auth.common.password.toast.change_password.success.message"],
    ]);
  });

  it.each([
    {
      refused: "the current password, wrong",
      error: refusal(403, "identity.current_password_incorrect"),
      under: [["old_password", "errors.current_password_incorrect"]],
    },
    {
      refused: "the new password",
      error: refusal(422, "validation_failed", [{ field: "new_password", code: "weak_password" }]),
      under: [["new_password", "errors.field.weak_password"]],
    },
  ])("says nerve's refusal of $refused under its field, focused, the form as it was", async ({ error, under }) => {
    form.changePassword.mockRejectedValueOnce(error);
    await changed();
    expect([underFields(form.setError), form.reset.mock.calls, toasts]).toEqual([under, [], []]);
    expect(focusedUnder(form.setError)).toEqual(under.map(([field]) => field));
  });

  it("says in a toast a refusal of neither password", async () => {
    form.changePassword.mockRejectedValueOnce(refusal(403, "forbidden"));
    await changed();
    expect([underFields(form.setError), toasts.map((toast) => toast.type)]).toEqual([[], ["error"]]);
  });

  it.each(lateSettlings)(
    "says nothing, nor empties the form, when the change $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      form.changePassword.mockReturnValueOnce(change.sent);
      const submitted = changed();
      switchAccount();
      settle(change);
      await submitted;
      expect([form.setError.mock.calls, form.reset.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
