/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for @nerve/i18n, for the tests of the pages that render text: a test file mocks that module with this
// one, vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n")). A text is its key, so a test reads the keys a page
// shows.

export function useTranslation() {
  return { t: (key: string) => key };
}
