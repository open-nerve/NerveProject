/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import type { ProjectToggleField } from "@/store/project/project.store";
import { ProjectSettingsMemberDefaults } from "./project-settings-member-defaults";

// What the project's member defaults send (v0 design 7.7: a change whose body depends on what the store holds is
// built in its turn), and what the page says of them (M3 design 7.1): the page renders on the server with stand-ins
// for the member selects and the guests' switch (fake-controls.ts), which keep the props they were given, and for the
// project store, whose changes nerve has not answered yet unless the test says. Its session is fake-tab.ts's.

type Select = { onChange: (value: string) => void };
const page = vi.hoisted(() => {
  const selects: Select[] = [];
  // nerve has not answered: a change stays out, unless a test answers it
  const updateProject = vi.fn((_projectId: string, _data: ProjectUpdate): Promise<unknown> => new Promise(() => {}));
  const toggleProject = vi.fn(
    (_projectId: string, _field: ProjectToggleField): Promise<unknown> => new Promise(() => {})
  );
  return { selects, updateProject, toggleProject };
});
const web: Project = projectOf("WEB", "w-acme", { project_lead_id: "u-bob", default_assignee_id: "u-cat" });
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: () => web,
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
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

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
  signedIn();
  page.updateProject.mockClear();
  page.toggleProject.mockClear();
  toasts.length = 0;
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

  it("says a change is made, and nerve's reason for refusing one", async () => {
    page.updateProject.mockResolvedValueOnce(web);
    page.updateProject.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "project_lead_id", code: "not_allowed" }])
    );
    const { lead } = render();
    lead.onChange("u-ann");
    lead.onChange("u-gus");
    await vi.waitFor(() => expect(toasts).toHaveLength(2));
    expect(toasts).toEqual([
      { type: "success", title: "success!", message: "project_settings.general.toast.success" },
      { type: "error", title: "toast.error", message: "errors.validation_failed" },
    ]);
  });

  it.each(lateSettlings)("says nothing when a change $settles after another tab moved this one", async ({ settle }) => {
    const change = heldChange<undefined>();
    page.toggleProject.mockReturnValueOnce(change.sent);
    const { guests } = render();
    const turned = guests.onCheckedChange(true);
    switchAccount();
    settle(change);
    await turned;
    expect(toasts).toEqual([]);
  });
});
