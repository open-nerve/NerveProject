/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Listbox } from "@headlessui/react";

import React, { useState } from "react";
import { createPortal } from "react-dom";
import { usePopper } from "react-popper";
import { ChevronDownOutline, TickOutline } from "@makeplane/propel/icons";
// helpers
import { useInsideModal } from "../modals/modal-core";
import { isComposing } from "@nerve/utils";
import { cn } from "../utils";
import { fieldButtonProps } from "./field-button";
// types
import type { ICustomSelectItemProps, ICustomSelectProps } from "./helper";

// Enter on the button opens the list, as Space does. Headless UI's button would submit the form around it, as a
// native select's Enter does, and a keyboard user would send a form (a member's role, say) unawares; prevented here,
// before Headless UI's handler, the Enter reaches none of Headless UI's, and the button is given Space's keydown, which
// Headless UI opens the list on. Not a click: once the pointer has opened the list, Headless UI's button takes a click
// with no pointer before it for the pointer's own, and ignores it.
function openOnEnter(event: React.KeyboardEvent<HTMLButtonElement>) {
  if (event.key !== "Enter") return;
  event.preventDefault();
  if (isComposing(event.nativeEvent)) return;
  event.currentTarget.dispatchEvent(new KeyboardEvent("keydown", { key: " ", bubbles: true, cancelable: true }));
}

function CustomSelect(props: ICustomSelectProps) {
  const {
    customButtonClassName = "",
    buttonClassName = "",
    placement,
    children,
    className = "",
    customButton,
    disabled = false,
    input = false,
    label,
    maxHeight = "md",
    noChevron = false,
    onChange,
    optionsClassName = "",
    value,
    tabIndex,
  } = props;
  // states
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  // in a form, the button is one of its fields
  const described = fieldButtonProps(props);
  // in a dialog, the list is not modal: the dialog keeps the page behind it hidden (useInsideModal)
  const insideModal = useInsideModal();

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
  });

  return (
    // A select, so Headless UI's Listbox, whose open state is the only one. Its button opens the list (a click, Enter,
    // Space, the arrows), and the list takes the focus: the arrows move, Enter or Space picks, typing jumps to a name.
    // A pick, Escape, Tab or a click outside closes it.
    // The caller's tabIndex is the button's: Tab stops at the button alone, not at the Listbox's root (a div).
    <Listbox
      as="div"
      value={value}
      onChange={onChange}
      className={cn("relative flex-shrink-0 text-left", className)}
      disabled={disabled}
    >
      {({ open }) => (
        <>
          {customButton ? (
            <Listbox.Button as={React.Fragment}>
              <button
                ref={setReferenceElement}
                type="button"
                tabIndex={tabIndex}
                onKeyDown={openOnEnter}
                {...described}
                className={`flex items-center justify-between gap-1 rounded text-11 ${
                  disabled ? "cursor-not-allowed text-secondary" : "cursor-pointer hover:bg-layer-transparent-hover"
                } ${customButtonClassName}`}
              >
                {customButton}
              </button>
            </Listbox.Button>
          ) : (
            <Listbox.Button as={React.Fragment}>
              <button
                ref={setReferenceElement}
                type="button"
                tabIndex={tabIndex}
                onKeyDown={openOnEnter}
                {...described}
                className={cn(
                  "flex w-full items-center justify-between gap-1 rounded border border-strong",
                  {
                    "px-3 py-2 text-13": input,
                    "px-2 py-1 text-11": !input,
                    "cursor-not-allowed text-secondary": disabled,
                    "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                  },
                  buttonClassName
                )}
              >
                {label}
                {!noChevron && !disabled && <ChevronDownOutline className="h-3 w-3" aria-hidden="true" />}
              </button>
            </Listbox.Button>
          )}
          {open &&
            createPortal(
              // Popper places the list itself, whose ref Headless UI forwards.
              <Listbox.Options
                as="ul"
                modal={!insideModal}
                data-prevent-outside-click
                ref={setPopperElement}
                className={cn(
                  "z-30 my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 px-2 py-2.5 text-11 whitespace-nowrap focus:outline-none",
                  optionsClassName
                )}
                style={styles.popper}
                {...attributes.popper}
              >
                <div
                  className={cn("space-y-1 overflow-y-scroll", {
                    "max-h-60": maxHeight === "lg",
                    "max-h-48": maxHeight === "md",
                    "max-h-36": maxHeight === "rg",
                    "max-h-28": maxHeight === "sm",
                  })}
                >
                  {children}
                </div>
              </Listbox.Options>,
              document.body
            )}
        </>
      )}
    </Listbox>
  );
}

function Option(props: ICustomSelectItemProps) {
  const { children, value, className } = props;

  return (
    <Listbox.Option
      as="li"
      value={value}
      className={({ focus }) =>
        cn(
          "flex cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 text-secondary select-none",
          {
            "bg-layer-transparent-hover": focus,
          },
          className
        )
      }
    >
      {({ selected }) => (
        <div className="flex w-full items-center justify-between gap-2">
          {children}
          {selected && <TickOutline className="h-3.5 w-3.5 flex-shrink-0" />}
        </div>
      )}
    </Listbox.Option>
  );
}

CustomSelect.Option = Option;

export { CustomSelect };
