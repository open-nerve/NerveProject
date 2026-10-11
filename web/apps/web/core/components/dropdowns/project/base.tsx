/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, useState } from "react";
import { observer } from "mobx-react";
import { usePopper } from "react-popper";
import { Combobox } from "@headlessui/react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import { ProjectsOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
import { ComboDropDown } from "@nerve/ui";
import { cn, sortBySelectedFirst } from "@nerve/utils";
// hooks
import { useDropdown } from "@/hooks/use-dropdown";
// nerve imports
import type { Project } from "@nerve/api-client";
// local imports
import { DropdownButton } from "../buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "../constants";
import type { TDropdownProps } from "../types";

type Props = TDropdownProps & {
  getProjectById: (projectId: string | null | undefined) => Project | undefined;
  projectIds: string[];
  renderCondition?: (projectId: string) => boolean;
} & (
    | {
        multiple: false;
        onChange: (val: string) => void;
        value: string | null;
      }
    | {
        multiple: true;
        onChange: (val: string[]) => void;
        value: string[];
      }
  );

/** A project's logo, as the dropdown's button shows it. */
const renderProjectLogo = (logoProps: Project["logo_props"]) => (
  <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
    <Logo logo={logoProps} size={14} />
  </span>
);

export const ProjectDropdownBase = observer(function ProjectDropdownBase(props: Props) {
  const {
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className = "",
    disabled = false,
    getProjectById,
    hideIcon = false,
    multiple,
    onChange,
    placeholder = "Project",
    placement,
    projectIds,
    renderCondition,
    showTooltip = false,
    tabIndex,
    value,
  } = props;
  // refs
  const dropdownRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  // popper-js refs
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  // states
  const [query, setQuery] = useState("");
  const [isOpen, setIsOpen] = useState(false);
  // nerve hooks
  const { t } = useTranslation();
  // popper-js init
  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
    modifiers: [
      {
        name: "preventOverflow",
        options: {
          padding: 12,
        },
      },
    ],
  });
  // store hooks
  const options = projectIds?.map((projectId) => {
    const projectDetails = getProjectById(projectId);
    if (renderCondition && !renderCondition(projectId)) return;
    return {
      value: projectId,
      query: `${projectDetails?.name}`,
      content: (
        <div className="flex items-center gap-2">
          {projectDetails?.logo_props && (
            <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
              <Logo logo={projectDetails?.logo_props} size={12} />
            </span>
          )}
          <span className="flex-grow truncate">{projectDetails?.name}</span>
        </div>
      ),
    };
  });

  const filteredOptions = sortBySelectedFirst(
    (query === "" ? options : options?.filter((o) => o?.query.toLowerCase().includes(query.toLowerCase())))?.filter(
      (o): o is NonNullable<typeof o> => o !== undefined
    ),
    value
  );

  const { handleClose, handleKeyDown, handleOnClick, searchInputKeyDown } = useDropdown({
    dropdownRef,
    inputRef,
    isOpen,
    query,
    setIsOpen,
    setQuery,
  });

  const dropdownOnChange = (val: string & string[]) => {
    onChange(val);
    if (!multiple) handleClose();
  };

  // what the button says: the project picked, how many are, or the placeholder
  const getDisplayName = () => {
    if (Array.isArray(value)) {
      const firstProject = getProjectById(value[0]);
      return value.length ? (value.length === 1 ? firstProject?.name : `${value.length} projects`) : placeholder;
    } else {
      return value ? (getProjectById(value)?.name ?? placeholder) : placeholder;
    }
  };

  // the logos of the projects picked
  const getProjectIcon = () => {
    if (Array.isArray(value)) {
      return (
        <div className="flex items-center gap-0.5">
          {value.length > 0 ? (
            value.map((projectId) => {
              const projectDetails = getProjectById(projectId);
              return projectDetails?.logo_props ? renderProjectLogo(projectDetails.logo_props) : null;
            })
          ) : (
            <ProjectsOutline className="size-3 text-tertiary" />
          )}
        </div>
      );
    } else {
      const projectDetails = getProjectById(value);
      return projectDetails?.logo_props ? renderProjectLogo(projectDetails.logo_props) : null;
    }
  };

  const comboButton = (
    <button
      ref={setReferenceElement}
      type="button"
      className={cn(
        "clickable block h-full max-w-full outline-none",
        {
          "cursor-not-allowed text-secondary": disabled,
          "cursor-pointer": !disabled,
        },
        buttonContainerClassName
      )}
      onClick={handleOnClick}
      disabled={disabled}
    >
      <DropdownButton
        className={buttonClassName}
        isActive={isOpen}
        tooltipHeading="Project"
        tooltipContent={value?.length ? `${value.length} project${value.length !== 1 ? "s" : ""}` : placeholder}
        showTooltip={showTooltip}
        variant={buttonVariant}
      >
        {!hideIcon && getProjectIcon()}
        {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
          <span className="max-w-40 truncate">{getDisplayName()}</span>
        )}
      </DropdownButton>
    </button>
  );

  return (
    <ComboDropDown
      as="div"
      ref={dropdownRef}
      tabIndex={tabIndex}
      className={cn("h-full", className)}
      value={value}
      onChange={dropdownOnChange}
      disabled={disabled}
      onKeyDown={handleKeyDown}
      role="presentation"
      button={comboButton}
      multiple={multiple}
    >
      {isOpen && (
        <Combobox.Options as="ul" className="fixed z-10" static>
          <div
            className="my-1 w-48 rounded-sm border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 text-11 shadow-raised-200 focus:outline-none"
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
          >
            <div className="flex items-center gap-1.5 rounded-sm border border-subtle bg-surface-2 px-2">
              <SearchOutline className="h-3.5 w-3.5 text-placeholder" />
              <Combobox.Input
                as="input"
                ref={inputRef}
                className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("search")}
                displayValue={(assigned: any) => assigned?.name}
                onKeyDown={searchInputKeyDown}
              />
            </div>
            <div className="mt-2 max-h-48 space-y-1 overflow-y-scroll">
              {filteredOptions ? (
                filteredOptions.length > 0 ? (
                  filteredOptions.map((option) => {
                    if (!option) return;
                    return (
                      <Combobox.Option
                        as="li"
                        key={option.value}
                        value={option.value}
                        className={({ active, selected }) =>
                          `flex w-full cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none ${
                            active ? "bg-layer-transparent-hover" : ""
                          } ${selected ? "text-primary" : "text-secondary"}`
                        }
                      >
                        {({ selected }) => (
                          <>
                            <span className="flex-grow truncate">{option.content}</span>
                            {selected && <TickOutline className="h-3.5 w-3.5 flex-shrink-0" />}
                          </>
                        )}
                      </Combobox.Option>
                    );
                  })
                ) : (
                  <p className="px-1.5 py-1 text-placeholder italic">{t("no_matching_results")}</p>
                )
              ) : (
                <p className="px-1.5 py-1 text-placeholder italic">{t("loading")}</p>
              )}
            </div>
          </div>
        </Combobox.Options>
      )}
    </ComboDropDown>
  );
});
