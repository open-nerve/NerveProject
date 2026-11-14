/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ReactNode } from "react";
import { observer } from "mobx-react";
import type { Cycle } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { CycleGroupIcon } from "@nerve/propel/icons";
import { ChevronDownOutline, CyclesOutline } from "@makeplane/propel/icons";
import { CustomSearchSelect, Spinner } from "@nerve/ui";
import { cn } from "@nerve/utils";
import { DropdownButton } from "@/components/dropdowns/buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "@/components/dropdowns/constants";
import type { TDropdownProps } from "@/components/dropdowns/types";
import { usePlatformOS } from "@/hooks/use-platform-os";

/** No cycle as the select's value: the shared select takes no null for an option (null resets its search). */
const NO_CYCLE = "";

export type TCycleDropdownBaseProps = TDropdownProps & {
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  /** A cycle as the store has it, an ended or an archived one too: the value's, which may be no option. */
  getCycleById: (cycleId: string | null | undefined) => Cycle | undefined;
  /** The row's work item, for a dropdown each row of a list has (M3 closeout 2.9, P7a spec 2.4, R20). */
  issueName?: string;
  isInitializing?: boolean;
  /** The cycle picked, null for No cycle; never the cycle the work item is in. */
  onChange: (cycleId: string | null) => void;
  onDropdownOpen?: () => void;
  /** No cycle (null) and the cycles the work item may join (cycleOptions); undefined until read. */
  options: (Cycle | null)[] | undefined;
  readStatus?: ReactNode;
  /** The cycle the work item is in, null for none. */
  value: string | null;
};

/**
 * A work item's cycle (P7a spec 2.4, R4): one pick, which closes the list. Its button shows the cycle the work item is
 * in by its name and status, an ended or an archived one too, which the list does not offer; an archived one with the
 * caption Archived. The Popover owns the one native trigger, named by the field and the value; the Combobox is only
 * the search and the options in its panel.
 */
export const CycleDropdownBase = observer(function CycleDropdownBase(props: TCycleDropdownBaseProps) {
  const {
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className,
    disabled = false,
    dropdownArrow,
    dropdownArrowClassName,
    getCycleById,
    hideIcon,
    issueName,
    isInitializing,
    onChange,
    onDropdownOpen,
    placeholder,
    placement,
    readStatus,
    showTooltip = false,
    tabIndex,
    value,
  } = props;
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const selected = getCycleById(value);
  const text = value === null ? (placeholder ?? t("cycle.no_cycle")) : selected?.name;
  const archived = selected?.archived_at ? t("common.archived") : undefined;
  /** The value as the trigger is named by it: the name, and the caption of an archived cycle. */
  const named = archived ? `${text}, ${archived}` : text;
  const withText = BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant);
  const content = (
    <DropdownButton
      presentation
      className={cn("group-data-open:bg-layer-transparent-active", buttonClassName)}
      isActive={false}
      tooltipHeading={t("common.cycle")}
      tooltipContent={named}
      showTooltip={showTooltip}
      variant={buttonVariant}
    >
      <>
        {!hideIcon &&
          (isInitializing ? (
            <Spinner className="size-3 flex-shrink-0" />
          ) : selected ? (
            <CycleGroupIcon cycleGroup={selected.status} className="size-3 flex-shrink-0" />
          ) : (
            <CyclesOutline className="size-3 flex-shrink-0" />
          ))}
        {withText && text && <span className="flex-grow truncate text-left">{text}</span>}
        {withText && archived && (
          <span className="flex-shrink-0 text-caption-sm-regular text-placeholder">
            <span className="sr-only">, </span>
            {archived}
          </span>
        )}
        {dropdownArrow && (
          <ChevronDownOutline className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)} aria-hidden="true" />
        )}
      </>
    </DropdownButton>
  );
  const options = props.options?.map((cycle) =>
    cycle
      ? {
          value: cycle.id,
          query: cycle.name,
          content: (
            <div className="flex items-center gap-2">
              <CycleGroupIcon cycleGroup={cycle.status} className="size-3.5 flex-shrink-0" />
              <span className="flex-grow truncate">{cycle.name}</span>
            </div>
          ),
        }
      : {
          value: NO_CYCLE,
          query: t("cycle.no_cycle"),
          content: (
            <div className="flex items-center gap-2">
              <CyclesOutline className="size-3.5 flex-shrink-0" />
              <span className="flex-grow truncate">{t("cycle.no_cycle")}</span>
            </div>
          ),
        }
  );
  return (
    <CustomSearchSelect
      value={value ?? NO_CYCLE}
      onChange={(picked: string) => {
        const cycleId = picked === NO_CYCLE ? null : picked;
        if (cycleId !== value) onChange(cycleId);
      }}
      options={isInitializing ? undefined : readStatus ? [] : options}
      footerOption={readStatus}
      onOpen={onDropdownOpen}
      disabled={disabled}
      tabIndex={tabIndex}
      customButton={
        <>
          <span className="sr-only">{t("common.cycle")}: </span>
          {content}
          {!withText && <span className="sr-only">{named}</span>}
          {issueName && <span className="sr-only">, {issueName}</span>}
        </>
      }
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
