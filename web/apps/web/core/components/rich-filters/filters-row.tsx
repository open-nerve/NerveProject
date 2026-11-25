/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useCallback, useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { AddFilterOutline } from "@makeplane/propel/icons";
import { Transition } from "@headlessui/react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { IFilterInstance } from "@nerve/shared-state";
import type { TExternalFilter, TFilterProperty } from "@nerve/types";
import { cn, EHeaderVariant, Header, Loader } from "@nerve/ui";
// local imports
import type { TAddFilterButtonProps } from "./add-filters/button";
import { AddFilterButton } from "./add-filters/button";
import { FilterItem } from "./filter-item/root";

export type TFiltersRowProps<K extends TFilterProperty, E extends TExternalFilter> = {
  buttonConfig?: TAddFilterButtonProps<K, E>["buttonConfig"];
  /**
   * The page's draft is being saved (a view's page's Update view out, P6a spec 2.7, R29): the conditions, Add filter
   * (gone meanwhile), Clear all and Save view stop; Update view says itself out, as ever. Off unless given.
   */
  disabled?: boolean;
  disabledAllOperations?: boolean;
  filter: IFilterInstance<K, E>;
  variant?: "modal" | "header";
};

/**
 * Gives the focus, lost as Update view's button went (P6a spec 2.7; re-96 m6), to the row's Add filter, or else, the
 * row gone, to the page's filters toggle (FiltersToggle's mark): to the first of them that takes it. Not when the focus
 * is elsewhere: where the person put it while nerve answered.
 */
function landAfterUpdate(update: HTMLElement, addFilter: HTMLElement | null) {
  const focused = document.activeElement;
  if (focused !== null && focused !== document.body && focused !== update) return;
  const toggles = document.querySelectorAll<HTMLElement>("[data-filters-toggle] button");
  for (const target of [addFilter?.querySelector<HTMLElement>("button"), ...toggles]) {
    target?.focus();
    if (target && document.activeElement === target) return;
  }
}

