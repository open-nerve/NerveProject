// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState } from "react";
import { beforeEach, expect, it, vi } from "vitest";
import { close, create, formUI, host, tab, mountDialog, useFormFixture, writes } from "./fake-form";
import { ParentIssuesListModal } from "../parent-issues-list-modal";
import { draftOf, issueDetailOf } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";

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
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/workspaces/ws/issue-search") call.answer(json(200, { data: [] }));
  };
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
async function fill(selector: string, value: string) {
  await act(async () => {
    const input = document.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector)!;
    const proto = input.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const rename = () => fill("#name", "Edited");
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
it("innerEscapeOnlyClosesDraftPrompt and returns to the discard trigger", async () => {
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

it("saveToDraftsFromCreateKeepsFormOnRefusal: empty title bypasses publish validation and one confirmed draft closes the form", async () => {
  await openForm();
  await fill("#name", "");
  await fill('textarea[aria-label="description"]', "<p>Unfinished</p>");
  const saveDraft = button("save_to_drafts");
  expect(saveDraft).toBeDefined();
  await act(async () => {
    saveDraft.click();
    saveDraft.click();
  });
  await vi.waitFor(() => expect(writes()).toHaveLength(1));
  expect(writes()[0]).toMatchObject({
    method: "POST",
    path: "/api/v0/workspaces/ws/drafts",
    body: {
      project_id: "A",
      payload: { name: "", description_html: "<p>Unfinished</p>" },
    },
  });
  expect(writes()[0]!.body).toEqual({
    project_id: "A",
    payload: {
      name: "",
      description_html: "<p>Unfinished</p>",
      priority: "none",
      assignee_ids: [],
      label_ids: [],
    },
  });
  expect(writes()[0]!.body).not.toHaveProperty("skip_activity");
  expect(writes()[0]!.body).not.toHaveProperty("is_draft");
  await escape(dialogs()[0]);
  expect(close).not.toHaveBeenCalled();
  await act(async () =>
    writes()[0]!.answer(
      json(422, { status: 422, code: "validation_failed", errors: [{ field: "name", code: "too_long" }] })
    )
  );
  expect(dialogs()).toHaveLength(1);
  expect(document.querySelector<HTMLInputElement>("#name")!.value).toBe("");
  expect(document.querySelector<HTMLTextAreaElement>('[aria-label="description"]')!.value).toBe("<p>Unfinished</p>");
  expect(document.querySelector("#name")!.getAttribute("aria-invalid")).toBe("true");
  expect(document.activeElement).toBe(document.querySelector("#name"));
  await click("save_to_drafts");
  await vi.waitFor(() => expect(writes()).toHaveLength(2));
  await act(async () =>
    writes()[1]!.answer(
      json(201, { ...draftOf("new-draft"), payload: { name: "", description_html: "<p>Unfinished</p>" } })
    )
  );
  expect(close).toHaveBeenCalledOnce();
  expect(dialogs()).toHaveLength(0);
  expect(writes()).toHaveLength(2);
  expect(create).not.toHaveBeenCalled();
});

it("draft saving from a create form checks the actual editor readiness before sending", async () => {
  await openForm();
  expect(button("save_to_drafts")).toBeDefined();
  formUI.editorReady = false;
  await click("save_to_drafts");
  expect(writes()).toHaveLength(0);
  expect(close).not.toHaveBeenCalled();
  expect(dialogs()).toHaveLength(1);
});

it("saving on close finishes the create form even when create more was selected", async () => {
  await openForm();
  await act(async () => document.querySelector<HTMLElement>('[role="switch"][aria-label="create_more"]')!.click());
  await rename();
  await click("discard");
  expect(dialogs()).toHaveLength(2);
  await click("save_to_drafts", dialogs().at(-1)!);
  await vi.waitFor(() => expect(writes()).toHaveLength(1));
  await act(async () => writes()[0]!.answer(json(201, { ...draftOf("saved"), payload: { name: "Edited" } })));
  expect(close).toHaveBeenCalledOnce();
  expect(dialogs()).toHaveLength(0);
});

it("a workspace guest cannot save from either the direct action or close prompt", async () => {
  await openForm();
  await rename();
  await act(async () => {
    const previous = tab.nerve.respond;
    tab.nerve.respond = (call) => {
      if (call.path === "/api/v0/workspaces") call.answer(json(200, { data: [{ ...tab.workspace, role: 5 }] }));
      else previous?.(call);
    };
    await tab.app.workspaceRoot.fetchWorkspaces();
    tab.nerve.respond = previous;
  });
  expect(button("save_to_drafts")).toBeUndefined();
  await click("discard");
  expect(dialogs()).toHaveLength(0);
  expect(close).toHaveBeenCalledOnce();
  expect(writes()).toHaveLength(0);
});
