/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyShown, shown } from "@/lib/fake-controls";
import { projectOf } from "@/store/project/fake-projects";
import { ConfirmProjectMemberRemove } from "./confirm-project-member-remove";
import { JoinProjectModal } from "./join-project-modal";
import { LeaveProjectModal } from "./leave-project-modal";

// The texts of the project's dialogs that no story opens in Chinese (M3 closeout): each renders on the server, open,
// with the stand-in for @nerve/i18n, whose text is its key, and stand-ins for its controls (fake-controls.ts), which
// keep the props they were given: each title, sentence and button is a key of the page's language.

const web = projectOf("WEB", "w-acme", { name: "Web" });
vi.mock("react-router", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({ workspaceSlug: "acme", projectId: web.id }),
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ getProjectById: () => web }) }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { id: "u-admin" } }) }));
vi.mock("./use-join-project", () => ({ useJoinProject: () => vi.fn() }));
vi.mock("@/components/project/settings/use-project-membership-changes", () => ({
  useProjectMembershipChanges: () => ({ leave: vi.fn() }),
}));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
  emptyShown();
});

describe("the project's dialogs", () => {
  it.each<{ dialog: string; open: () => ReactElement; sentences: string[]; buttons: string[] }>([
    {
      dialog: "the joining",
      open: () => <JoinProjectModal isOpen workspaceSlug="acme" project={web} handleClose={() => {}} />,
      sentences: ["project_dialogs.join.title", "project_dialogs.join.body"],
      buttons: ["cancel", "project_dialogs.join.submit"],
    },
    {
      dialog: "the leaving",
      open: () => <LeaveProjectModal isOpen project={web} onClose={() => {}} />,
      sentences: [
        "project_dialogs.leave.title",
        "project_dialogs.leave.body",
        "project_dialogs.type_project_name",
        "project_dialogs.type_confirmation",
      ],
      buttons: ["cancel", "project_dialogs.leave.title"],
    },
    {
      dialog: "a member's removal",
      open: () => (
        <ConfirmProjectMemberRemove
          isOpen
          data={{ id: "u-bob", display_name: "bob" }}
          onSubmit={() => Promise.resolve()}
          onClose={() => {}}
        />
      ),
      sentences: ["project_dialogs.remove_member.remove_title", "project_dialogs.remove_member.remove_body"],
      buttons: ["cancel", "remove"],
    },
    {
      dialog: "one's own leaving, from the members",
      open: () => (
        <ConfirmProjectMemberRemove
          isOpen
          data={{ id: "u-admin", display_name: "admin" }}
          onSubmit={() => Promise.resolve()}
          onClose={() => {}}
        />
      ),
      sentences: ["project_dialogs.remove_member.leave_title", "project_dialogs.remove_member.leave_body"],
      buttons: ["cancel", "leave"],
    },
  ])("$dialog says each of its texts in the page's language", ({ open, sentences, buttons }) => {
    const markup = renderToStaticMarkup(open());
    expect(sentences.filter((key) => !markup.includes(key))).toEqual([]);
    expect(shown.buttons.map((button) => button.children)).toEqual(buttons);
  });
});
