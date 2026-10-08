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

/** A toast's title, and its message from what the promise gave. */
type Outcome<T> = { title: string; message?: (outcome: T) => string };

/** The toast of a promise once it settles, as propel's: its success or its error (the loading toast is not kept). */
export function setPromiseToast<T>(promise: Promise<T>, options: { success: Outcome<T>; error: Outcome<unknown> }) {
  promise.then(
    (value) =>
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: options.success.title,
        message: options.success.message?.(value) ?? "",
      }),
    (error: unknown) =>
      setToast({ type: TOAST_TYPE.ERROR, title: options.error.title, message: options.error.message?.(error) ?? "" })
  );
}
