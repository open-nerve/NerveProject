/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { useCreateProject, type ProjectCreationForm } from "./use-create-project";

// How the creation's form creates a project (M3 design 2 P1, 7.1, 7.6): the hook runs as a plain function, with a
// stand-in for the project store's creation, which nerve answers when the test says, and for the form's setError. Its
// session is fake-tab.ts's.

const page = vi.hoisted(() => ({ createProject: vi.fn(), setError: vi.fn() }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ createProject: page.createProject }) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const form: ProjectCreationForm = {
  name: "Web",
  identifier: "WEB",
  description: "The site",
  network: 2,
  logo_props: { in_use: "emoji", emoji: { value: "128640" } },
  project_lead_id: null,
};
const web = projectOf("WEB", "w-acme", { member_role: 20 });
const create = (values = form) => useCreateProject()("acme", values, page.setError);

beforeEach(() => {
  signedIn();
  page.createProject.mockReset();
  page.createProject.mockResolvedValue(web);
  page.setError.mockReset();
  toasts.length = 0;
});

describe("useCreateProject", () => {
  it("creates the project from the form's fields, no lead until one is picked, says so and gives it", async () => {
    expect(await create()).toBe(web);
    expect(page.createProject.mock.calls).toEqual([
      [
        "acme",
        {
          name: "Web",
          identifier: "WEB",
          description: "The site",
          network: 2,
          logo_props: { in_use: "emoji", emoji: { value: "128640" } },
        },
      ],
    ]);
    expect(toasts).toEqual([{ type: "success", title: "success", message: "project_created_successfully" }]);
  });

  it("sends the lead picked", async () => {
    await create({ ...form, project_lead_id: "u-ada" });
    expect(page.createProject.mock.calls[0]?.[1]).toMatchObject({ project_lead_id: "u-ada" });
  });

  it.each([
    {
      refused: "a name taken",
      error: refusal(409, "project.name_taken"),
      fields: [["name", "errors.project_name_taken"]],
    },
    {
      refused: "an identifier taken",
      error: refusal(409, "project.identifier_taken"),
      fields: [["identifier", "errors.project_identifier_taken"]],
    },
    {
      refused: "a name of characters nerve does not allow",
      error: refusal(422, "validation_failed", [{ field: "name", code: "not_allowed" }]),
      fields: [["name", "errors.field.not_allowed"]],
    },
  ])("shows $refused under its field, gives nothing and says no success", async ({ error, fields }) => {
    page.createProject.mockRejectedValueOnce(error);
    expect(await create()).toBeUndefined();
    expect([underFields(page.setError), toasts]).toEqual([fields, []]);
  });

  it.each([
    {
      refused: "a lead nerve does not allow",
      error: refusal(422, "validation_failed", [{ field: "project_lead_id", code: "not_allowed" }]),
      message: "errors.validation_failed",
    },
    { refused: "a guest's creation", error: refusal(403, "forbidden"), message: "errors.forbidden" },
  ])("shows nerve's reason for $refused in a toast", async ({ error, message }) => {
    page.createProject.mockRejectedValueOnce(error);
    expect(await create()).toBeUndefined();
    expect([underFields(page.setError), toasts]).toEqual([[], [{ type: "error", title: "toast.error", message }]]);
  });

  // answered with the project created, which the form must not get once the tab is another account's
  it.each(lateSettlingsAnswering(web))(
    "does nothing on the page and gives nothing when the creation $settles after another tab moved this one",
    async ({ settle }) => {
      const answer = heldChange<Project>();
      page.createProject.mockReturnValueOnce(answer.sent);
      const created = create();
      switchAccount();
      settle(answer);
      expect(await created).toBeUndefined();
      expect([underFields(page.setError), toasts]).toEqual([[], []]);
    }
  );
});
