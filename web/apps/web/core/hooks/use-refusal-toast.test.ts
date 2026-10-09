/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { useRefusalToast } from "./use-refusal-toast";

// How a page says why a change failed (M2 design 7.3): the hook runs as a plain function; its toasts are
// fake-toast.ts's, its texts their keys (fake-i18n.ts).

vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
  toasts.length = 0;
});

describe("useRefusalToast", () => {
  it.each([
    {
      failure: "nerve's refusal, by its problem's code",
      error: refusal(409, "workspace.slug_taken"),
      message: "errors.workspace_slug_taken",
    },
    { failure: "a failure without an answer", error: new TypeError("offline"), message: "errors.unknown" },
  ])("says $failure in a toast under the usual title", ({ error, message }) => {
    useRefusalToast()(error);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message }]);
  });

  it("says it under the title it is given", () => {
    useRefusalToast("workspace_settings.settings.general.delete_modal.error_title")(refusal(403, "forbidden"));
    expect(toasts).toEqual([
      {
        type: "error",
        title: "workspace_settings.settings.general.delete_modal.error_title",
        message: "errors.forbidden",
      },
    ]);
  });
});
