/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { ProjectSettingsFeatureControlItem } from "@/components/settings/project/content/feature-control-item";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import type { ProjectToggleField } from "@/store/project/project.store";
import { ProjectFeaturesList } from "./features-list";
import { useFeatureToggle } from "./use-feature-toggle";

// What a flip of a feature's switch asks the project store (v0 design 7.7: a change whose body depends on what the
// store holds is built in its turn), and what the page says of it (M3 design 7.1): the hook runs as a plain function,
// and the two pages that flip a feature render on the server, with stand-ins for their switches (fake-controls.ts),
// which keep the props they were given, and for the project store, whose changes record their arguments. Its session
// is fake-tab.ts's.

const web: Project = projectOf("WEB", "w-acme", { cycle_view: true, module_view: false });
const store = vi.hoisted(() => ({
  toggleProject: vi.fn((_projectId: string, _field: ProjectToggleField): Promise<unknown> => Promise.resolve()),
  updateProject: vi.fn((_projectId: string, _data: ProjectUpdate): Promise<unknown> => Promise.resolve()),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: () => web,
    toggleProject: store.toggleProject,
    updateProject: store.updateProject,
  }),
}));
vi.mock("@makeplane/propel/components/switch", () => import("@/lib/fake-controls"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
  signedIn();
  store.toggleProject.mockClear();
  store.updateProject.mockClear();
  toasts.length = 0;
  emptyShown();
});

describe("useFeatureToggle", () => {
  it("turns a feature with the store's toggle, from nerve's last answer: no body made of the project held", () => {
    useFeatureToggle("acme", web.id)("module_view");
    expect(store.toggleProject.mock.calls).toEqual([[web.id, "module_view"]]);
    expect(store.updateProject).not.toHaveBeenCalled();
  });

  it("says a turn is done, and nerve's reason for refusing one", async () => {
    useFeatureToggle("acme", web.id)("module_view");
    store.toggleProject.mockImplementationOnce(() => Promise.reject(refusal(403, "forbidden")));
    useFeatureToggle("acme", web.id)("cycle_view");
    await pageSettled();
    expect(toasts).toEqual([
      { type: "success", title: "Success!", message: "Project feature updated successfully." },
      { type: "error", title: "toast.error", message: "errors.forbidden" },
    ]);
  });

  it.each(lateSettlings)("says nothing when a turn $settles after another tab moved this one", async ({ settle }) => {
    const turn = heldChange<undefined>();
    store.toggleProject.mockReturnValueOnce(turn.sent);
    useFeatureToggle("acme", web.id)("module_view");
    switchAccount();
    settle(turn);
    await pageSettled();
    expect(toasts).toEqual([]);
  });

  it("is what each switch of the features list flips, its own feature", () => {
    renderToStaticMarkup(<ProjectFeaturesList workspaceSlug="acme" projectId={web.id} />);
    for (const toggle of shown.switches) void toggle.onCheckedChange(true);
    expect(store.toggleProject.mock.calls).toEqual([
      [web.id, "cycle_view"],
      [web.id, "module_view"],
      [web.id, "issue_views_view"],
      [web.id, "intake_view"],
    ]);
    expect(store.updateProject).not.toHaveBeenCalled();
  });

  it("is what the switch of a feature's own page flips", () => {
    renderToStaticMarkup(
      <ProjectSettingsFeatureControlItem
        title="Intake"
        featureProperty="intake_view"
        projectId={web.id}
        value={false}
        workspaceSlug="acme"
      />
    );
    for (const toggle of shown.switches) void toggle.onCheckedChange(true);
    expect(store.toggleProject.mock.calls).toEqual([[web.id, "intake_view"]]);
    expect(store.updateProject).not.toHaveBeenCalled();
  });
});
