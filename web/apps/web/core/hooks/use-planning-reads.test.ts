/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";
import type { IProjectStore } from "@/store/project/project.store";
import { its, permissionsOver, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { planningOn } from "./planning-switch";

// The one gate of what a page of a project's cycles or modules reads (P6a spec 2.3, R3): the project is readable to a
// caller who has a role in it, a guest too, while it is not archived; its list of a domain is read while it is
// readable and the domain is turned on (cycle_view, module_view: planningOn). The caller's role is the permission
// store's own (fake-permissions.ts), the project the project store's, each read as the pages read them (their hooks).

const held = vi.hoisted(() => ({
  permissions: undefined as IUserPermissionStore | undefined,
  projects: undefined as IProjectStore | undefined,
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({ getProjectById: (id: string | undefined) => held.projects?.getProjectById(id) }),
}));

const { usePlanningReads } = await import("./use-planning-reads");

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

/**
 * The caller of a role in acme and in web, web being as fields say, as nerve lists it (an archived one in acme's
 * archive): his gate of web's cycles and of its modules. The project store holds web, with his role in it.
 */
async function gateOf(caller: Caller, fields: Partial<Project> = {}) {
  const acme = workspaceOf("acme", { role: caller.workspaceRole });
  const web = projectOf("WEB", acme.id, {
    member_role: caller.projectRole,
    guest_view_all_features: caller.guestViewAll ?? false,
    ...fields,
  });
  const { nerve, projects, permissions } = await permissionsOver([acme], []);
  await (web.archived_at === null ? loadProjects : loadArchivedProjects)(nerve, projects, acme, [web]);
  Object.assign(held, { permissions, projects });
  expect(projects.getProjectById(web.id)).toMatchObject({
    archived_at: web.archived_at,
    member_role: caller.projectRole,
  });
  return {
    cycles: usePlanningReads(acme.slug, web.id, "cycles"),
    modules: usePlanningReads(acme.slug, web.id, "modules"),
  };
}

describe("usePlanningReads", () => {
  it.each([its.member, its.guest, its.admin, its.guestAdmin])(
    "reads the project's cycles and modules for $who, the project not archived",
    async (caller) => {
      expect(await gateOf(caller)).toEqual({
        cycles: { readable: true, listed: true },
        modules: { readable: true, listed: true },
      });
    }
  );

  it.each<{ when: string; caller: Caller; fields: Partial<Project> }>([
    { when: "the caller sees it and is no member", caller: its.outsider, fields: {} },
    {
      when: "it is archived, though he is its member",
      caller: its.member,
      fields: { archived_at: "2026-11-01T00:00:00Z" },
    },
    {
      when: "it is archived, though he is its guest",
      caller: its.guest,
      fields: { archived_at: "2026-11-01T00:00:00Z" },
    },
  ])("reads nothing of the project when $when, its cycles and modules on", async ({ caller, fields }) => {
    expect(await gateOf(caller, fields)).toEqual({
      cycles: { readable: false, listed: false },
      modules: { readable: false, listed: false },
    });
  });

  it("reads nothing of a project of another of his workspaces than the address's", async () => {
    const acme = workspaceOf("acme");
    const beta = workspaceOf("beta");
    const web = projectOf("WEB", acme.id);
    const { permissions, projects } = await permissionsOver([acme, beta], [web]);
    Object.assign(held, { permissions, projects });
    expect(usePlanningReads(acme.slug, web.id, "cycles")).toEqual({ readable: true, listed: true });
    expect(usePlanningReads(beta.slug, web.id, "cycles")).toEqual({ readable: false, listed: false });
    expect(usePlanningReads(beta.slug, web.id, "modules")).toEqual({ readable: false, listed: false });
    expect(usePlanningReads(undefined, web.id, "modules")).toEqual({ readable: false, listed: false });
    expect(usePlanningReads(acme.slug, "p-gone", "modules")).toEqual({ readable: false, listed: false });
  });

  describe.each([its.member, its.guest])("for $who", (caller) => {
    it.each<{ when: string; fields: Partial<Project>; cycles: boolean; modules: boolean }>([
      { when: "its cycles alone are off", fields: { cycle_view: false }, cycles: false, modules: true },
      { when: "its modules alone are off", fields: { module_view: false }, cycles: true, modules: false },
      { when: "both are off", fields: { cycle_view: false, module_view: false }, cycles: false, modules: false },
    ])("lists a domain while its own switch is on: when $when", async ({ fields, cycles, modules }) => {
      expect(await gateOf(caller, fields)).toEqual({
        cycles: { readable: true, listed: cycles },
        modules: { readable: true, listed: modules },
      });
    });
  });
});

describe("planningOn", () => {
  it.each<{ cycle_view: boolean; module_view: boolean }>([
    { cycle_view: true, module_view: true },
    { cycle_view: true, module_view: false },
    { cycle_view: false, module_view: true },
    { cycle_view: false, module_view: false },
  ])("says each domain's own switch: cycle_view $cycle_view, module_view $module_view", (switches) => {
    expect(planningOn(switches, "cycles")).toBe(switches.cycle_view);
    expect(planningOn(switches, "modules")).toBe(switches.module_view);
  });
});
