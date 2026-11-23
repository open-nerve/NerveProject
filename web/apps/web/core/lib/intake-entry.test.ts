/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { describe, expect, it } from "vitest";
import { EUserPermissions } from "@nerve/constants";
import { projectOf } from "@/store/project/fake-projects";
import { intakeEntryOn } from "./intake-entry";

// The one rule of the ways into a project's intake (P5a spec R21, M7 design 3.11, M6-M7-07): its switch is on and the
// caller has a role in the project, an admin's, a member's or a guest's; an archived project is no exception, its
// intake is read as ever. The sidebar, the tab bar and the command palette offer the intake by it.

const on = projectOf("WEB", "id-acme", { intake_view: true });
const archived = { ...on, archived_at: "2026-11-01T00:00:00Z" };

describe("intakeEntryOn", () => {
  it.each([
    ["an admin", EUserPermissions.ADMIN],
    ["a member", EUserPermissions.MEMBER],
    ["a guest", EUserPermissions.GUEST],
  ])("offers the intake to %s of a project whose intake is on", (_, role) => {
    expect(intakeEntryOn(on, role)).toBe(true);
  });

  it("offers none to one with no role in the project, nor for a project not held", () => {
    expect(intakeEntryOn(on, undefined)).toBe(false);
    expect(intakeEntryOn(undefined, EUserPermissions.ADMIN)).toBe(false);
  });

  it("offers none while the project's intake is off, whatever the role", () => {
    const off = { ...on, intake_view: false };
    for (const role of [EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST])
      expect(intakeEntryOn(off, role)).toBe(false);
  });

  it("offers it in an archived project as in any other, to an admin and to a guest", () => {
    expect(intakeEntryOn(archived, EUserPermissions.ADMIN)).toBe(true);
    expect(intakeEntryOn(archived, EUserPermissions.GUEST)).toBe(true);
    expect(intakeEntryOn({ ...archived, intake_view: false }, EUserPermissions.ADMIN)).toBe(false);
  });
});
