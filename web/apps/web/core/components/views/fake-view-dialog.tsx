/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState } from "react";
import type { Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { expect, vi, type Mock } from "vitest";
import { Where, dialog, settle } from "@/components/common/fake-quick-actions";
import type { ViewContent } from "@/store/views/view-content";
import { CreateUpdateViewModal } from "./modal";

// The view's dialog as a person uses it (P6a spec 2.4), for its tests: a page of web's views with the control that
// opens it, as a new view's ("Add view"), a Save as's with the content it was given, or the change of the view of
// viewId, under a router whose address the test reads (Where); and the dialog's controls, found and used as a person
// does. The views' store is the test file's stand-in; the dialog is real, in a real ModalCore.

/** The project of the page. */
export const WEB = "p-web";

/** What the page's handleClose was called. */
export const closed: Mock<() => void> = vi.fn();

type Opened = { viewId?: string; content?: ViewContent };

/** web's views page, with the control that opens the dialog as opened says. */
function Page(props: Opened) {
  const [shown, setShown] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setShown(true)}>
        opener
      </button>
      <CreateUpdateViewModal
        isOpen={shown}
        handleClose={() => {
          closed();
          setShown(false);
        }}
        workspaceSlug="acme"
        container={{ kind: "project", projectId: WEB }}
        {...props}
      />
    </>
  );
}

/** The page's control that opens the dialog. */
export const opener = () =>
  [...document.querySelectorAll("button")].find((button) => button.textContent === "opener") ?? null;
/** The dialog's field of name (its name, its description). */
export const field = (name: string) =>
  document.querySelector<HTMLInputElement | HTMLTextAreaElement>(`[role="dialog"] [name="${name}"]`)!;
/** The dialog's button whose text is text. */
export const button = (text: string) =>
  [...(dialog()?.querySelectorAll("button") ?? [])].find((item) => item.textContent?.trim() === text)!;
/** The dialog's name: its title's text (aria-labelledby). */
export const title = () => document.getElementById(dialog()?.getAttribute("aria-labelledby") ?? "")?.textContent;

/** Opens the dialog of web's views page, rendered in root, as opened says, from its opener, which has the focus. */
export async function open(root: Root, opened: Opened = {}) {
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={[`/acme/projects/${WEB}/views`]}>
        <Page {...opened} />
        <Where />
      </MemoryRouter>
    )
  );
  await act(async () => {
    opener()?.focus();
    opener()?.click();
  });
  await settle();
  expect(dialog()).not.toBeNull();
}

/** Types value into the dialog's field of name, as a person does. */
export async function fill(name: string, value: string) {
  await act(async () => {
    const input = field(name);
    const proto = input instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

/** Submits the dialog's form, as its submit button or Enter does. */
export async function submit() {
  await act(async () =>
    dialog()!
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  await settle();
}

/** Presses Escape where the focus is; composing, as an input method's Escape that ends its composition. */
export async function escape(composing = false) {
  await act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true, isComposing: composing })
    )
  );
  await settle();
}

/** A click on the dialog's backdrop, outside its panel, as a pointer gives it (its press and its release). */
export async function clickOutside() {
  const backdrop = document.querySelector<HTMLElement>('[class*="bg-backdrop"]');
  expect(backdrop).not.toBeNull();
  await act(async () => {
    for (const type of ["pointerdown", "mousedown", "pointerup", "mouseup"])
      backdrop!.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true }));
    backdrop!.click();
  });
  await settle();
}

/** The dialog's icon picker's button, by its name (the i18n stand-in's key). */
export const iconButton = () => dialog()!.querySelector<HTMLButtonElement>('[aria-label="aria_labels.view_icon"]')!;

/** Opens the dialog's icon picker by its button; gives the picker's panel. */
export async function openIconPicker() {
  await act(async () => iconButton().click());
  await settle();
  const panel = document.querySelector<HTMLElement>('[data-slot="popover-content"]');
  expect(panel).not.toBeNull();
  return panel!;
}
