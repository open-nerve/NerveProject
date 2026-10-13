/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Dialog, DialogBackdrop, DialogPanel } from "@headlessui/react";
import React, { useEffect, useState } from "react";
// constants
import { cn } from "../utils";
import { EModalPosition, EModalWidth } from "./constants";

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
 * stay usable and read. Closed, it gives the focus back to what had it before it opened.
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

  return (
    <Dialog open={isOpen && mounted} onClose={() => handleClose?.()} className="relative z-30">
      <DialogBackdrop
        transition
        className="fixed inset-0 bg-backdrop transition-opacity duration-300 ease-out data-closed:opacity-0 data-leave:duration-200 data-leave:ease-in"
      />
      <div className="fixed inset-0 z-30 overflow-y-auto">
        <div className={position}>
          <DialogPanel
            transition
            className={cn(
              "relative w-full transform rounded-lg bg-surface-1 text-left shadow-raised-200 transition-all duration-300 ease-out data-closed:translate-y-4 data-closed:opacity-0 data-leave:duration-200 data-leave:ease-in sm:data-closed:translate-y-0 sm:data-closed:scale-95",
              width,
              className
            )}
            // an Escape that ends an IME composition is the composition's: Headless UI's Dialog closes on a keydown
            // of Escape on the window unless its default is prevented (useEscape), so this prevents it on the way
            // out, once the controls inside have had it (in the capture phase, it would make Headless UI skip their
            // own handlers, which it does for a prevented event: a select's, which keeps its Escape from the page)
            onKeyDown={(event) => {
              if (event.key === "Escape" && event.nativeEvent.isComposing) event.preventDefault();
            }}
          >
            {children}
          </DialogPanel>
        </div>
      </div>
    </Dialog>
  );
}
