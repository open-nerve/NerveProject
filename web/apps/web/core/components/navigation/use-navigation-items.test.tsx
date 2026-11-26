/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { useNavigationItems } from "./use-navigation-items";

// The tab bar's intake tab, and the sidebar's that reads the same items (P5a spec R21, M6-M7-07): offered by the intake
// entry rule (intakeEntryOn), as the command palette offers it, so to the project's member and its guest, and to none
// who has no role in it. The caller's role is the permission store's own (fake-permissions.ts), web the project
// store's, over what nerve lists in fake time.

const held = vi.hoisted(() => ({ permissions: undefined as IUserPermissionStore | undefined }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

/** The keys of the tabs the bar offers the caller on web's page, web being as fields say (its intake on). */
async function tabsOf(caller: Caller, fields: Partial<Project> = {}): Promise<string[]> {
  const { permissions, projects, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole, false, {
    intake_view: true,
    ...fields,
  });
  held.permissions = permissions;
  let keys: string[] = [];
  function Consumer() {
    keys = useNavigationItems({
      workspaceSlug: acme.slug,
      projectId: web.id,
      project: projects.getProjectById(web.id),
      allowPermissions: permissions.allowPermissions,
    }).map((item) => item.key);
    return null;
  }
  renderToStaticMarkup(<Consumer />);
  return keys;
}

it.each([its.member, its.guest])("offers the intake tab to $who of a project whose intake is on", async (caller) => {
  expect(await tabsOf(caller)).toContain("intake");
});

it("offers no intake tab to the workspace's admin who has no role in the project, though its intake is on", async () => {
  expect(await tabsOf(its.outsider)).not.toContain("intake");
});

it("offers no intake tab while the project's intake is off, to its member", async () => {
  expect(await tabsOf(its.member, { intake_view: false })).not.toContain("intake");
});

// The tab bar's views tab (P6a spec 2.2, R14; M6-M7-07): offered by the views entry rule (viewEntryOn), as the sidebar
// and the command palette offer it, so to the project's member and its guest, and to none who has no role in it.
it.each([its.member, its.guest])("offers the views tab to $who of a project whose views are on", async (caller) => {
  expect(await tabsOf(caller)).toContain("views");
});

it("offers no views tab to the workspace's admin who has no role in the project, though its views are on", async () => {
  expect(await tabsOf(its.outsider)).not.toContain("views");
});

it("offers no views tab while the project's views are off, to its member", async () => {
  expect(await tabsOf(its.member, { issue_views_view: false })).not.toContain("views");
});