export const FiltersRow = observer(function FiltersRow<K extends TFilterProperty, E extends TExternalFilter>(
  props: TFiltersRowProps<K, E>
) {
  const {
    buttonConfig,
    disabled = false,
    disabledAllOperations: disabledAllOperationsProp = false,
    filter,
    variant = "header",
  } = props;
  const { t } = useTranslation();
  // states
  const [isUpdating, setIsUpdating] = useState(false);
  // Update view confirmed: where the focus lands, once the row is drawn again (its Add filter's place)
  const addFilter = useRef<HTMLSpanElement>(null);
  const [landing, setLanding] = useState<{ from: HTMLElement }>();
  const landed = useRef<typeof landing>(undefined);
  useLayoutEffect(() => {
    // once the row works again: its Add filter, gone while the draft was being saved, drawn again
    if (!landing || disabled || landed.current === landing) return;
    landed.current = landing;
    queueMicrotask(() => landAfterUpdate(landing.from, addFilter.current));
  }, [landing, disabled]);
  // derived values
  const disabledAllOperations = disabledAllOperationsProp || !filter.configManager.areConfigsReady;
  const hasAnyConditions = filter.allConditions.length > 0;
  const hasAvailableOperations =
    !disabledAllOperations && (filter.canClearFilters || filter.canSaveView || filter.canUpdateView);

  const headerButtonConfig: Partial<TAddFilterButtonProps<K, E>["buttonConfig"]> = {
    label: null,
  };

  const modalButtonConfig: Partial<TAddFilterButtonProps<K, E>["buttonConfig"]> = {
    label: !hasAnyConditions ? t("common.filters") : null,
  };

  const handleUpdate = useCallback(
    async (event: React.MouseEvent<HTMLButtonElement>) => {
      const update = event.currentTarget;
      setIsUpdating(true);
      try {
        await filter.updateView();
        setLanding({ from: update });
      } catch {
        // refused: the page said why; the change stays the row's, and so does its button
      } finally {
        setTimeout(() => setIsUpdating(false), 240); // To avoid flickering
      }
    },
    [filter]
  );

  const leftContent = (
    <>
      {filter.allConditions.map((condition) => (
        <FilterItem
          key={condition.id}
          filter={filter}
          condition={condition}
          isDisabled={disabledAllOperations || disabled}
        />
      ))}
      <span ref={addFilter} className="contents">
        <AddFilterButton
          filter={filter}
          buttonConfig={{
            label: null,
            ...(variant === "modal" ? modalButtonConfig : headerButtonConfig),
            size: "lg",
            iconConfig: {
              shouldShowIcon: true,
              iconComponent: AddFilterOutline,
            },
            ...buttonConfig,
            isDisabled: disabledAllOperations || disabled,
          }}
        />
      </span>
    </>
  );

  const rightContent = !disabledAllOperations && (
    <>
      <ElementTransition show={filter.canClearFilters}>
        <Button
          variant="secondary"
          className={COMMON_OPERATION_BUTTON_CLASSNAME}
          onClick={filter.clearFilters}
          disabled={disabled}
        >
          {filter.clearFilterOptions?.label ?? t("common.clear_all")}
        </Button>
      </ElementTransition>
      <ElementTransition show={filter.canSaveView}>
        <Button
          variant="secondary"
          className={COMMON_OPERATION_BUTTON_CLASSNAME}
          onClick={filter.saveView}
          disabled={disabled}
        >
          {filter.saveViewOptions?.label ?? t("common.filter_options.save_view")}
        </Button>
      </ElementTransition>
      <ElementTransition show={filter.canUpdateView}>
        <Button
          variant="secondary"
          className={COMMON_OPERATION_BUTTON_CLASSNAME}
          onClick={handleUpdate}
          loading={isUpdating}
          disabled={isUpdating}
        >
          {isUpdating
            ? t("common.filter_options.confirming")
            : (filter.updateViewOptions?.label ?? t("common.filter_options.update_view"))}
        </Button>
      </ElementTransition>
    </>
  );

  const mainContent = (
    <div className="flex w-full items-start gap-2 rounded-lg bg-layer-1 px-4 py-2">
      <div className="flex w-full flex-wrap items-center gap-2">{leftContent}</div>
      <div
        className={cn("flex items-center gap-2 border-l border-subtle pl-4", {
          "border-l-transparent pl-0": !hasAvailableOperations,
        })}
      >
        {rightContent}
      </div>
    </div>
  );

  const ModalVariant = (
    <div className="flex min-h-11 w-full flex-wrap items-center gap-2 rounded-lg bg-layer-1 p-2">{mainContent}</div>
  );

  const HeaderVariant = (
    <Header variant={EHeaderVariant.TERNARY} className="min-h-11 bg-surface-1 !px-3">
      {mainContent}
    </Header>
  );

  if (!filter.configManager.areConfigsReady && !hasAnyConditions) {
    return (
      <RowTransition show={filter.isVisible}>
        <Loader>
          <Loader.Item height="44px" width="100%" className={cn({ "rounded-none": variant === "header" })} />
        </Loader>
      </RowTransition>
    );
  }

  return <RowTransition show={filter.isVisible}>{variant === "modal" ? ModalVariant : HeaderVariant}</RowTransition>;
});

const COMMON_OPERATION_BUTTON_CLASSNAME = "py-1";

type TElementTransitionProps = {
  children: React.ReactNode;
  show: boolean;
};

const ElementTransition = observer(function ElementTransition(props: TElementTransitionProps) {
  return (
    <Transition
      as="div"
      show={props.show}
      enter="transition ease-out duration-200"
      enterFrom="opacity-0 scale-95"
      enterTo="opacity-100 scale-100"
      leave="transition ease-in duration-150"
      leaveFrom="opacity-100 scale-100"
      leaveTo="opacity-0 scale-95"
    >
      {props.children}
    </Transition>
  );
});

type TRowTransitionProps = {
  children: React.ReactNode;
  show: boolean;
};

const RowTransition = observer(function RowTransition(props: TRowTransitionProps) {
  return (
    <Transition
      as="div"
      show={props.show}
      enter="transition-all duration-150 ease-out"
      enterFrom="opacity-0 -translate-y-1"
      enterTo="opacity-100 translate-y-0"
      leave="transition-all duration-100 ease-in"
      leaveFrom="opacity-100 translate-y-0"
      leaveTo="opacity-0 -translate-y-1"
    >
      {props.children}
    </Transition>
  );
});
