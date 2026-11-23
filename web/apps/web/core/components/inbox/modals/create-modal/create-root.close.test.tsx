// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The create modal's close rule (P5b spec 2.1, R4, R28, R29): Escape, the backdrop and Discard go through one rule.
import {
  answer,
  button,
  click,
  createModal,
  dialogs,
  escape,
  nameField,
  nameOf,
  openModal,
  pressBackdrop,
  settled,
  submission,
  type,
  useCreateModalFixture,
} from "./fake-create-modal";
import { act } from "react";
import { expect, it } from "vitest";
import { problem } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";

useCreateModalFixture();

const description = () => document.querySelector<HTMLTextAreaElement>('textarea[aria-label="description"]')!;
const createMore = () => document.querySelector<HTMLElement>('[role="switch"][aria-label="create_more"]')!;
const focusName = () => act(async () => nameField().focus());
/** The discard's confirmation, open over the form: the open dialog its title names. */
const confirmation = () => dialogs().find((dialog) => nameOf(dialog) === "inbox_issue.modals.discard.title");
/** Closes the form by close, which asks first; cancels, back to the form. */
async function askedThenCancelled(close: () => Promise<void>) {
  await close();
  expect(confirmation()).toBeDefined();
  await click("common.cancel", confirmation());
  expect(dialogs()).toHaveLength(1);
}

it("while the submission is out, neither Escape, the backdrop nor Discard closes it, and Escape leaves the focus in the name (R28)", async () => {
  await openModal();
  await type(nameField(), "Held");
  await focusName();
  await act(async () => nameField().form!.requestSubmit());
  const held = await submission();
  expect(button("discard")!.disabled).toBe(true);
  expect(createMore().hasAttribute("data-disabled")).toBe(true);
  await escape(nameField());
  expect(dialogs()).toHaveLength(1);
  expect(document.activeElement).toBe(nameField());
  await pressBackdrop();
  await click("discard");
  expect(dialogs()).toHaveLength(1);
  expect(createModal.close).not.toHaveBeenCalled();
  // refused, the form is the user's again: Escape asks, the name being there
  await answer(held, problem(503, "server_busy"));
  await escape(nameField());
  expect(confirmation()).toBeDefined();
  expect(createModal.close).not.toHaveBeenCalled();
});

it("closes an empty form while the editor is not built yet (its ref null), by Escape and by Discard", async () => {
  createModal.editor = "null";
  await openModal();
  await click("discard");
  expect(createModal.close).toHaveBeenCalledTimes(1);
  expect(dialogs()).toHaveLength(0);
  await click("add_work_item");
  await escape(nameField());
  expect(createModal.close).toHaveBeenCalledTimes(2);
  expect(dialogs()).toHaveLength(0);
  expect(toasts).toEqual([]);
});

it("asks before it discards a name while the editor is not built yet", async () => {
  createModal.editor = "null";
  await openModal();
  await type(nameField(), "Unsaved");
  await escape(nameField());
  expect(confirmation()).toBeDefined();
  expect(createModal.close).not.toHaveBeenCalled();
});

it("does not close while the editor built is not ready to discard: it says so", async () => {
  createModal.editor = "unready";
  await openModal();
  await escape(nameField());
  expect(dialogs()).toHaveLength(1);
  await click("discard");
  expect(dialogs()).toHaveLength(1);
  await pressBackdrop();
  expect(dialogs()).toHaveLength(1);
  expect(createModal.close).not.toHaveBeenCalled();
  const said = { type: "error", title: "toast.error", message: "editor_is_not_ready_to_discard_changes" };
  expect(toasts).toEqual([said, said, said]);
});

it.each([
  ["name", () => type(nameField(), "Unsaved")],
  ["description", () => type(description(), "<p>Unsaved</p>")],
  ["priority", () => click("priority")],
  ["assignees", () => click("assignees")],
  ["labels", () => click("labels")],
  ["due date", () => click("due date")],
])("asks before it discards a changed %s, by Escape, the backdrop and Discard", async (_what, change) => {
  await openModal();
  await change();
  await askedThenCancelled(() => escape(nameField()));
  await askedThenCancelled(pressBackdrop);
  await askedThenCancelled(() => click("discard"));
  expect(createModal.close).not.toHaveBeenCalled();
});

it.each([
  ["a name of spaces only", () => type(nameField(), "   "), false],
  [
    "the empty paragraph the editor writes as it is built",
    async () => expect(description().value).toBe("<p></p>"),
    true,
  ],
])("closes without asking with %s", async (_what, change, emptyParagraph) => {
  createModal.emptyParagraph = emptyParagraph;
  await openModal();
  await change();
  await escape(nameField());
  expect(dialogs()).toHaveLength(0);
  expect(createModal.close).toHaveBeenCalledTimes(1);
});

it("the backdrop closes an unchanged form", async () => {
  await openModal();
  await pressBackdrop();
  expect(dialogs()).toHaveLength(0);
  expect(createModal.close).toHaveBeenCalledTimes(1);
});

it("the confirmation is named by its title, its buttons Discard and Cancel; Discard closes both and the focus goes back to Add work item", async () => {
  await openModal();
  await type(nameField(), "Unsaved");
  await act(async () => button("discard")!.focus());
  await click("discard");
  expect(dialogs()).toHaveLength(2);
  const inner = confirmation();
  expect(inner?.textContent).toContain("inbox_issue.modals.discard.content");
  expect([...inner!.querySelectorAll("button")].map((item) => item.textContent?.trim())).toEqual([
    "common.cancel",
    "discard",
  ]);
  await click("discard", inner);
  await settled();
  expect(createModal.close).toHaveBeenCalledTimes(1);
  expect(dialogs()).toHaveLength(0);
  expect(document.activeElement).toBe(button("add_work_item"));
});

it("Cancel and the confirmation's Escape close the confirmation alone: the form stays, its name kept, the focus back where it was", async () => {
  await openModal();
  await type(nameField(), "Unsaved");
  await act(async () => button("discard")!.focus());
  await click("discard");
  expect(confirmation()).toBeDefined();
  await click("common.cancel", confirmation());
  expect(dialogs()).toHaveLength(1);
  expect(document.activeElement).toBe(button("discard"));
  await click("discard");
  expect(confirmation()).toBeDefined();
  await escape(button("common.cancel", confirmation()));
  await settled();
  expect(dialogs()).toHaveLength(1);
  expect(nameField().value).toBe("Unsaved");
  expect(document.activeElement).toBe(button("discard"));
  expect(createModal.close).not.toHaveBeenCalled();
});

it("an Escape that ends an input method's composition neither closes nor asks", async () => {
  await openModal();
  await escape(nameField(), true);
  expect(dialogs()).toHaveLength(1);
  await type(nameField(), "名字");
  await escape(nameField(), true);
  expect(dialogs()).toHaveLength(1);
  expect(createModal.close).not.toHaveBeenCalled();
});

it("opens anew from the initial values: what was discarded is gone", async () => {
  await openModal();
  await type(nameField(), "Discarded");
  await click("priority");
  await escape(nameField());
  expect(confirmation()).toBeDefined();
  await click("discard", confirmation());
  await settled();
  expect(dialogs()).toHaveLength(0);
  await click("add_work_item");
  expect(nameField().value).toBe("");
  await escape(nameField());
  expect(dialogs()).toHaveLength(0);
  expect(createModal.close).toHaveBeenCalledTimes(2);
});
