/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Dialog, DialogBackdrop, DialogPanel, DialogTitle } from "@headlessui/react";
import React, { useEffect, useState } from "react";
import { isComposing } from "@nerve/utils";
// helpers
import { cn } from "../utils";
// constants
import { EModalPosition, EModalWidth } from "./constants";
import { keepFocusIn } from "./keep-focus";

/** Whether what renders here is inside a ModalCore dialog (useInsideModal). */
const InsideModal = React.createContext(false);

/**
 * Whether the calling component renders inside a ModalCore dialog. A list or menu that opens there is not modal
 * (Headless UI's modal prop, true by default): a modal one makes itself Headless UI's top layer, which turns the
 * dialog's own inert-others off, and its own makes inert only what is beside it below the body, so the page behind the
 * dialog would be read again while the list is open. The dialog already keeps the pointer, the scrolling and
 * assistive technology from that page.
 */
export function useInsideModal(): boolean {
  return React.useContext(InsideModal);
}

/**
 * The keydowns of Escape a ModalCore dialog has decided on: React bubbles one through each dialog it is nested in,
 * innermost first, and the first owns it (ModalCore's onKeyDown).
 */
const claimedEscapes = new WeakSet<Event>();

type Props = {
  children: React.ReactNode;
  /** Closes the dialog on Escape and a click on the backdrop; without it, neither closes it. */
  handleClose?: () => void;
  isOpen: boolean;
  position?: EModalPosition;
  width?: EModalWidth;
  className?: string;
};

/**
 * A modal dialog: Headless UI's Dialog, shown while isOpen, its backdrop and panel fading in and out; its caller may
 * mount it with its page or only while it is open. While it is open the page behind it is inert and hidden from
 * assistive technology (aria-hidden): Tab stays within the dialog, the pointer reaches nothing behind it, and a screen
 * reader reads the dialog alone; the lists, pickers and toasts that open from it render outside the page (portals) and
 * stay usable and read. The focus stays within it: a control that loses it while disabled (a button whose request is
 * out) leaves it to the dialog, and has it back once enabled (keepFocusIn); an Escape while it has no handleClose
 * leaves the focus where it is. Closed, it gives the focus back to what had it before it opened.
 */
export function ModalCore(props: Props) {
  const {
    children,
    handleClose,
    isOpen,
    position = EModalPosition.CENTER,
    width = EModalWidth.XXL,
    className = "",
  } = props;
  // Headless UI's Dialog finds the page behind it (its main tree) only once it has mounted, and makes that page inert
  // as it opens: one mounted open would leave the page live (P10's defect ii). So it opens from the render after its
  // first, a caller's dialog mounted only while open too.
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  // the dialog's element (Headless UI's, focusable from script: tabIndex -1), there while it is open
  const [dialog, setDialog] = useState<HTMLElement | null>(null);
  useEffect(() => (dialog ? keepFocusIn(dialog) : undefined), [dialog]);

  return (
    <Dialog
      ref={setDialog}
      open={isOpen && mounted}
      onClose={() => handleClose?.()}
      className="relative z-30 outline-none"
      // Headless UI's Dialog, on a keydown of Escape on the window, blurs what has the focus and asks to close, unless
      // its default is prevented (useEscape). This prevents both when Headless UI is to do neither: an Escape that ends
      // an input method's composition is the method's (isComposing); and while the dialog has no handleClose (its
      // request is out, or Headless UI is not to close it), Headless UI's close does nothing, and its blur would only
      // take the focus from where it is: a field, or the dialog itself while a disabled button waits to have it back
      // (keepFocusIn). A caller's own listener may still close the dialog on that Escape (useKeypress, on the document,
      // reads no defaultPrevented).
      // It prevents it on the way out, once the controls inside have had it, the lists they open too (React's tree): in
      // the capture phase, it would make Headless UI skip their own handlers, which it does for a prevented event (a
      // select's, which keeps its Escape from the page). It is the dialog element's, not the panel's: the dialog
      // element itself has the focus while a button's request is out, and a key pressed then never reaches the panel.
      // And the innermost dialog owns an Escape: React bubbles it through each dialog it is nested in, innermost first,
      // so the first to see it claims it and the others leave it alone, and a dialog with a handleClose nested in one
      // that may not close still closes on it.
      onKeyDown={(event) => {
        if (event.key !== "Escape" || claimedEscapes.has(event.nativeEvent)) return;
        claimedEscapes.add(event.nativeEvent);
        if (!handleClose || isComposing(event.nativeEvent)) event.preventDefault();
      }}
    >
      <DialogBackdrop
        transition
        className="fixed inset-0 bg-backdrop transition-opacity duration-300 ease-out data-closed:opacity-0 data-leave:duration-200 data-leave:ease-in"
      />
      <div className="fixed inset-0 z-30 overflow-y-auto">
        <div className={position}>
          <DialogPanel
            transition
            className={cn(
              // the timing is the enter's and the leave's alone: a caller's own transition (M4's issue dialog widens in
              // 150 ms) keeps its own
              "relative w-full transform rounded-lg bg-surface-1 text-left shadow-raised-200 transition-all data-closed:translate-y-4 data-closed:opacity-0 data-enter:duration-300 data-enter:ease-out data-leave:duration-200 data-leave:ease-in sm:data-closed:translate-y-0 sm:data-closed:scale-95",
              width,
              className
            )}
          >
            <InsideModal.Provider value={true}>{children}</InsideModal.Provider>
          </DialogPanel>
        </div>
      </div>
    </Dialog>
  );
}

type TitleProps = {
  children: React.ReactNode;
  className?: string;
};

/**
 * A ModalCore dialog's heading, and so its name: Headless UI's Dialog is named (aria-labelledby) only by its
 * DialogTitle, so a dialog whose heading is a plain one is announced with no name. A dialog with no visible heading
 * gives one that is visually hidden (className "sr-only"), never a second visible one.
 */
export function ModalTitle(props: TitleProps) {
  const { children, className } = props;
  return (
    <DialogTitle as="h3" className={className}>
      {children}
    </DialogTitle>
  );
}
