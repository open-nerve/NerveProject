/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ReactNode } from "react";
import { observer } from "mobx-react";
import type { Module } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { ChevronDownOutline, ModuleOutline } from "@makeplane/propel/icons";
import type { ModuleSetChange } from "@nerve/types";
import { CustomSearchSelect, Spinner } from "@nerve/ui";
import { cn } from "@nerve/utils";
import { DropdownButton } from "@/components/dropdowns/buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "@/components/dropdowns/constants";
import type { TDropdownProps } from "@/components/dropdowns/types";
import { usePlatformOS } from "@/hooks/use-platform-os";

export type TModuleDropdownBaseProps = TDropdownProps & {
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  /** A module as the store has it, an archived one too: a value's, which may be no option. */
  getModuleById: (moduleId: string | null | undefined) => Module | undefined;
  /** The row's work item, for a dropdown each row of a list has (M3 closeout 2.9, P7a spec 2.4, R20). */
  issueName?: string;
  isInitializing?: boolean;
  /** A pick: the one module it adds the work item to, or takes it out of; never the set it is in (R3). */
  onChange: (change: ModuleSetChange) => void;
  onDropdownOpen?: () => void;
  /** The modules the work item may join (moduleOptions); undefined until read. */
  options: Module[] | undefined;
  readStatus?: ReactNode;
  /** The modules the work item is in. */
  value: readonly string[];
};

/** A module's name in a list or on the button, with the caption of an archived one. */
function ModuleName({ name, caption }: { name: string; caption?: string }) {
  return (
    <>
      <span className="flex-grow truncate text-left">{name}</span>
      {caption && (
        <span className="flex-shrink-0 text-caption-sm-regular text-placeholder">
          <span className="sr-only">, </span>
          {caption}
        </span>
      )}
    </>
  );
}

/**
 * A work item's modules (P7a spec 2.4, R3, R12, R13): many picks, the list staying open, each one module added or
 * taken out, never the set, so that each is applied in its turn to the set the work item then holds. Its options are
 * in nerve's order, those it is in where they are; an archived module it is in follows them, captioned, there only to
 * be taken out. Its button says No module, the module's name (an archived one captioned), or how many; it holds no
 * button of its own: the Popover owns the one native trigger, named by the field and the value (each module's name);
 * the Combobox is only the search and the options in its panel.
 */
export const ModuleDropdownBase = observer(function ModuleDropdownBase(props: TModuleDropdownBaseProps) {
  const {
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className,
    disabled = false,
    dropdownArrow,
    dropdownArrowClassName,
    getModuleById,
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
  const archived = t("common.archived");
  const held = value.map((moduleId) => getModuleById(moduleId)).filter((item): item is Module => item !== undefined);
  const captionOf = (item: Module) => (item.archived_at ? archived : undefined);
  /** The modules it is in as the trigger is named by them: each name, and the caption of an archived one. */
  const names = held.map((item) => (item.archived_at ? `${item.name}, ${archived}` : item.name)).join("; ");
  const many = value.length > 1;
  const one = value.length === 1 ? held[0] : undefined;
  const text = many
    ? t("module.count", { count: value.length })
    : value.length === 0
      ? (placeholder ?? t("module.no_module"))
      : one?.name;
  const named = many ? `${text}: ${names}` : value.length === 0 ? text : names;
  const withText = BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant);
  const content = (
    <DropdownButton
      presentation
      className={cn("group-data-open:bg-layer-transparent-active", buttonClassName)}
      isActive={false}
      tooltipHeading={t("common.modules")}
      tooltipContent={value.length === 0 ? text : names}
      showTooltip={showTooltip}
      variant={buttonVariant}
    >
      <>
        {!hideIcon &&
          (isInitializing ? (
            <Spinner className="size-3 flex-shrink-0" />
          ) : (
            <ModuleOutline className="size-3 flex-shrink-0" />
          ))}
        {withText && text && <ModuleName name={text} caption={one && captionOf(one)} />}
        {withText && many && <span className="sr-only">: {names}</span>}
        {dropdownArrow && (
          <ChevronDownOutline className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)} aria-hidden="true" />
        )}
      </>
    </DropdownButton>
  );
  const option = (item: Module) => ({
    value: item.id,
    query: item.name,
    content: (
      <div className="flex items-center gap-2">
        <ModuleOutline className="size-3.5 flex-shrink-0" />
        <ModuleName name={item.name} caption={captionOf(item)} />
      </div>
    ),
  });
  // The archived modules it is in, after the options, which never have them: picked, they only take it out.
  const options = props.options && [...props.options, ...held.filter((item) => item.archived_at)].map(option);
  return (
    <CustomSearchSelect
      multiple
      value={[...value]}
      onChange={(picked: string[]) => {
        const added = picked.find((moduleId) => !value.includes(moduleId));
        const removed = value.find((moduleId) => !picked.includes(moduleId));
        if (added) onChange({ add: [added] });
        else if (removed) onChange({ remove: [removed] });
      }}
      options={isInitializing ? undefined : readStatus ? [] : options}
      footerOption={readStatus}
      onOpen={onDropdownOpen}
      disabled={disabled}
      tabIndex={tabIndex}
      customButton={
        <>
          <span className="sr-only">{t("common.modules")}: </span>
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
