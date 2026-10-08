/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for @nerve/propel/toast, for the tests of the pages that show a toast: a test file mocks that module with
// this one, vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast")), and reads in `toasts` what the page
// showed.

/** A toast as a page shows it. */
type Toast = { type: string; title: string; message: string };

/** Every toast shown so far, in order; a test empties it before each case. */
export const toasts: Toast[] = [];

export const TOAST_TYPE = { SUCCESS: "success", ERROR: "error" };

export function setToast(toast: Toast) {
  toasts.push(toast);
}
