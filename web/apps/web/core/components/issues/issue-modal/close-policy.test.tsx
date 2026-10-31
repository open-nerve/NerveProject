// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState } from "react";
import { beforeEach, expect, it, vi } from "vitest";
import { close, create, formUI, host, mountDialog, useFormFixture, writes } from "./fake-form";
import { ParentIssuesListModal } from "../parent-issues-list-modal";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";

// This task exercises the actual nested consumer and Dialog; T6 replaces its legacy search boundary.
vi.mock("@/services/project", () => ({
  ProjectService: class {
    async projectIssuesSearch() {
      return [];
    }
  },
}));
function ParentField() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        Choose parent
      </button>
      <ParentIssuesListModal isOpen={open} handleClose={() => setOpen(false)} projectId="A" onChange={() => {}} />
    </>
  );
}
useFormFixture({ realDialogs: true });
beforeEach(() => {
  formUI.extra = ParentField;
});
const dialogs = () => [...document.querySelectorAll<HTMLElement>('[role="dialog"][aria-modal="true"]')];
const button = (name: string, root: ParentNode = document) =>
  [...root.querySelectorAll<HTMLButtonElement>("button")].find((item) => item.textContent?.trim() === name)!;
const settle = () => act(() => new Promise<void>((resolve) => setTimeout(resolve, 0)));
const click = async (name: string, root?: ParentNode) => {
  await act(async () => button(name, root).click());
  await settle();
};
async function openForm() {
  await mountDialog();
  await act(async () => button("Open form").focus());
  await click("Open form");
  expect(dialogs()).toHaveLength(1);
}
async function rename() {
  await act(async () => {
    const input = document.querySelector<HTMLInputElement>("#name")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "Edited");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
async function escape(target = document.activeElement as HTMLElement) {
  await act(async () =>
    target.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  await settle();
}
it.each([false, true])("innerEscapeDoesNotOpenDiscard or close the outer form (dirty=%s)", async (dirty) => {
  await openForm();
  if (dirty) await rename();
  const trigger = button("Choose parent");
  await act(async () => trigger.focus());
  await click("Choose parent");
  expect(dialogs()).toHaveLength(2);
  await escape(dialogs().at(-1)!.querySelector("input")!);
  expect(dialogs()).toHaveLength(1);
  expect(document.body.textContent).not.toContain("Save this draft?");
  expect(document.body.textContent).not.toContain("issue.draft.save_prompt.title");
  expect(close).not.toHaveBeenCalled();
  expect(document.activeElement).toBe(trigger);
});
it("innerEscapeLeavesDirtyOuterOpen and returns to the discard trigger", async () => {
  await openForm();
  await rename();
  const discard = button("discard");
  await act(async () => discard.focus());
  await click("discard");
  expect(dialogs()).toHaveLength(2);
  const inner = dialogs().at(-1)!;
  const title = document.getElementById(inner.getAttribute("aria-labelledby") ?? "");
  expect(title?.textContent).toBe("issue.draft.save_prompt.title");
  await act(async () => button("cancel", inner).focus());
  await escape(button("cancel", inner));
  // Unlike the parent selector, this confirmation stays mounted while its exit transition finishes.
  await vi.waitFor(async () => {
    await settle();
    expect(dialogs()).toHaveLength(1);
  });
  expect(close).not.toHaveBeenCalled();
  expect(document.querySelector<HTMLInputElement>("#name")!.value).toBe("Edited");
  expect(document.activeElement).toBe(discard);
});
it("pendingCannotCloseOrResubmit, retains a refused form and closes only after a new confirmation", async () => {
  await openForm();
  await act(async () => {
    const form = document.querySelector("#name")!.closest("form")!;
    form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
  await vi.waitFor(() => expect(writes()).toHaveLength(1));
  await escape(dialogs()[0]);
  expect(create).toHaveBeenCalledOnce();
  expect(close).not.toHaveBeenCalled();
  expect(dialogs()).toHaveLength(1);
  await act(async () => writes()[0]!.answer(problem(503, "unavailable")));
  expect(dialogs()).toHaveLength(1);
  await click("save");
  await vi.waitFor(() => expect(writes()).toHaveLength(2));
  await act(async () => writes()[1]!.answer(json(201, issueDetailOf("created"))));
  await settle();
  expect(close).toHaveBeenCalledOnce();
  expect(dialogs()).toHaveLength(0);
});
it("returnsFocusToConnectedTrigger after outer Escape, and readiness blocks the same close decision", async () => {
  await openForm();
  formUI.editorReady = false;
  await escape(document.querySelector("#name")! as HTMLElement);
  await click("discard");
  expect(close).not.toHaveBeenCalled();
  expect(dialogs()).toHaveLength(1);
  formUI.editorReady = true;
  await escape(document.querySelector("#name")! as HTMLElement);
  expect(close).toHaveBeenCalledOnce();
  expect(dialogs()).toHaveLength(0);
  expect(document.activeElement).toBe(button("Open form", host));
});
