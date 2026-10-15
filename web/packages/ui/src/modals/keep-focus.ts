/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * Keeps the focus within dialog, an open dialog that its title names, until the returned function stops it (M3
 * closeout, C11-3). A control inside the dialog that loses the focus to nothing, so that the document's body has it,
 * gives it to the dialog: a button disabled while its dialog's request is out does (Chromium and Firefox blur it, WebKit
 * too when it was pressed from the keyboard), and a screen reader would be left on the page otherwise. Once that
 * control is enabled again while the dialog still has the focus, the focus goes back to it; a control the page focuses
 * meanwhile, a refused field say, keeps it. The window's losing the focus (the user switching to another application)
 * leaves the document's active element where it was, and is no such loss.
 *
 * The dialog takes the focus once the focusout has been dispatched (a microtask), not within it: Chromium, given the
 * focus within a focusout, sends no key to the page once the element that took it is removed (the dialog, closed).
 * The control has it back a microtask after the observer hears it enabled: the commit that enables it queues the
 * observer's callback as it changes the DOM, before its layout effects queue theirs, and a focus one of those gives (a
 * refused field's, useRefusedFocus) comes first, the control never taking the focus in between.
 */
export function keepFocusIn(dialog: HTMLElement): () => void {
  const document = dialog.ownerDocument;
  let stopped = false;
  // the return to the control that lost the focus, while it waits for the control to be enabled
  let waiting: MutationObserver | undefined;
  const stopWaiting = () => {
    waiting?.disconnect();
    waiting = undefined;
  };
  const lostToNothing = () => document.activeElement === null || document.activeElement === document.body;

  const onFocusOut = (event: FocusEvent) => {
    const lost = event.target;
    if (event.relatedTarget !== null || !(lost instanceof HTMLElement) || !lostToNothing()) return;
    queueMicrotask(() => {
      if (stopped || !dialog.isConnected || !lostToNothing()) return;
      stopWaiting();
      dialog.focus({ preventScroll: true });
      waiting = new MutationObserver(() => {
        if (lost.matches(":disabled")) return;
        stopWaiting();
        queueMicrotask(() => {
          if (!stopped && lost.isConnected && document.activeElement === dialog) lost.focus({ preventScroll: true });
        });
      });
      waiting.observe(lost, { attributes: true, attributeFilter: ["disabled"] });
    });
  };

  dialog.addEventListener("focusout", onFocusOut);
  return () => {
    stopped = true;
    dialog.removeEventListener("focusout", onFocusOut);
    stopWaiting();
  };
}
