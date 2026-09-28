/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Combobox } from "@headlessui/react";
import { ChevronDownOutline, InfoOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { usePopper } from "react-popper";
// local imports
import { Tooltip } from "@nerve/propel/tooltip";
import { cn } from "../utils";
import type { ICustomSearchSelectProps } from "./helper";

/** Calls onOpen once as it mounts, with the list: Headless UI has no callback for opening. */
function OnOpen(props: { onOpen: () => void }) {
  // the callback of the moment the list opened: a caller's new function on each render does not call it again
  const [onOpen] = useState(() => props.onOpen);
  useEffect(() => {
    onOpen();
  }, [onOpen]);
  return null;
}

export function CustomSearchSelect(props: ICustomSearchSelectProps) {
  const {
    customButtonClassName = "",
    buttonClassName = "",
    className = "",
    chevronClassName = "",
    customButton,
    placement,
    disabled = false,
    footerOption,
    input = false,
    label,
    maxHeight = "md",
    multiple = false,
    noChevron = false,
    onChange,
    options,
    onOpen,
    onClose,
    optionsClassName = "",
    value,
    tabIndex,
    searchPlaceholder = "Search",
    noResultsMessage = "No matches found",
    loadingMessage = "Loading...",
    defaultOpen = false,
  } = props;
  const [query, setQuery] = useState("");

  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  const openedByDefault = useRef(false);
  // The search input is in the list: it takes the focus as the list opens, so that typing, the arrows, Enter and
  // Escape reach Headless UI's input (a click on the button leaves the focus nowhere). Not scrolled to: popper has
  // not placed the list yet.
  const focusSearch = useCallback((search: HTMLInputElement | null) => {
    search?.focus({ preventScroll: true });
  }, []);

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
  });

  // Headless UI's Combobox has no defaultOpen: a list asked to start open opens once, as a click on its button does.
  useEffect(() => {
    if (!defaultOpen || openedByDefault.current || !referenceElement) return;
    openedByDefault.current = true;
    referenceElement.click();
  }, [defaultOpen, referenceElement]);

  const filteredOptions =
    query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase()));

  const comboboxProps: any = {
    value,
    // Headless UI's Combobox picks null when its input is emptied (single mode). The input here only filters the
    // options, so emptying the search leaves the value as it is.
    onChange: (picked: unknown) => {
      if (multiple || picked !== null) onChange(picked);
    },
    disabled,
  };

  if (multiple) comboboxProps.multiple = true;

  return (
    <Combobox
      as="div"
      tabIndex={tabIndex}
      className={cn("relative flex-shrink-0 text-left", className)}
      onClose={onClose}
      {...comboboxProps}
    >
      {({ open }: { open: boolean }) => (
        <>
          {customButton ? (
            <Combobox.Button as={React.Fragment}>
              <button
                ref={setReferenceElement}
                type="button"
                className={cn(
                  "flex w-full items-center justify-between gap-1 text-11",
                  {
                    "cursor-not-allowed text-secondary": disabled,
                    "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                  },
                  customButtonClassName
                )}
              >
                {customButton}
              </button>
            </Combobox.Button>
          ) : (
            <Combobox.Button as={React.Fragment}>
              <button
                ref={setReferenceElement}
                type="button"
                className={cn(
                  "flex w-full items-center justify-between gap-1 rounded-sm border-[0.5px] border-strong",
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
                {!noChevron && !disabled && (
                  <ChevronDownOutline className={cn("h-3 w-3 flex-shrink-0", chevronClassName)} aria-hidden="true" />
                )}
              </button>
            </Combobox.Button>
          )}
          {/* Headless UI's own open state is the only one: its button opens the list (a click, Enter, Space, the
              arrows); it closes it (Escape, a pick, a click outside, another list opening) and calls onClose. */}
          {open && onOpen && <OnOpen onOpen={onOpen} />}
          {open &&
            createPortal(
              // Popper places the list itself, whose ref Headless UI forwards. A ref on the list's one child would
              // never be set: Combobox.Options clones that child with a ref of its own (Headless UI 2.2), so the
              // list would stay where the page begins, at its top left. Not modal: a modal list makes inert
              // whatever holds none of the input, the button and the list, going up from each (Headless UI 2.2);
              // the search input is in the list, so that would be the options.
              <Combobox.Options
                as="ul"
                data-prevent-outside-click
                modal={false}
                ref={setPopperElement}
                className={cn(
                  "z-30 my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 py-2.5 text-11 whitespace-nowrap focus:outline-none",
                  optionsClassName
                )}
                style={styles.popper}
                {...attributes.popper}
              >
                <div className="mx-2 flex items-center gap-1.5 rounded-sm border border-subtle px-2">
                  <SearchOutline className="h-3.5 w-3.5 text-placeholder" />
                  <Combobox.Input
                    ref={focusSearch}
                    className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    placeholder={searchPlaceholder}
                    displayValue={(assigned: any) => assigned?.name}
                  />
                </div>
                <div
                  className={cn("vertical-scrollbar mt-2 scrollbar-xs space-y-1 overflow-y-scroll px-2", {
                    "max-h-96": maxHeight === "2xl",
                    "max-h-80": maxHeight === "xl",
                    "max-h-60": maxHeight === "lg",
                    "max-h-48": maxHeight === "md",
                    "max-h-36": maxHeight === "rg",
                    "max-h-28": maxHeight === "sm",
                  })}
                >
                  {filteredOptions ? (
                    filteredOptions.length > 0 ? (
                      filteredOptions.map((option) => (
                        <Combobox.Option
                          as="li"
                          key={option.value}
                          value={option.value}
                          className={({ active }) =>
                            cn(
                              "flex w-full cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none",
                              {
                                "bg-layer-transparent-hover": active,
                                "cursor-not-allowed text-placeholder opacity-60": option.disabled,
                              }
                            )
                          }
                          disabled={option.disabled}
                        >
                          {({ selected }) => (
                            <>
                              <span className="flex-grow truncate">{option.content}</span>
                              {selected && <TickOutline className="h-3.5 w-3.5 flex-shrink-0" />}
                              {option.tooltip && (
                                <>
                                  {typeof option.tooltip === "string" ? (
                                    <Tooltip tooltipContent={option.tooltip}>
                                      <InfoOutline className="h-3.5 w-3.5 flex-shrink-0 cursor-pointer text-secondary" />
                                    </Tooltip>
                                  ) : (
                                    option.tooltip
                                  )}
                                </>
                              )}
                            </>
                          )}
                        </Combobox.Option>
                      ))
                    ) : (
                      <p className="px-1.5 py-1 text-placeholder italic">{noResultsMessage}</p>
                    )
                  ) : (
                    <p className="px-1.5 py-1 text-placeholder italic">{loadingMessage}</p>
                  )}
                </div>
                {footerOption}
              </Combobox.Options>,
              document.body
            )}
        </>
      )}
    </Combobox>
  );
}
