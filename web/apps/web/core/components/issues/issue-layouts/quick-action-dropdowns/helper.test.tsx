/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { expect, it, vi } from "vitest";
import type { TContextMenuItem } from "@nerve/ui";
import { toasts } from "@/lib/fake-toast";
import { issueOf } from "@/store/issue/fake-issues";
import { useArchivedIssueMenuItems, useProjectIssueMenuItems } from "./helper";

// A work item's menus speak the page's language (P8 spec 2.3): the archive's restore is named by its key, and an item
// whose caller gave it no action says so by a toast of keys, which names the action by its translated name, never by
// an English one. A text is its key, then the values it was given.

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "menus" } }, publicClient: {} }));

const issue = { ...issueOf("one"), archived_at: "2026-10-30T00:00:00Z" };
/** A menu's item of a key. */
const itemOf = (menu: TContextMenuItem[], key: string) => menu.find((entry) => entry.key === key)!;

/** The items of the archive's menu and of the project's, each allowed and given no action. */
function menus() {
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

it("names the archive's restore by its key, and an item given no action says so by keys, naming it translated", () => {
  const items = menus();
  expect(items.restore.title).toBe("common.actions.restore");
  items.restore.action();
  items.archive.action();
  const title = "issue.action_unavailable.title";
  expect(toasts).toEqual([
    { type: "error", title, message: "issue.action_unavailable.message common.actions.restore" },
    { type: "error", title, message: "issue.action_unavailable.message common.actions.archive" },
  ]);
});
