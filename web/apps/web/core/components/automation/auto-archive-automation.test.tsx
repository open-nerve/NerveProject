/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { emptyShown, shown } from "@/lib/fake-controls";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { AutoArchiveAutomation } from "./auto-archive-automation";

// What the auto-archiving's controls ask the project store (v0 design 7.7: a change whose body depends on what the
// store holds is built in its turn): the component renders on the server for web, whose closed work items are
// archived after 3 months, with stand-ins for its switch and its select of months (fake-controls.ts), which keep the
// props they were given, and for the project store, whose changes record their arguments.

const web: Project = projectOf("WEB", "w-acme", { archive_in: 3 });
const store = vi.hoisted(() => ({
  updateProject: vi.fn((_projectId: string, _data: ProjectUpdate): Promise<unknown> => Promise.resolve()),
  toggleAutoArchive: vi.fn((_projectId: string): Promise<unknown> => Promise.resolve()),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    currentProjectDetails: web,
    updateProject: store.updateProject,
    toggleAutoArchive: store.toggleAutoArchive,
  }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ allowPermissions: () => true }) }));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("@/components/automation", () => ({ SelectMonthModal: () => null }));
vi.mock("@makeplane/propel/components/switch", () => import("@/lib/fake-controls"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders web's auto-archiving: gives its switch and its select of months. */
function render() {
  emptyShown();
  renderToStaticMarkup(<AutoArchiveAutomation />);
  const [toggle] = shown.switches;
  const [months] = shown.selects;
  if (!toggle || !months) throw new Error("the auto-archiving showed no switch or no select");
  return { toggle, months };
}

beforeEach(() => {
  store.updateProject.mockClear();
  store.toggleAutoArchive.mockClear();
  toasts.length = 0;
});

describe("AutoArchiveAutomation", () => {
  it("turns the auto-archiving with the store's toggle, from nerve's last answer, not from what the switch shows", async () => {
    const { toggle } = render();
    await toggle.onCheckedChange(false);
    expect(store.toggleAutoArchive.mock.calls).toEqual([[web.id]]);
    expect(store.updateProject).not.toHaveBeenCalled();
  });

  it("sends the months picked", () => {
    const { months } = render();
    months.onChange(6);
    expect(store.updateProject.mock.calls).toEqual([[web.id, { archive_in: 6 }]]);
    expect(store.toggleAutoArchive).not.toHaveBeenCalled();
  });

  it("shows a toast when nerve refuses the turn", async () => {
    store.toggleAutoArchive.mockImplementationOnce(() => Promise.reject(new Error("refused")));
    const { toggle } = render();
    await toggle.onCheckedChange(false);
    expect(toasts).toEqual([{ type: "error", title: "Error!", message: "Something went wrong. Please try again." }]);
  });
});
