/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for @nerve/i18n, for the tests that read what a page puts into a text (a name, an address): a test file
// mocks that module with this one, vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values")). A text is its key,
// then the values it was given, each after a space. fake-i18n.ts gives the key alone.

export function useTranslation() {
  return { t: (key: string, values: Record<string, unknown> = {}) => [key, ...Object.values(values)].join(" ") };
}
