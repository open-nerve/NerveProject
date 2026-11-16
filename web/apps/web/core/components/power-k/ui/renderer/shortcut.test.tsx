/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { expect, it, vi } from "vitest";
import { ShortcutRenderer } from "./shortcut";

// The shortcuts' help with no shortcut of the search (P8 spec 2.3) says so by its key, the search its value, set apart
// within the sentence. A text is its key, then the values it was given.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
// The palette's contextual pages, whose titles the help reads, import the session's client and stores.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "help" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));

it("says by its key that no shortcut matches the search, the search its value", () => {
  const markup = renderToStaticMarkup(<ShortcutRenderer searchQuery="zzz" commands={[]} />);
  expect(markup.replace(/<[^>]*>/g, "")).toBe("power_k.no_shortcuts_found zzz");
  expect(markup).toContain('<span class="font-semibold italic">zzz</span>');
});
