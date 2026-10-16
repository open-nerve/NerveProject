// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { act } from "react";
import { createPortal } from "react-dom";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ModalCore, ModalTitle } from "./modal-core";

// An Escape in a ModalCore dialog (M3 closeout, C11-3), in a browser's document (jsdom). Headless UI's Dialog hears the
// keydown on the window, so the document has it before the Dialog does, prevented or not.

declare global {
  // React's flag for a test that renders within act
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

let root: Root;
let container: HTMLElement;

/** Lets what keepFocusIn queues run: its focus, a microtask after a loss, and the mutation observers' callbacks. */
async function settled(): Promise<void> {
  await act(() => new Promise<void>((resolve) => setTimeout(resolve, 0)));
}

/** Opens a dialog that holds a field and a button, closed by handleClose when it is given. */
async function open(handleClose?: () => void): Promise<void> {
  await act(async () => {
    root.render(
      <ModalCore isOpen handleClose={handleClose}>
        <ModalTitle>Invite people</ModalTitle>
        <input aria-label="Email" />
        <button type="button" aria-label="Send">
          Send
        </button>
      </ModalCore>
    );
  });
  await settled();
}

/** The open dialog's element, Headless UI's: the one keepFocusIn gives the focus to. */
function dialog(): HTMLElement {
  const shown = document.querySelector<HTMLElement>('[role="dialog"]');
  if (!shown) throw new Error("no dialog is open");
  return shown;
}

/** The control of the open dialog named name. */
function control<T extends HTMLElement>(name: string): T {
  const named = dialog().querySelector<T>(`[aria-label="${name}"]`);
  if (!named) throw new Error(`no control named ${name}`);
  return named;
}

/** Presses Escape on target, and resolves with whether its keydown reached the document prevented. */
async function escape(target: HTMLElement): Promise<boolean> {
  let prevented = false;
  const seen = (event: KeyboardEvent) => {
    prevented = event.defaultPrevented;
  };
  document.addEventListener("keydown", seen);
  await act(async () => {
    target.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  document.removeEventListener("keydown", seen);
  await settled();
  return prevented;
}

beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  // jsdom has no Web Animations, which Headless UI's transitions read: no animation runs (its own stand-in, which it
  // puts in jsdom's elements under a test, warns on each call)
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});

afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
  vi.restoreAllMocks();
});

describe("ModalCore, an Escape", () => {
  it("leaves the focus on a field while the dialog has no handleClose, Headless UI's Dialog never acting on it", async () => {
    await open();
    const field = control<HTMLInputElement>("Email");
    await act(async () => field.focus());
    const prevented = await escape(field);
    expect(document.activeElement).toBe(field);
    expect(prevented).toBe(true);
  });

  it("leaves the dialog itself the focus while a button's request is out, the button having it back once enabled", async () => {
    await open();
    const button = control<HTMLButtonElement>("Send");
    // pressed, then disabled while its request is out: the browser blurs it, and the dialog takes the focus
    await act(async () => {
      button.focus();
      button.blur();
      button.disabled = true;
    });
    await settled();
    expect(document.activeElement).toBe(dialog());
    const prevented = await escape(dialog());
    // the request refused: the button enabled again
    await act(async () => {
      button.disabled = false;
    });
    await settled();
    expect(document.activeElement).toBe(button);
    expect(prevented).toBe(true);
  });

  it("closes the dialog while it has a handleClose", async () => {
    const handleClose = vi.fn();
    await open(handleClose);
    const field = control<HTMLInputElement>("Email");
    await act(async () => field.focus());
    await escape(field);
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it("closes a dialog with a handleClose nested in one without, the innermost dialog owning the Escape", async () => {
    const handleClose = vi.fn();
    // a confirmation rendered among the children of a dialog that may not close, as M4's "Save this draft?" in the
    // work item dialog: React bubbles the confirmation's keydown through the outer dialog too
    await act(async () => {
      root.render(
        <ModalCore isOpen>
          <ModalTitle>Work item</ModalTitle>
          <ModalCore isOpen handleClose={handleClose}>
            <ModalTitle>Save this draft?</ModalTitle>
            <input aria-label="Title" />
          </ModalCore>
        </ModalCore>
      );
    });
    await settled();
    const field = document.querySelector<HTMLInputElement>('[aria-label="Title"]');
    if (!field) throw new Error("no inner field");
    await act(async () => field.focus());
    await escape(field);
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it("keeps an Escape from a popup portaled out of the dialog the dialog's, by React's tree, not the DOM", async () => {
    // a popup among the children of a dialog that may not close, portaled to the body and a dialog by its role, as
    // Base UI's popover is (the emoji picker in create project): outside the dialog's element and its role="dialog",
    // yet React bubbles its keydown through the dialog that renders it (a composing Escape, which the popup lets by)
    await act(async () => {
      root.render(
        <ModalCore isOpen>
          <ModalTitle>Create project</ModalTitle>
          {createPortal(
            <div role="dialog" aria-label="Emoji">
              <input aria-label="Search" />
            </div>,
            document.body
          )}
        </ModalCore>
      );
    });
    await settled();
    const search = document.querySelector<HTMLInputElement>('[aria-label="Search"]');
    if (!search) throw new Error("no search in the popup");
    expect(await escape(search)).toBe(true);
  });
});
