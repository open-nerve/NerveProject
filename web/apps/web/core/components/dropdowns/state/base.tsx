/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { ReactNode } from "react";
import { useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { StateGroupIcon } from "@nerve/propel/icons";
import { ChevronDownOutline } from "@makeplane/propel/icons";
import type { State } from "@nerve/api-client";
import { CustomSearchSelect, Spinner } from "@nerve/ui";
import { cn } from "@nerve/utils";
import { DropdownButton } from "@/components/dropdowns/buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "@/components/dropdowns/constants";
import type { TDropdownProps } from "@/components/dropdowns/types";
import { usePlatformOS } from "@/hooks/use-platform-os";

export type TWorkItemStateDropdownBaseProps = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  getStateById: (stateId: string | null | undefined) => State | undefined;
  getStatePercentageInGroup: (stateId: string | null | undefined) => number | undefined;
  iconSize?: string;
  isInitializing?: boolean;
  readStatus?: ReactNode;
  onChange: (val: string) => void;
  onClose?: () => void;
  onDropdownOpen?: () => void;
  projectId: string | undefined;
  renderByDefault?: boolean;
  showDefaultState?: boolean;
  stateIds: string[];
  value: string | undefined | null;
};

/** The Popover owns the one native trigger; the Combobox is only the search and options in its panel. */
export const WorkItemStateDropdownBase = observer(function WorkItemStateDropdownBase(
  props: TWorkItemStateDropdownBaseProps
) {
  const {
    button,
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className,
    disabled = false,
    dropdownArrow,
    dropdownArrowClassName,
    getStateById,
    getStatePercentageInGroup,
    hideIcon,
    iconSize = "size-4",
    isInitializing,
    readStatus,
    onChange,
    onClose,
    onDropdownOpen,
    placement,
    renderByDefault = true,
    showDefaultState = true,
    showTooltip = false,
    stateIds,
    tabIndex,
    value,
  } = props;
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const [rendered, setRendered] = useState(renderByDefault);
  const [openOnMount, setOpenOnMount] = useState(false);
  const states = stateIds.map((id) => getStateById(id)).filter((state) => !!state);
  const stateValue = value || (showDefaultState ? states.find((state) => state.default)?.id : undefined);
  const selected = stateValue ? getStateById(stateValue) : undefined;
  const show = () => {
    setOpenOnMount(true);
    setRendered(true);
  };
  const content = button ?? (
    <DropdownButton
      presentation={rendered}
      tabIndex={tabIndex}
      onClick={show}
      disabled={disabled}
      className={cn("group-data-open:bg-layer-transparent-active", buttonClassName)}
      isActive={false}
      tooltipHeading={t("state")}
      tooltipContent={selected?.name ?? t("state")}
      showTooltip={showTooltip}
      variant={buttonVariant}
    >
      {isInitializing ? (
        <Spinner className="h-3.5 w-3.5" />
      ) : (
        <>
          {!hideIcon && (
            <StateGroupIcon
              stateGroup={selected?.group ?? "backlog"}
              color={selected?.color ?? "var(--text-color-tertiary)"}
              className={cn("flex-shrink-0", iconSize)}
              percentage={getStatePercentageInGroup(selected?.id)}
            />
          )}
          {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
            <span className="flex-grow truncate text-left">{selected?.name ?? t("state")}</span>
          )}
          {dropdownArrow && (
            <ChevronDownOutline
              className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)}
              aria-hidden="true"
            />
          )}
        </>
      )}
    </DropdownButton>
  );
  if (!rendered)
    return (
      <div className="flex h-full items-center" onMouseEnter={() => setRendered(true)}>
        {button ? (
          <button type="button" onClick={show} tabIndex={tabIndex} disabled={disabled}>
            {content}
          </button>
        ) : (
          content
        )}
      </div>
    );
  const options = states.map((state) => ({
    value: state.id,
    query: state.name,
    content: (
      <div className="flex items-center gap-2">
        <StateGroupIcon
          stateGroup={state.group}
          color={state.color}
          className={cn("flex-shrink-0", iconSize)}
          percentage={getStatePercentageInGroup(state.id)}
        />
        <span className="flex-grow truncate text-left">{state.name}</span>
      </div>
    ),
  }));
  return (
    <CustomSearchSelect
      value={stateValue ?? null}
      onChange={(id: string) => onChange(id)}
      options={isInitializing ? undefined : readStatus ? [] : options}
      footerOption={readStatus}
      onOpen={onDropdownOpen}
      onClose={onClose}
      disabled={disabled}
      defaultOpen={openOnMount}
      tabIndex={tabIndex}
      customButton={content}
      customButtonClassName={cn("group h-full max-w-full hover:bg-transparent", buttonContainerClassName)}
      className={cn("h-full shrink", className)}
      optionsClassName="w-48"
      placement={placement}
      searchPlaceholder={t("common.search.label")}
      noResultsMessage={readStatus ? "" : t("no_matching_results")}
      loadingMessage={t("common.loading")}
      focusSearchOnOpen={!isMobile}
      popperModifiers={[{ name: "preventOverflow", options: { padding: 12 } }]}
    />
  );
});
