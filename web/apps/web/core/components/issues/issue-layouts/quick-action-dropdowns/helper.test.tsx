/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, expect, it, vi } from "vitest";
import type { TContextMenuItem } from "@nerve/ui";
import { setLanguage } from "@/lib/fake-i18n-locales";
import { toasts } from "@/lib/fake-toast";
import { issueOf } from "@/store/issue/fake-issues";
import { useArchivedIssueMenuItems, useProjectIssueMenuItems } from "./helper";

// A work item's menus speak the page's language (P8 spec 2.3): the archive's restore is named by its key, and an item
// whose caller gave it no action says so by a toast of keys that names the action by its own text. The texts are the
// locales' own, in both languages: English in Chinese fails here.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-locales"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "menus" } }, publicClient: {} }));

beforeEach(() => {
  toasts.length = 0;
});

const issue = { ...issueOf("one"), archived_at: "2026-10-30T00:00:00Z" };
/** A menu's item of a key. */
const itemOf = (menu: TContextMenuItem[], key: string) => menu.find((entry) => entry.key === key)!;

/** The items of the archive's menu and of the project's, in a language, each allowed and given no action. */
function menus(language: "en" | "zh-CN") {
  setLanguage(language);
  const items: { archived: TContextMenuItem[]; project: TContextMenuItem[] } = { archived: [], project: [] };
  function Menus() {
    items.archived = useArchivedIssueMenuItems({
      issue,
      isDeletingAllowed: false,
      isRestoringAllowed: true,
      setDeleteIssueModal: () => {},
    });
    items.project = useProjectIssueMenuItems({
      issue,
      isEditingAllowed: false,
      isDeletingAllowed: false,
      isArchivingAllowed: true,
      isInArchivableGroup: true,
      openDialog: () => {},
      setDeleteIssueModal: () => {},
    });
    return null;
  }
  renderToStaticMarkup(<Menus />);
  return { restore: itemOf(items.archived, "restore"), archive: itemOf(items.project, "archive") };
}

it.each([
  ["en", "Restore", "Action not available", "Restore action is not implemented.", "Archive action is not implemented."],
  ["zh-CN", "恢复", "操作不可用", "“恢复”操作尚未实现。", "“归档”操作尚未实现。"],
] as const)(
  "in %s the archive's menu names its restore, and an item given no action says so, naming it",
  (language, restore, title, restoreMessage, archiveMessage) => {
    const items = menus(language);
    expect(items.restore.title).toBe(restore);
    items.restore.action();
    items.archive.action();
    expect(toasts).toEqual([
      { type: "error", title, message: restoreMessage },
      { type: "error", title, message: archiveMessage },
    ]);
  }
);
