/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { projectDetailsOf, useUpdateProjectDetails, type ProjectDetails } from "./use-update-project-details";

// How a project's general page changes it (M3 design 2 P3, 7.1, 7.6): the hook runs as a plain function, with
// stand-ins for the project store's change and identifier check, which nerve answers when the test says, and for the
// form's setError. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ updateProject: vi.fn(), checkProjectIdentifier: vi.fn(), setError: vi.fn() }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({ updateProject: page.updateProject, checkProjectIdentifier: page.checkProjectIdentifier }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme", { member_role: 20, description: "The site", timezone: "Asia/Shanghai" });
const renamed: ProjectDetails = { ...projectDetailsOf(web), name: "Site", network: 0 };
const update = (details = renamed) => useUpdateProjectDetails()(web, "acme", details, page.setError);

beforeEach(() => {
  signedIn();
  page.updateProject.mockReset();
  page.updateProject.mockResolvedValue(web);
  page.checkProjectIdentifier.mockReset();
  page.checkProjectIdentifier.mockResolvedValue({ available: true });
  page.setError.mockReset();
  toasts.length = 0;
});

describe("useUpdateProjectDetails", () => {
  it("sends the fields the page edits, asks nothing of an identifier unchanged, and says so", async () => {
    await update();
    expect(page.updateProject.mock.calls).toEqual([
      [
        "p-web",
        {
          name: "Site",
          identifier: "WEB",
          description: "The site",
          network: 0,
          logo_props: {},
          timezone: "Asia/Shanghai",
        },
      ],
    ]);
    expect(page.checkProjectIdentifier).not.toHaveBeenCalled();
    expect(toasts).toEqual([
      { type: "success", title: "toast.success", message: "project_settings.general.toast.success" },
    ]);
  });

  it("asks nerve of an identifier changed, then sends it", async () => {
    await update({ ...renamed, identifier: "SITE" });
    expect(page.checkProjectIdentifier.mock.calls).toEqual([["acme", "SITE"]]);
    expect(page.updateProject.mock.calls[0]?.[1]).toMatchObject({ identifier: "SITE" });
  });

  it("says under the identifier that another project has it, and sends nothing", async () => {
    page.checkProjectIdentifier.mockResolvedValueOnce({ available: false });
    await update({ ...renamed, identifier: "OPS" });
    expect(page.updateProject).not.toHaveBeenCalled();
    expect([underFields(page.setError), toasts]).toEqual([[["identifier", "errors.project_identifier_taken"]], []]);
  });

  it("says an identifier the check finds taken as nerve says one it refuses as taken", async () => {
    page.checkProjectIdentifier.mockResolvedValueOnce({ available: false });
    await update({ ...renamed, identifier: "OPS" });
    const checked = underFields(page.setError);
    page.setError.mockReset();
    page.updateProject.mockRejectedValueOnce(refusal(409, "project.identifier_taken"));
    await update();
    expect(underFields(page.setError)).toEqual(checked);
  });

  it.each([
    {
      refused: "a name taken",
      error: refusal(409, "project.name_taken"),
      fields: [["name", "errors.project_name_taken"]],
    },
    {
      refused: "an identifier taken meanwhile",
      error: refusal(409, "project.identifier_taken"),
      fields: [["identifier", "errors.project_identifier_taken"]],
    },
    {
      refused: "a name of characters nerve does not allow",
      error: refusal(422, "validation_failed", [{ field: "name", code: "not_allowed" }]),
      fields: [["name", "errors.field.not_allowed"]],
    },
  ])("shows $refused under its field and says no success", async ({ error, fields }) => {
    page.updateProject.mockRejectedValueOnce(error);
    await update();
    expect([underFields(page.setError), toasts]).toEqual([fields, []]);
  });

  it.each([
    { refused: "a change by one no longer its admin", error: refusal(403, "forbidden"), message: "errors.forbidden" },
    {
      refused: "a change of a project archived meanwhile",
      error: refusal(409, "project.archived"),
      message: "errors.project_archived",
    },
  ])("shows nerve's reason for $refused in a toast", async ({ error, message }) => {
    page.updateProject.mockRejectedValueOnce(error);
    await update();
    expect([underFields(page.setError), toasts]).toEqual([[], [{ type: "error", title: "toast.error", message }]]);
  });

  it("shows nerve's reason in a toast when it refuses the identifier's check, sends nothing and settles", async () => {
    page.checkProjectIdentifier.mockRejectedValueOnce(refusal(403, "forbidden"));
    await update({ ...renamed, identifier: "SITE" });
    expect(page.updateProject).not.toHaveBeenCalled();
    expect([underFields(page.setError), toasts]).toEqual([
      [],
      [{ type: "error", title: "toast.error", message: "errors.forbidden" }],
    ]);
  });

  it.each(lateSettlings)(
    "does nothing on the page when the change $settles after another tab moved this one",
    async ({ settle }) => {
      const answer = heldChange<undefined>();
      page.updateProject.mockReturnValueOnce(answer.sent);
      const updated = update();
      switchAccount();
      settle(answer);
      await updated;
      expect([underFields(page.setError), toasts]).toEqual([[], []]);
    }
  );

  it("says nothing of an identifier nerve answers taken after another tab moved this one", async () => {
    const answer = heldChange<{ available: boolean }>();
    page.checkProjectIdentifier.mockReturnValueOnce(answer.sent);
    const updated = update({ ...renamed, identifier: "OPS" });
    switchAccount();
    answer.answer({ available: false });
    await updated;
    expect([page.updateProject.mock.calls, underFields(page.setError), toasts]).toEqual([[], [], []]);
  });
});
