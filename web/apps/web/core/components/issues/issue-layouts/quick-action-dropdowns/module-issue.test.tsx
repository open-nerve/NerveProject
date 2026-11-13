// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// Whom a module page's row offers the removal of its work item from the module (P6b spec 2.7, R9, R15): whoever may
// change the work item (an admin or a member of its project, not a guest), while the row is not read-only and the
// module takes work items (its project's modules on, the module not archived, whatever its status). Each condition
// alone withholds it, whatever the others say. The menu, the layout's command host and the stores are real; the dialog
// is not opened.
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { IssueListCommandHost } from "../command-host";
import { M1, planningPageTab } from "../roots/fake-planning-page";
import { PlanningPageIssueQuickActions } from "./planning-page-issue";
import { listIssue } from "@/store/issue/fake-issues";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, params: {} as Record<string, string> }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));

let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", moduleId: M1 };
});
afterEach(async () => page.close());

/** Whether the row's menu offers the removal, on the page fields say, the row read-only or not. */
async function offersRemoval(fields: Parameters<typeof planningPageTab>[1], readOnly: boolean) {
  page = await planningPageTab(EIssuesStoreType.MODULE, fields);
  bound.app = page.app;
  const host = document.createElement("div");
  await page.render(
    <MemoryRouter>
      <IssueListCommandHost>
        <PlanningPageIssueQuickActions
          issue={listIssue("first", 1000, { module_ids: [M1] })}
          parentRef={{ current: host }}
          handleDelete={async () => {}}
          handleRemoveFromView={async () => {}}
          readOnly={readOnly}
        />
      </IssueListCommandHost>
    </MemoryRouter>
  );
  const items = [...document.querySelectorAll('[data-context-menu="true"] [role="menuitem"]')];
  expect(items.length, "the row's menu").toBeGreaterThan(0);
  return items.some((node) => node.textContent === "issue.remove.module.label");
}

// nerve archives only completed or cancelled modules: being archived alone withholds the removal, its status none
const archived: Partial<Module> = { status: "completed", archived_at: "2026-12-01T09:00:00Z" };
it.each([
  { to: "a member, on a row not read-only of a module in its backlog", fields: {}, readOnly: false, removal: true },
  { to: "an admin, on a row not read-only", fields: { role: 20 as const }, readOnly: false, removal: true },
  {
    to: "a member, on a completed module's row not read-only",
    fields: { module: { status: "completed" as const } },
    readOnly: false,
    removal: true,
  },
  {
    to: "a member, on a cancelled module's row not read-only",
    fields: { module: { status: "cancelled" as const } },
    readOnly: false,
    removal: true,
  },
  { to: "a guest, though the row is not read-only", fields: { role: 5 as const }, readOnly: false, removal: false },
  { to: "a member, on a read-only row", fields: {}, readOnly: true, removal: false },
  {
    to: "a member, on an archived module's row not read-only",
    fields: { module: archived },
    readOnly: false,
    removal: false,
  },
  {
    to: "a member, while the project's modules are off",
    fields: { project: { module_view: false } },
    readOnly: false,
    removal: false,
  },
])("offers the removal from the module to $to: $removal", async ({ fields, readOnly, removal }) => {
  expect(await offersRemoval(fields, readOnly)).toBe(removal);
});
