/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { ProfileSetupStep } from "./root";

// When the onboarding's profile step hands the onboarding on (M3 design 7.1, 7.4): the step renders on the server with
// a stand-in for its form (react-hook-form's useForm), which keeps the step's submit; the test submits names as the
// form would, and nerve answers them when the test says. Its session is fake-tab.ts's. What the step itself says of a
// refusal is M2's (M2 design 7.3).

/** The names the step's form gives its submit. */
type Names = { first_name: string; last_name: string };

const page = vi.hoisted(
  (): { submit: ((names: Names) => Promise<void>) | undefined; updateCurrentUser: ReturnType<typeof vi.fn> } => ({
    submit: undefined,
    updateCurrentUser: vi.fn(),
  })
);
/** The step hands the onboarding on: the root's part. */
const onDone = vi.fn<() => void>();
vi.mock("react-hook-form", () => ({
  useForm: () => ({
    handleSubmit: (submit: (names: Names) => Promise<void>) => {
      page.submit = submit;
      return () => undefined;
    },
    control: {},
    watch: () => "",
    setError: () => undefined,
    formState: { errors: {}, isSubmitting: false, isValid: true },
  }),
  Controller: () => null,
}));
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
  onDone.mockReset();
});

describe("ProfileSetupStep", () => {
  it("sends the names, then hands the onboarding on", async () => {
    await named();
    expect(page.updateCurrentUser.mock.calls).toEqual([[{ first_name: "Ada", last_name: "Lovelace" }]]);
    expect(onDone.mock.calls).toEqual([[]]);
  });

  it("stays when nerve refuses the names", async () => {
    page.updateCurrentUser.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "first_name", code: "required" }])
    );
    await named();
    expect(onDone.mock.calls).toEqual([]);
  });

  it("hands nothing on when nerve has the names after another tab moved this one to another account", async () => {
    const names = heldChange<undefined>();
    page.updateCurrentUser.mockReturnValueOnce(names.sent);
    const submitted = named();
    switchAccount();
    names.answer(undefined);
    await submitted;
    expect(onDone.mock.calls).toEqual([]);
  });
});
