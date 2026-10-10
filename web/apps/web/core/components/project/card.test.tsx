/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
import { emptyShown, shown } from "@/lib/fake-controls";
import { callerInWeb } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { ProjectCard } from "./card";

// What a project's card offers its caller (M3 design 3.4, 9.2). Archived, it offers restoring and deleting the project
// to those nerve lets do it (project.unarchive and project.delete: its admins, and its members who are the workspace's
// admins), in the card and in its menu. Not archived, it links its settings to its admins and members, who read them,
// and to whom nerve lets change them (project.update: its admins, and its members who are the workspace's admins, its
// guests among them); its other guests are told they joined it. The card renders on the server, at the workspace's
// projects, with stand-ins for its menu (fake-controls.ts, which keeps the items it was given) and for its dialogs;
// the caller's permissions are the store's own, over acme and its project web as nerve lists them to him
// (fake-permissions.ts).

const caller = vi.hoisted(() => {
  const held: { permissions?: IUserPermissionStore } = {};
  return held;
});
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => caller.permissions }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("./delete-project-modal", () => ({ DeleteProjectModal: () => null }));
vi.mock("./join-project-modal", () => ({ JoinProjectModal: () => null }));
vi.mock("./archive-restore-modal", () => ({ ArchiveRestoreProjectModal: () => null }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/**
 * Signs the caller in as nerve lists acme and web to him, his role in each (null: he sees web, no member of it), and
 * shows him web's card, archived or not: what it offers him, in the card and in its menu.
 */
async function offeredTo(workspaceRole: WorkspaceRole, projectRole: ProjectRole | null, archived: boolean) {
  const { permissions, acme, web } = await callerInWeb(workspaceRole, projectRole);
  caller.permissions = permissions;
  emptyShown();
  const project = { ...web, archived_at: archived ? "2026-10-02T09:00:00Z" : null };
  const card = renderToStaticMarkup(
    <MemoryRouter initialEntries={[`/${acme.slug}/projects`]}>
      <Routes>
        <Route path=":workspaceSlug/projects" element={<ProjectCard project={project} />} />
      </Routes>
    </MemoryRouter>
  );
  const inCard = {
    restore: />Restore</.test(card),
    delete: card.includes('aria-label="Delete"'),
    settings: card.includes(`href="/${acme.slug}/settings/projects/${web.id}"`),
  };
  const menu = shown.menus.flatMap(({ items }) => items.filter((item) => item.shouldRender).map((item) => item.key));
  return { inCard, menu };
}

type Caller = { who: string; workspaceRole: WorkspaceRole; projectRole: ProjectRole | null };
const its = {
  admin: { who: "its admin", workspaceRole: 15, projectRole: 20 },
  member: { who: "its member", workspaceRole: 15, projectRole: 15 },
  guest: { who: "its guest", workspaceRole: 5, projectRole: 5 },
  memberAdmin: { who: "its member who is the workspace's admin", workspaceRole: 20, projectRole: 15 },
  guestAdmin: { who: "its guest who is the workspace's admin", workspaceRole: 20, projectRole: 5 },
  outsider: { who: "the workspace's admin who is not its member", workspaceRole: 20, projectRole: null },
} satisfies Record<string, Caller>;

describe("a project's card", () => {
  it.each<Caller & { offered: boolean }>([
    { ...its.admin, offered: true },
    { ...its.memberAdmin, offered: true },
    { ...its.guestAdmin, offered: true },
    { ...its.member, offered: false },
    { ...its.guest, offered: false },
    { ...its.outsider, offered: false },
  ])("archived, offers $who restoring and deleting it when nerve lets him", async (row) => {
    const { inCard, menu } = await offeredTo(row.workspaceRole, row.projectRole, true);
    const offered = row.offered ? ["restore", "delete"] : [];
    expect([inCard.restore, inCard.delete, menu]).toEqual([row.offered, row.offered, offered]);
  });

  it.each<Caller & { linked: boolean }>([
    { ...its.admin, linked: true },
    { ...its.memberAdmin, linked: true },
    { ...its.guestAdmin, linked: true },
    { ...its.member, linked: true },
    { ...its.guest, linked: false },
    { ...its.outsider, linked: false },
  ])(
    "not archived, links its settings to its members and whom nerve lets change them, not its guests: $who",
    async (row) => {
      const { inCard, menu } = await offeredTo(row.workspaceRole, row.projectRole, false);
      expect([inCard.settings, menu.includes("settings")]).toEqual([row.linked, row.linked]);
    }
  );
});
