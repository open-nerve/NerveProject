// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The create modal's form (P5b spec 2.1, R6, R24, R29): the submission and its follow-ups, the dialog's name, the
// "Create more" switch, the natural Tab order, the texts through keys.
import {
  answer,
  click,
  createModal,
  dialogs,
  leavePage,
  nameField,
  nameOf,
  openModal,
  selectedAddress,
  submission,
  submissions,
  submit,
  type,
  useCreateModalFixture,
} from "./fake-create-modal";
import { act } from "react";
import { expect, it } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { detailOf } from "@/store/issue/intake/fake-intake";

useCreateModalFixture();

const createMore = () => document.querySelector<HTMLElement>('[role="switch"][aria-label="create_more"]')!;
const created = detailOf("new", { project_id: "p-a", workspace_id: "id-ws" });
const success = { type: "success", title: "toast.success", message: "issue_created_successfully" };

it.each(["null", "unready"] as const)(
  "sends nothing while the editor is not ready (its ref %s): it says so",
  async (editor) => {
    createModal.editor = editor;
    await openModal();
    await type(nameField(), "Waiting");
    await submit();
    expect(submissions()).toEqual([]);
    expect(toasts).toEqual([
      { type: "error", title: "toast.error", message: "editor_is_not_ready_to_discard_changes" },
    ]);
    expect(dialogs()).toHaveLength(1);
  }
);

it("closes on success without asking, a changed form too: to the new work item, said in a toast", async () => {
  await openModal();
  await type(nameField(), "Created");
  await click("priority");
  await submit();
  await answer(await submission(), json(201, created));
  expect(dialogs()).toHaveLength(0);
  expect(createModal.close).toHaveBeenCalledTimes(1);
  expect(createModal.navigate).toHaveBeenCalledExactlyOnceWith(selectedAddress(created.id));
  expect(toasts).toEqual([success]);
});

it("with Create more on, a success empties the form and keeps it open", async () => {
  await openModal();
  await act(async () => createMore().click());
  await type(nameField(), "First");
  await submit();
  await answer(await submission(), json(201, created));
  expect(dialogs()).toHaveLength(1);
  expect(nameField().value).toBe("");
  expect(createModal.clear).toHaveBeenCalledTimes(1);
  expect(createModal.close).not.toHaveBeenCalled();
  expect(createModal.navigate).not.toHaveBeenCalled();
  expect(toasts).toEqual([success]);
});

it.each([
  ["success", () => json(201, created)],
  ["refusal", () => problem(409, "issue.intake_disabled")],
])("follows a %s only while the form is mounted: once the page is left, nothing (R29)", async (_what, response) => {
  await openModal();
  await type(nameField(), "Late");
  await submit();
  const held = await submission();
  await leavePage();
  await answer(held, response());
  expect(createModal.navigate).not.toHaveBeenCalled();
  expect(createModal.close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
});

it("is named by its title", async () => {
  await openModal();
  expect(nameOf(dialogs()[0]!)).toBe("inbox_issue.modal.title");
});

it("Create more is the switch itself, in no button, out of use while the submission is out", async () => {
  await openModal();
  expect(createMore().closest('[role="button"]')).toBeNull();
  expect(createMore().getAttribute("aria-checked")).toBe("false");
  await act(async () => createMore().click());
  expect(createMore().getAttribute("aria-checked")).toBe("true");
  await type(nameField(), "Held");
  await submit();
  await submission();
  expect(createMore().hasAttribute("data-disabled")).toBe(true);
  await act(async () => createMore().click());
  expect(createMore().getAttribute("aria-checked")).toBe("true");
});

it("its controls follow the document's order (no positive tabIndex), the properties' texts through keys", async () => {
  createModal.realProperties = true;
  await openModal();
  const dialog = dialogs()[0]!;
  const positive = [...dialog.querySelectorAll<HTMLElement>("[tabindex]")].filter((element) => element.tabIndex > 0);
  expect(positive).toEqual([]);
  expect(dialog.textContent).toContain("issue.add.assignee");
  expect(dialog.textContent).toContain("issue.add.due_date");
  expect(dialog.textContent).not.toMatch(/Assignees|Due date/);
});
