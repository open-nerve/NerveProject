/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * Keeps the focus within dialog, an open dialog that its title names, until the returned function stops it (M3
 * closeout, C11-3). When an element inside the dialog loses the focus to nothing, so that the document's body has it
 * and a screen reader would be left on the page, what follows depends on what lost it:
 * - the dialog itself takes it again, and a return the dialog is waiting to make (below) still comes. Headless UI's
 *   Dialog blurs whatever has the focus on every Escape before it asks to close, so a dialog that stays open (busy
 *   while its request is out, or with no handleClose, as the invitations' dialog) loses it this way;
 * - a control that can take the focus now (connected, not disabled) has it back: that is an Escape's blur too, not a
 *   busy state, and the person keeps their place;
 * - a disabled control leaves it to the dialog: a button disabled while its dialog's request is out does (Chromium and
 *   Firefox blur it, WebKit too when it was pressed from the keyboard). Once that control is enabled again while the
 *   dialog still has the focus, the focus goes back to it; a control the page focuses meanwhile, a refused field say,
 *   keeps it.
 *
 * The first two cases are one in the code: what lost the focus and can take it now has it back, and a return being
 * waited for stays. The window's losing the focus (the user switching to another application) leaves the document's
 * active element where it was, and is no such loss.
 *
 * The focus is given once the focusout has been dispatched (a microtask), not within it: Chromium, given the focus
 * within a focusout, sends no key to the page once the element that took it is removed (the dialog, closed). A dialog
 * that does close on Escape closes after that microtask, and gives the focus back to its opener as it closes. The
 * control has it back a microtask after the observer hears it enabled: the commit that enables it queues the
 * observer's callback as it changes the DOM, before its layout effects queue theirs, and a focus one of those gives (a
 * refused field's, useRefusedFocus) comes first, the control never taking the focus in between.
 */
export function keepFocusIn(dialog: HTMLElement): () => void {
  const document = dialog.ownerDocument;
  let stopped = false;
  // the return to the disabled control that lost the focus, while it waits for the control to be enabled
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
      // the dialog itself, or a control an Escape blurred: it has the focus back, and a return waited for still comes
      if (lost.isConnected && !lost.matches(":disabled")) {
        lost.focus({ preventScroll: true });
        return;
      }
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
