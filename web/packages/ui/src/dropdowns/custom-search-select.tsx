/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Combobox, Popover } from "@headlessui/react";
import { ChevronDownOutline, InfoOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { usePopper } from "react-popper";
// local imports
import { Tooltip } from "@nerve/propel/tooltip";
import { cn } from "../utils";
import type { ICustomSearchSelectProps } from "./helper";

/**
 * Calls onOpen as it mounts, with the list, and onClose as it unmounts, when the list closes: Headless UI's Popover
 * has no callback for either.
 */
function WhileOpen(props: { onOpen?: () => void; onClose?: () => void }) {
  // the callbacks of the moment the list opened: a caller's new functions on each render do not call them again
  const [{ onOpen, onClose }] = useState(() => props);
  useEffect(() => {
    onOpen?.();
    return () => onClose?.();
  }, [onOpen, onClose]);
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
    focusSearchOnOpen = true,
    popperModifiers,
  } = props;
  const [query, setQuery] = useState("");

  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  const openedByDefault = useRef(false);
  // The search input is in the list: it takes the focus as the list opens (unless the caller says not), so that
  // typing, the arrows, Enter and Escape reach Headless UI's input (a click on the button leaves the focus on the
  // button). Not scrolled to: popper has not placed the list yet.
  const focusSearch = useCallback((search: HTMLInputElement | null) => {
    search?.focus({ preventScroll: true });
  }, []);

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
    modifiers: popperModifiers,
  });

  // Headless UI's Popover has no defaultOpen: a list asked to start open opens once, as a click on its button does.
  useEffect(() => {
    if (!defaultOpen || openedByDefault.current || !referenceElement) return;
    openedByDefault.current = true;
    referenceElement.click();
  }, [defaultOpen, referenceElement]);

  const filteredOptions =
    query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase()));

  const comboboxProps: any = {
    value,
    disabled,
  };

  if (multiple) comboboxProps.multiple = true;

  return (
    // The button is Headless UI's Popover's, which Tab reaches (a Combobox's button is out of the Tab order: Headless
    // UI 2.2 gives it tabIndex -1 whatever its caller says); the search and the options are a Combobox in its panel.
    // The Popover's open state is the only one: its button opens the list (a click, Enter, Space); it closes it
    // (Escape, a pick of a single value, a click outside, the focus leaving it), and the focus goes back to the
    // button.
    <Popover className={cn("relative flex-shrink-0 text-left", className)}>
      {({ open, close }) => (
        <>
          {customButton ? (
            <Popover.Button
              ref={setReferenceElement}
              type="button"
              disabled={disabled}
              tabIndex={tabIndex}
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
            </Popover.Button>
          ) : (
            <Popover.Button
              ref={setReferenceElement}
              type="button"
              disabled={disabled}
              tabIndex={tabIndex}
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
            </Popover.Button>
          )}
          {open && (
            <WhileOpen
              onOpen={onOpen}
              onClose={() => {
                // the list opens again with no search
                setQuery("");
                onClose?.();
              }}
            />
          )}
          {createPortal(
            // Popper places the panel, which holds the list and nothing else.
            <Popover.Panel
              data-prevent-outside-click
              ref={setPopperElement}
              className="z-30"
              style={styles.popper}
              {...attributes.popper}
              // Escape closes the list, from the search too: once typing has opened the Combobox, its input takes
              // Escape first and prevents its default, and Headless UI then skips the panel's own handler. Not
              // passed on: a modal around the select stays open. An Escape that ends an input method's composition
              // (Chinese, say) is the input method's.
              onKeyDownCapture={(event: React.KeyboardEvent) => {
                if (event.key !== "Escape" || event.nativeEvent.isComposing) return;
                event.preventDefault();
                event.stopPropagation();
                close();
              }}
            >
              <Combobox
                {...comboboxProps}
                // A pick of a single value closes the list. Headless UI's Combobox picks null when its input is
                // emptied (single mode): the input here only filters the options, so emptying the search leaves the
                // value as it is.
                onChange={(picked: unknown) => {
                  if (multiple) {
                    onChange(picked);
                  } else if (picked !== null) {
                    onChange(picked);
                    close();
                  }
                }}
              >
                {/* Static: the list shows while the panel does, whatever the Combobox's own state. Not modal: the
                    panel is the list's, and a modal list makes the rest of the page inert (Headless UI 2.2). */}
                <Combobox.Options
                  as="ul"
                  static
                  modal={false}
                  className={cn(
                    "my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 py-2.5 text-11 whitespace-nowrap focus:outline-none",
                    optionsClassName
                  )}
                >
                  <div className="mx-2 flex items-center gap-1.5 rounded-sm border border-subtle px-2">
                    <SearchOutline className="h-3.5 w-3.5 text-placeholder" />
                    <Combobox.Input
                      ref={focusSearchOnOpen ? focusSearch : undefined}
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
                </Combobox.Options>
              </Combobox>
            </Popover.Panel>,
            document.body
          )}
        </>
      )}
    </Popover>
  );
}
