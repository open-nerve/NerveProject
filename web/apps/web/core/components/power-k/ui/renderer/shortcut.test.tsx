/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { expect, it, vi } from "vitest";
import { setLanguage } from "@/lib/fake-i18n-locales";
import { ShortcutRenderer } from "./shortcut";

// The shortcuts' help with no shortcut of the search (P8 spec 2.3) says so in the page's language, the search set
// apart within the sentence. The texts are the locales' own, in both languages.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-locales"));
// The palette's contextual pages, whose titles the help reads, import the session's client and stores.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "help" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));

it.each([
  ["en", 'No shortcuts found for "zzz"'],
  ["zh-CN", "未找到与“zzz”匹配的快捷键"],
] as const)("in %s the help says no shortcut matches the search, naming it", (language, text) => {
  setLanguage(language);
  const markup = renderToStaticMarkup(<ShortcutRenderer searchQuery="zzz" commands={[]} />);
  expect(markup.replace(/<[^>]*>/g, "").replaceAll("&quot;", '"')).toBe(text);
  expect(markup).toContain('<span class="font-semibold italic">zzz</span>');
});
