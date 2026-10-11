/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { ProjectSettingLabelGroup } from "./project-setting-label-group";

// A label with labels under it, on the label settings page (M3 design 2 P7, 3.16, 9.2): its row and theirs show the
// controls that change them (a grip, a menu) to those nerve lets change the labels alone (label.update and the others:
// the project's admins, and its members who are the workspace's admins), as the page says by isEditable. The group
// renders on the server with stand-ins for its row and for the rows under it, which say whether they have their
// controls. Its session is fake-tab.ts's; the label store, which it does not read, a stand-in.

vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => ({}) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/components/labels/label-block/label-item-block", () => ({
  LabelItemBlock: ({ label, disabled }: { label: { name: string }; disabled?: boolean }) =>
    `${label.name}${disabled ? "" : " and its controls"};`,
}));
vi.mock("@/components/labels/project-setting-label-item", () => ({
  ProjectSettingLabelItem: ({ label, isEditable }: { label: { name: string }; isEditable: boolean }) =>
    `${label.name}${isEditable ? " and its controls" : ""};`,
}));

const web = projectOf("WEB", "w-acme");
const bug = labelOf(web, "Bug", 10000);
const ui = labelOf(web, "UI", 20000, { parent_id: bug.id });

/** The text the group of Bug, with UI under it, shows, for one who may change the labels or not. */
const shownWith = (isEditable: boolean) =>
  renderToStaticMarkup(
    <ProjectSettingLabelGroup
      label={bug}
      labelChildren={[ui]}
      handleLabelDelete={vi.fn()}
      isUpdating={false}
      setIsUpdating={vi.fn()}
      isLastChild
      onDrop={vi.fn()}
      isEditable={isEditable}
    />
  ).replace(/<[^>]*>/g, "");

describe("ProjectSettingLabelGroup", () => {
  it("shows one who may change the labels the group's controls and those of the labels under it", () => {
    expect(shownWith(true)).toBe("Bug and its controls;UI and its controls;");
  });

  it("shows any other member the group and the labels under it, with no control", () => {
    expect(shownWith(false)).toBe("Bug;UI;");
  });
});
