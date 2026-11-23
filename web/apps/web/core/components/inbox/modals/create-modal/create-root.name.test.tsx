// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The create modal's name (P5b spec 2.1, R5): the contract's rule (Array.from(name.trim()), 1 to 255), its hint a
// FieldError under the field, a refused submit giving the field the focus; nerve's refusal of the name there too.
import {
  answer,
  button,
  dialogs,
  nameField,
  openModal,
  submission,
  submissions,
  submit,
  type,
  useCreateModalFixture,
  createModal,
} from "./fake-create-modal";
import { act } from "react";
import { expect, it } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";

useCreateModalFixture();

/** What the name field shows of a refusal: whether it is invalid, and the message that describes it. */
const shown = () => {
  const field = nameField();
  return {
    invalid: field.getAttribute("aria-invalid") === "true",
    message: (field.getAttribute("aria-describedby")?.split(/\s+/) ?? [])
      .map((id) => document.getElementById(id)?.textContent ?? "")
      .join(" "),
  };
};
const clear = { invalid: false, message: "" };
/** Submits from the submit button, which has the focus. */
async function submitFromButton() {
  await act(async () => button("create_work_item")!.focus());
  await submit();
}

it("says too long as one types, the field invalid and described by it; 255 is not too long", async () => {
  await openModal();
  await type(nameField(), "x".repeat(256));
  expect(shown()).toEqual({ invalid: true, message: "errors.field.too_long" });
  await type(nameField(), "x".repeat(255));
  expect(shown()).toEqual(clear);
});

it("counts characters, not UTF-16 units: 255 emoji are not too long and are sent", async () => {
  await openModal();
  await type(nameField(), "😀".repeat(255));
  expect(shown()).toEqual(clear);
  await submitFromButton();
  expect((await submission()).body).toMatchObject({ name: "😀".repeat(255) });
});

it("counts the name trimmed: 255 characters with spaces around are not too long and are sent", async () => {
  await openModal();
  await type(nameField(), `  ${"y".repeat(255)}  `);
  expect(shown()).toEqual(clear);
  await submitFromButton();
  expect(submissions()).toHaveLength(1);
});

it("never disables the submit: one too long is refused, the focus given to the name", async () => {
  await openModal();
  await type(nameField(), "z".repeat(256));
  expect(button("create_work_item")!.disabled).toBe(false);
  await submitFromButton();
  expect(submissions()).toEqual([]);
  expect(document.activeElement).toBe(nameField());
  expect(shown()).toEqual({ invalid: true, message: "errors.field.too_long" });
});

it.each(["", "   "])(
  "refuses a blank name (%j): Title is required shows once refused, no request, the focus on the name",
  async (name) => {
    await openModal();
    await type(nameField(), name);
    expect(shown()).toEqual(clear);
    await submitFromButton();
    expect(submissions()).toEqual([]);
    expect(shown()).toEqual({ invalid: true, message: "title_is_required" });
    expect(document.activeElement).toBe(nameField());
    await type(nameField(), "Named");
    expect(shown()).toEqual(clear);
  }
);

it("shows nerve's refusal of the name under it and gives it the focus; no toast", async () => {
  await openModal();
  await type(nameField(), "Refused");
  await submitFromButton();
  await answer(
    await submission(),
    json(422, { status: 422, code: "validation_failed", errors: [{ field: "name", code: "not_allowed", message: "" }] })
  );
  expect(shown()).toEqual({ invalid: true, message: "errors.field.not_allowed" });
  expect(document.activeElement).toBe(nameField());
  expect(toasts).toEqual([]);
  expect(dialogs()).toHaveLength(1);
});

it("says another refusal of nerve's in a toast, the form kept", async () => {
  await openModal();
  await type(nameField(), "Kept");
  await submitFromButton();
  await answer(await submission(), problem(409, "issue.intake_disabled"));
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.issue_intake_disabled" }]);
  expect(shown()).toEqual(clear);
  expect(nameField().value).toBe("Kept");
  expect(dialogs()).toHaveLength(1);
  expect(createModal.close).not.toHaveBeenCalled();
});
