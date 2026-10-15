// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { keepFocusIn } from "./keep-focus";

// A dialog's focus while its controls are disabled under it (M3 closeout, C11-3), in jsdom: a button's blur() is a
// focusout to nothing, the body taking the focus, as a browser's blur of a button disabled while it has the focus.

let dialog: HTMLElement;
let button: HTMLButtonElement;
let field: HTMLInputElement;
let stop: () => void;

/** Lets the mutation observers' callbacks run. */
const observed = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

/**
 * Disables button, which has the focus, as a dialog's request does, and the browser blurs it. A browser blurs the
 * button as it is disabled; jsdom blurs no disabled element, so the blur comes first here.
 */
function disabledUnderFocus(): void {
  button.focus();
  button.blur();
  button.disabled = true;
}

beforeEach(() => {
  dialog = document.createElement("div");
  dialog.tabIndex = -1;
  dialog.setAttribute("role", "dialog");
  button = document.createElement("button");
  field = document.createElement("input");
  dialog.append(field, button);
  document.body.append(dialog);
  stop = keepFocusIn(dialog);
});

afterEach(() => {
  stop();
  dialog.remove();
});

describe("keepFocusIn", () => {
  it("gives the focus to the dialog when a control in it loses it to nothing, and back to the control once enabled", async () => {
    disabledUnderFocus();
    await observed();
    expect(document.activeElement).toBe(dialog);
    button.disabled = false;
    await observed();
    expect(document.activeElement).toBe(button);
  });

  it("gives the dialog the focus once the focusout has been dispatched, not within it", async () => {
    // the focus as the focusout reaches the document, after the dialog: Chromium sends no key to the page once an
    // element given the focus within a focusout is removed
    const during: (Element | null)[] = [];
    const record = () => during.push(document.activeElement);
    document.addEventListener("focusout", record);
    disabledUnderFocus();
    document.removeEventListener("focusout", record);
    await observed();
    expect([during, document.activeElement]).toEqual([[document.body], dialog]);
  });

  it("leaves the focus with a field the page focused meanwhile, as a refused one", async () => {
    disabledUnderFocus();
    await observed();
    field.focus();
    button.disabled = false;
    await observed();
    expect(document.activeElement).toBe(field);
  });

  it("leaves the focus where it is when the window loses it", () => {
    button.focus();
    // the window's blur: the control is told, and the document keeps it as its active element
    button.dispatchEvent(new FocusEvent("focusout", { bubbles: true, relatedTarget: null }));
    expect(document.activeElement).toBe(button);
  });

  it("does nothing once stopped", async () => {
    stop();
    disabledUnderFocus();
    expect(document.activeElement).toBe(document.body);
    button.disabled = false;
    await observed();
    expect(document.activeElement).toBe(document.body);
  });
});
