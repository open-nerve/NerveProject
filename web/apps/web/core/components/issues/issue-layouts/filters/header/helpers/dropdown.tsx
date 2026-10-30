/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { Fragment, cloneElement, useState } from "react";
import type { Placement } from "@popperjs/core";
import { usePopper } from "react-popper";
// headless ui
import { Popover, Transition } from "@headlessui/react";
// ui
import { Button } from "@nerve/propel/button";
import { useTranslation } from "@nerve/i18n";

type Props = {
  children: React.ReactNode;
  icon?: React.ReactElement<{ className?: string; strokeWidth?: number | string }>;
  miniIcon?: React.ReactNode;
  title?: string;
  placement?: Placement;
  disabled?: boolean;
  tabIndex?: number;
  menuButton?: React.ReactNode;
  isFiltersApplied?: boolean;
};

/**
 * A filters' popover. Its button is Headless UI's Popover.Button itself: Enter or Space opens it, which it says
 * (aria-expanded), and Escape closes it, the focus back on the button. The panel follows the button: Tab goes into it
 * and through its controls, and Headless UI closes it as the focus moves on past them, not as it goes into a dialog
 * the panel opens.
 */
export function FiltersDropdown(props: Props) {
  const { t } = useTranslation();
  const {
    children,
    miniIcon,
    icon,
    title = t("common.filters"),
    placement,
    disabled = false,
    tabIndex,
    menuButton,
    isFiltersApplied = false,
  } = props;

  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "auto",
    strategy: "fixed",
  });

  return (
    <Popover as="div">
      {({ open }) => (
        <>
          {menuButton ? (
            <Popover.Button as={React.Fragment}>
              <button
                type="button"
                ref={setReferenceElement}
                aria-label={title}
                disabled={disabled}
                tabIndex={tabIndex}
              >
                {menuButton}
              </button>
            </Popover.Button>
          ) : (
            // one button, named by the title, whose content follows the header's width: in a wide header the icon,
            // the title and the mark of the filters applied, in a narrow one the mini icon or the title
            <Popover.Button as={React.Fragment}>
              <Button
                ref={setReferenceElement}
                disabled={disabled}
                variant="secondary"
                tabIndex={tabIndex}
                className="@4xl:relative"
                size="lg"
                aria-label={title}
              >
                <span className="hidden @4xl:contents">
                  {/* the icon as the button's prependIcon shows it */}
                  {icon && cloneElement(icon, { className: "shrink-0 size-4", strokeWidth: 2 })}
                  <span className={`${open ? "text-primary" : "text-secondary"}`}>
                    <span>{title}</span>
                  </span>
                  {isFiltersApplied && (
                    <span className="absolute -top-0.5 -right-0.5 h-2 w-2 rounded-full bg-accent-primary" />
                  )}
                </span>
                <span className="contents @4xl:hidden">{miniIcon || title}</span>
              </Button>
            </Popover.Button>
          )}
          <Transition
            as={Fragment}
            enter="transition ease-out duration-200"
            enterFrom="opacity-0 translate-y-1"
            enterTo="opacity-100 translate-y-0"
            leave="transition ease-in duration-150"
            leaveFrom="opacity-100 translate-y-0"
            leaveTo="opacity-0 translate-y-1"
          >
            {/** translate-y-0 is a hack to create new stacking context. Required for safari  */}
            {/* The Transition takes the panel out of the page after its leave, so the panel keeps its place in Headless
                UI's state as the popover closes (unmount false): opened again during the leave, it would otherwise no
                longer close as the focus moves on past it. */}
            <Popover.Panel
              className="fixed z-10 translate-y-0"
              unmount={false}
              ref={setPopperElement}
              style={styles.popper}
              {...attributes.popper}
            >
              <div className="my-1 overflow-hidden rounded-sm border border-subtle bg-surface-1 shadow-raised-100">
                <div className="flex max-h-[30rem] w-[18.75rem] flex-col overflow-hidden lg:max-h-[37.5rem]">
                  {children}
                </div>
              </div>
            </Popover.Panel>
          </Transition>
        </>
      )}
    </Popover>
  );
}
