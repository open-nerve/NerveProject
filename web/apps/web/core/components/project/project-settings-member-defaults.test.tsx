/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { emptyShown, shown } from "@/lib/fake-controls";
import { projectOf } from "@/store/project/fake-projects";
import type { ProjectToggleField } from "@/store/project/project.store";
import { ProjectSettingsMemberDefaults } from "./project-settings-member-defaults";

// What the project's member defaults send (v0 design 7.7: a change whose body depends on what the store holds is
// built in its turn): the page renders on the server with stand-ins for the member selects and the guests' switch
// (fake-controls.ts), which keep the props they were given, and for the project store, whose changes nerve has not
// answered yet.

type Select = { onChange: (value: string) => void };
const page = vi.hoisted(() => {
  const selects: Select[] = [];
  // nerve has not answered: a change stays out
  const updateProject = vi.fn((_projectId: string, _data: ProjectUpdate) => new Promise<never>(() => {}));
  const toggleProject = vi.fn((_projectId: string, _field: ProjectToggleField) => new Promise<never>(() => {}));
  return { selects, updateProject, toggleProject };
});
const web: Project = projectOf("WEB", "w-acme", { project_lead_id: "u-bob", default_assignee_id: "u-cat" });
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    currentProjectDetails: web,
    updateProject: page.updateProject,
    toggleProject: page.toggleProject,
  }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ allowPermissions: () => true }) }));
vi.mock("./member-select", () => ({
  MemberSelect: (props: Select) => {
    page.selects.push(props);
    return null;
  },
}));
vi.mock("@makeplane/propel/components/switch", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the page: gives its two member selects, the lead's then the default assignee's, and its switch. */
function render() {
  page.selects.length = 0;
  emptyShown();
  renderToStaticMarkup(<ProjectSettingsMemberDefaults workspaceSlug="acme" projectId={web.id} />);
  const [lead, assignee] = page.selects;
  const [guests] = shown.switches;
  if (!lead || !assignee || !guests) throw new Error("the page showed no member selects or no switch");
  return { lead, assignee, guests };
}

beforeEach(() => {
  page.updateProject.mockClear();
  page.toggleProject.mockClear();
});

describe("the project's member defaults", () => {
  it("changes the lead, then the default assignee, each alone: nerve keeps the other as it has it", () => {
    const { lead, assignee } = render();
    lead.onChange("u-ann");
    // the lead's change is still out: nerve has not answered it
    assignee.onChange("none");
    expect(page.updateProject.mock.calls).toEqual([
      [web.id, { project_lead_id: "u-ann" }],
      [web.id, { default_assignee_id: null }],
    ]);
  });

  it("turns the guests' view with the store's toggle, from nerve's last answer, not the switch's value", () => {
    const { guests } = render();
    guests.onCheckedChange(true);
    expect(page.toggleProject.mock.calls).toEqual([[web.id, "guest_view_all_features"]]);
    expect(page.updateProject).not.toHaveBeenCalled();
  });
});
