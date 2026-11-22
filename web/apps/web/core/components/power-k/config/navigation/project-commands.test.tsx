/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import type { IProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { usePowerKProjectNavigationCommandsRecord } from "./project-commands";

// The command palette's way into a project's intake ("gk", P5a spec R21, M6-M7-07): offered by the intake entry rule
// (intakeEntryOn), as the sidebar and the tab bar offer it, so to the project's member and its guest, and to none who
// has no role in it. Which pages offer every navigation command is commands.test.tsx's table; it has no member page.

const held = vi.hoisted(() => ({
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as IProjectStore | undefined,
}));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { id: "me" }, permission: held.permissions }) }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => held.projects }));

let intake: TPowerKCommandConfig;
function Consumer() {
  intake = usePowerKProjectNavigationCommandsRecord().nav_project_intake;
  return null;
}
beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

/** The caller on web's page, web's intake on: the palette's context there, the command read. */
async function on(caller: Caller): Promise<TPowerKContext> {
  const { permissions, projects, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole, false, {
    intake_view: true,
  });
  Object.assign(held, { permissions, projects });
  renderToStaticMarkup(<Consumer />);
  return {
    params: { workspaceSlug: acme.slug, projectId: web.id },
    activeCommand: null,
    activeContext: null,
    shouldShowContextBasedActions: true,
    setShouldShowContextBasedActions: () => {},
    navigate: () => {},
    closePalette: () => {},
    setActiveCommand: () => {},
    setActivePage: () => {},
  };
}

it.each([its.member, its.guest])("offers the way into the intake to $who, shown as enabled", async (caller) => {
  const context = await on(caller);
  expect([intake.isVisible(context), intake.isEnabled(context)]).toEqual([true, true]);
});

it("offers none to the workspace's admin who has no role in the project, though its intake is on", async () => {
  const context = await on(its.outsider);
  expect([intake.isVisible(context), intake.isEnabled(context)]).toEqual([false, false]);
});
