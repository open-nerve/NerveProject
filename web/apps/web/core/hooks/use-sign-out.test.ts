/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { toasts } from "@/lib/fake-toast";
import { useSignOut } from "./use-sign-out";

// How a page signs the caller out (M2 design 7.1): the hook runs as a plain function, with a stand-in for the user's
// store, whose sign-out the test makes succeed or fail; its toasts are fake-toast.ts's, its texts their keys.

const user = vi.hoisted(() => ({ signOut: vi.fn() }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => user }));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
  user.signOut.mockReset();
  toasts.length = 0;
});

describe("useSignOut", () => {
  it("signs the caller out, and says nothing", async () => {
    user.signOut.mockResolvedValueOnce(undefined);
    expect(await useSignOut()()).toBe(true);
    expect([user.signOut.mock.calls, toasts]).toEqual([[[]], []]);
  });

  it("says so when nerve's logout fails, and does not reject: the session stays", async () => {
    user.signOut.mockRejectedValueOnce(new TypeError("offline"));
    expect(await useSignOut()()).toBe(false);
    expect(toasts).toEqual([
      { type: "error", title: "auth.sign_out.toast.error.title", message: "auth.sign_out.toast.error.message" },
    ]);
  });
});
