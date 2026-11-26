/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { describe, expect, it } from "vitest";
import { EUserPermissions } from "@nerve/constants";
import { projectOf } from "@/store/project/fake-projects";
import { viewEntryOn, viewsCreatableBy } from "./views-entry";

// The one rule of the ways into a project's views (P6a spec 2.2, R14; M7 design 3.11; M6-M7-07): its switch is on and
// the caller has a role in the project, an admin's, a member's or a guest's; an archived project is no exception, as
// for the intake. A new view is offered by it to an admin and a member only, as nerve's view.create decides. The
// sidebar, the tab bar and the command palette's two ways in offer the views by the first; create_view by the second.

const T = true,
  F = false;
const roles = [
  ["an admin", EUserPermissions.ADMIN],
  ["a member", EUserPermissions.MEMBER],
  ["a guest", EUserPermissions.GUEST],
  ["one with no role", undefined],
] as const;

/** Web, its views on or off, archived or not; and what each rule gives an admin, a member, a guest, one with no role. */
const table: [string, boolean, boolean, boolean[], boolean[]][] = [
  // web, its views on, archived, viewEntryOn, viewsCreatableBy
  ["web, its views on", T, F, [T, T, T, F], [T, T, F, F]],
  ["web archived, its views on", T, T, [T, T, T, F], [T, T, F, F]],
  ["web, its views off", F, F, [F, F, F, F], [F, F, F, F]],
  ["web archived, its views off", F, T, [F, F, F, F], [F, F, F, F]],
];
const cells = table.flatMap(([web, on, archived, entry, creatable]) =>
  roles.map(([who, role], index) => ({
    web,
    who,
    role,
    project: projectOf("WEB", "id-acme", {
      issue_views_view: on,
      archived_at: archived ? "2026-11-01T00:00:00Z" : null,
    }),
    entry: entry[index],
    creatable: creatable[index],
  }))
);

describe("viewEntryOn", () => {
  it.each(cells)("$web: offers the ways in to $who: $entry", ({ project, role, entry }) => {
    expect(viewEntryOn(project, role)).toBe(entry);
  });

  it("offers none for a project not held, whatever the role", () => {
    for (const [, role] of roles) expect(viewEntryOn(undefined, role)).toBe(false);
  });
});

describe("viewsCreatableBy", () => {
  it.each(cells)("$web: offers a new view to $who: $creatable", ({ project, role, creatable }) => {
    expect(viewsCreatableBy(project, role)).toBe(creatable);
  });

  it("offers none for a project not held, whatever the role", () => {
    for (const [, role] of roles) expect(viewsCreatableBy(undefined, role)).toBe(false);
  });
});
