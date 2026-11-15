/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7b spec 2.5, R8: the words of a work item list's options that name a project's planning, each its domain's (the
// display properties of the cycle and the modules, the groupings by the cycle and the module, the filters by them),
// offered only while the project is known and has the domain turned on; every other word always.
import { describe, expect, it } from "vitest";
import type { Project } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import {
  listWordOn,
  planningEntryOn,
  planningListRefusal,
  type ListWord,
  type PlanningDomain,
} from "./planning-switch";

const switches = (cycles: boolean, modules: boolean) => ({ cycle_view: cycles, module_view: modules });

it.each<{ word: ListWord; kind: string; domain: PlanningDomain }>([
  { word: "cycle", kind: "the display property and the grouping", domain: "cycles" },
  { word: "modules", kind: "the display property", domain: "modules" },
  { word: "module", kind: "the grouping", domain: "modules" },
  { word: "cycle_id", kind: "the filter", domain: "cycles" },
  { word: "module_id", kind: "the filter", domain: "modules" },
])("offers $word, $kind, only while the project has its $domain on", ({ word, domain }) => {
  const cycles = domain === "cycles";
  expect(listWordOn(switches(true, true), word)).toBe(true);
  expect(listWordOn(switches(cycles, !cycles), word)).toBe(true);
  expect(listWordOn(switches(!cycles, cycles), word)).toBe(false);
  expect(listWordOn(switches(false, false), word)).toBe(false);
  expect(listWordOn(undefined, word)).toBe(false);
});

it.each<ListWord>([
  "state",
  "priority",
  "labels",
  "assignees",
  null,
  "label_id",
  "assignee_id",
  "key",
  "sub_issue_count",
])("offers %s whatever the project has on, and while the project is unknown", (word) => {
  expect(listWordOn(switches(false, false), word)).toBe(true);
  expect(listWordOn(undefined, word)).toBe(true);
});

// M6 P7c spec 2.2, R2: the one rule of the ways into a project's cycles or modules (the command palette's commands, as
// the sidebar declares it): the caller's role in the project is an admin's or a member's (an admin's for the
// workspace's admin, its guest too), the project is not archived and has the domain turned on. And why the domain's
// list is not read, by nerve's problem code, in the order of the read gate (usePlanningReads' listed): no role first,
// then the archive, then the switch; none while it is read, a guest's too. The caller's role is the permission store's
// own (fake-permissions.ts).

type Held = Pick<Project, "archived_at" | "cycle_view" | "module_view">;
const open: Held = { archived_at: null, cycle_view: true, module_view: true };
const archived: Held = { ...open, archived_at: "2026-10-31T00:00:00Z" };
const unknownTo = { cycles: "project.not_found", modules: "project.not_found" };

/** The caller's role in web, as the permission store gives it. */
async function roleOf(caller: Caller) {
  const { permissions, acme, web } = await callerInWeb(caller.workspaceRole, caller.projectRole);
  return permissions.getProjectRoleByWorkspaceSlugAndProjectId(acme.slug, web.id);
}

describe("planningEntryOn", () => {
  it("takes the workspace's admin who is the project's guest for its admin, as the permission store does", async () => {
    expect(await roleOf(its.guestAdmin)).toBe(EUserPermissions.ADMIN);
  });

  it.each<{ when: string; caller: Caller; project: Held | undefined; cycles: boolean; modules: boolean }>([
    { when: "its admin", caller: its.admin, project: open, cycles: true, modules: true },
    { when: "its member", caller: its.member, project: open, cycles: true, modules: true },
    {
      when: "its guest who is the workspace's admin",
      caller: its.guestAdmin,
      project: open,
      cycles: true,
      modules: true,
    },
    { when: "its guest", caller: its.guest, project: open, cycles: false, modules: false },
    {
      when: "the workspace's admin who is not its member",
      caller: its.outsider,
      project: open,
      cycles: false,
      modules: false,
    },
    { when: "its admin, the project archived", caller: its.admin, project: archived, cycles: false, modules: false },
    {
      when: "its admin, its cycles off",
      caller: its.admin,
      project: { ...open, cycle_view: false },
      cycles: false,
      modules: true,
    },
    {
      when: "its admin, its modules off",
      caller: its.admin,
      project: { ...open, module_view: false },
      cycles: true,
      modules: false,
    },
    { when: "its admin, the project not held", caller: its.admin, project: undefined, cycles: false, modules: false },
  ])(
    "offers the ways into the project's domains to $when as the rule says",
    async ({ caller, project, cycles, modules }) => {
      const role = await roleOf(caller);
      expect({
        cycles: planningEntryOn(project, role, "cycles"),
        modules: planningEntryOn(project, role, "modules"),
      }).toEqual({ cycles, modules });
    }
  );
});

describe("planningListRefusal", () => {
  it.each<{ when: string; caller: Caller; project: Held | undefined; cycles?: string; modules?: string }>([
    { when: "the caller has no role in it", caller: its.outsider, project: open, ...unknownTo },
    {
      when: "the caller has no role in it, archived and both off",
      caller: its.outsider,
      project: { ...archived, cycle_view: false, module_view: false },
      ...unknownTo,
    },
    { when: "the project is not held", caller: its.admin, project: undefined, ...unknownTo },
    {
      when: "it is archived",
      caller: its.member,
      project: archived,
      cycles: "project.archived",
      modules: "project.archived",
    },
    {
      when: "it is archived and both are off",
      caller: its.guest,
      project: { ...archived, cycle_view: false, module_view: false },
      cycles: "project.archived",
      modules: "project.archived",
    },
    {
      when: "its cycles are off",
      caller: its.member,
      project: { ...open, cycle_view: false },
      cycles: "planning.cycles_disabled",
    },
    {
      when: "its modules are off",
      caller: its.guest,
      project: { ...open, module_view: false },
      modules: "planning.modules_disabled",
    },
    { when: "the caller is its guest", caller: its.guest, project: open },
    { when: "the caller is its admin", caller: its.admin, project: open },
  ])(
    "says why each domain's list is not read when $when, none for one that is",
    async ({ caller, project, cycles, modules }) => {
      const role = await roleOf(caller);
      expect({
        cycles: planningListRefusal(project, role, "cycles"),
        modules: planningListRefusal(project, role, "modules"),
      }).toEqual({ cycles, modules });
    }
  );
});
