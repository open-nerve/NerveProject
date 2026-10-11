/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import type { Mock } from "vitest";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, settledYet, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { ProfileSetupStep } from "./root";

// When the onboarding's profile step hands the onboarding on, and what it says of nerve's refusal of the names (M2
// design 7.3, M3 design 7.1, 7.4): the step renders on the server with a stand-in for its form (react-hook-form's
// useForm, fake-form.ts), which keeps the step's submit and what the step says under the name; the test submits names
// as the form would, and nerve answers them when the test says. Its session is fake-tab.ts's.

/** The names the step's form gives its submit. */
type Names = { first_name: string; last_name: string };

const page = vi.hoisted(
  (): {
    submit: ((names: Names) => Promise<void>) | undefined;
    updateCurrentUser: ReturnType<typeof vi.fn>;
    setError: Mock;
  } => ({
    submit: undefined,
    updateCurrentUser: vi.fn(),
    setError: vi.fn(),
  })
);
/** The step hands the onboarding on: the root's part. */
const onDone = vi.fn<() => Promise<void> | undefined>();
vi.mock("react-hook-form", async () => (await import("@/lib/fake-form")).fakeForm(page));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: { display_name: "ada" }, updateCurrentUser: page.updateCurrentUser }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the step, and submits the names as its form would; settles once the step has. */
async function named() {
  renderToStaticMarkup(<ProfileSetupStep onDone={onDone} />);
  if (!page.submit) throw new Error("the step showed no form");
  await page.submit({ first_name: "Ada", last_name: "Lovelace" });
}

beforeEach(() => {
  signedIn();
  page.submit = undefined;
  page.updateCurrentUser.mockReset();
  page.updateCurrentUser.mockResolvedValue(undefined);
  page.setError.mockReset();
  onDone.mockReset();
  toasts.length = 0;
});

describe("ProfileSetupStep", () => {
  it("sends the names, then hands the onboarding on", async () => {
    await named();
    expect(page.updateCurrentUser.mock.calls).toEqual([[{ first_name: "Ada", last_name: "Lovelace" }]]);
    expect(onDone.mock.calls).toEqual([[]]);
  });

  it("stays busy until the root is done with the onboarding it handed on (its end, for one who has a workspace)", async () => {
    const ending = heldChange<undefined>();
    onDone.mockReturnValueOnce(ending.sent);
    const settled = settledYet(named());
    await pageSettled();
    expect([onDone.mock.calls, settled()]).toEqual([[[]], false]);
    ending.answer(undefined);
    await pageSettled();
    expect(settled()).toBe(true);
  });

  it("stays when nerve refuses the names, its refusal of the name said under it", async () => {
    page.updateCurrentUser.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "first_name", code: "required" }])
    );
    await named();
    expect([onDone.mock.calls, underFields(page.setError), toasts]).toEqual([
      [],
      [["first_name", "errors.field.required"]],
      [],
    ]);
  });

  it("says in a toast a refusal that is not of the name", async () => {
    page.updateCurrentUser.mockRejectedValueOnce(refusal(403, "forbidden"));
    await named();
    expect([onDone.mock.calls, underFields(page.setError), toasts.map((toast) => toast.type)]).toEqual([
      [],
      [],
      ["error"],
    ]);
  });

  it.each(lateSettlings)(
    "hands nothing on and says nothing when the names' change $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const names = heldChange<undefined>();
      page.updateCurrentUser.mockReturnValueOnce(names.sent);
      const submitted = named();
      switchAccount();
      settle(names);
      await submitted;
      expect([onDone.mock.calls, page.setError.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
