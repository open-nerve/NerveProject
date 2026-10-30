import { Field, FieldError } from "@makeplane/propel/components/field";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DirectoryReadError, useDirectoryRead } from "@/components/issues/issue-detail-widgets/directory-read";
import { observer } from "mobx-react";
import { useCallback, useEffect, useMemo, useRef, useState, useId } from "react";
import type { Placement } from "@popperjs/core";
import { useParams } from "react-router";
import { usePopper } from "react-popper";
import { ChevronDownOutline, LoadingOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
import { Combobox } from "@headlessui/react";
// nerve imports
import { EUserPermissionsLevel, getRandomLabelColor } from "@nerve/constants";
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
// types
import type { Label } from "@nerve/api-client";
import { useLabelChanges } from "@/components/labels/use-label-changes";
import { useDetailScope } from "../../issue-detail-widgets/use-detail-scope";
import { useIssueLabels } from "../../use-issue-labels";
import { EUserProjectRoles } from "@nerve/types";
// components
import { ComboDropDown } from "@nerve/ui";
import { isComposing, sortBySelectedFirst } from "@nerve/utils";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useUserPermissions } from "@/hooks/store/user";
import { useDropdownKeyDown } from "@/hooks/use-dropdown-key-down";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useRefusalToast } from "@/hooks/use-refusal-toast";

export interface ILabelDropdownProps {
  projectId: string | null;
  value: string[];
  onChange: (data: string[]) => void;
  onClose?: () => void;
  disabled?: boolean;
  defaultOptions?: any;
  hideDropdownArrow?: boolean;
  className?: string;
  buttonClassName?: string;
  optionsClassName?: string;
  placement?: Placement;
  maxRender?: number;
  renderByDefault?: boolean;
  fullWidth?: boolean;
  fullHeight?: boolean;
  label: React.ReactNode;
}

const preventPropagation = (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
  e.stopPropagation();
  e.preventDefault();
};

const EMPTY_LABELS: Label[] = [];

export const LabelDropdown = observer(function LabelDropdown(props: ILabelDropdownProps) {
  const { workspaceSlug } = useParams();
  return <LabelDropdownContent key={JSON.stringify([workspaceSlug, props.projectId])} {...props} />;
});
const LabelDropdownContent = observer(function LabelDropdownContent(props: ILabelDropdownProps) {
  const {
    projectId,
    value,
    onChange,
    onClose,
    disabled,
    hideDropdownArrow = false,
    className,
    buttonClassName = "",
    optionsClassName = "",
    placement,
    maxRender = 2,
    renderByDefault = true,
    fullWidth = false,
    fullHeight = false,
    label: buttonLabel,
  } = props;
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();

  //router
  const { workspaceSlug } = useParams();

  //states
  const [isOpen, setIsOpen] = useState(false);

  const [query, setQuery] = useState<string>("");
  const [nameError, setNameError] = useState<string>();
  const errorId = useId();
  const pending = useRef(false);
  const [submitting, setSubmitting] = useState<boolean>(false);

  //refs
  const dropdownRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  // popper-js refs
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  //hooks
  const { fetchProjectLabels } = useLabel();
  const { isMobile } = usePlatformOS();
  const storeLabels = useIssueLabels(workspaceSlug, projectId);
  const capture = useDetailScope(workspaceSlug ?? "", projectId ?? undefined, "label-create", true);
  const { create } = useLabelChanges(capture);
  const read = useDirectoryRead(
    workspaceSlug,
    projectId ?? undefined,
    projectId ? () => fetchProjectLabels(projectId) : undefined
  );
  const { allowPermissions } = useUserPermissions();

  const canCreateLabel =
    projectId && allowPermissions([EUserProjectRoles.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, projectId);

  const projectLabels = storeLabels ?? EMPTY_LABELS;

  const options = useMemo(
    () =>
      projectLabels.map((label) => ({
        value: label?.id,
        query: label?.name,
        content: (
          <div className="flex items-center justify-start gap-2 overflow-hidden">
            <span
              className="h-2.5 w-2.5 flex-shrink-0 rounded-full"
              style={{
                backgroundColor: label?.color,
              }}
            />
            <div className="line-clamp-1 inline-block truncate">{label?.name}</div>
          </div>
        ),
      })),
    [projectLabels]
  );

  const filteredOptions = useMemo(
    () =>
      sortBySelectedFirst(
        query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase())),
        value
      ),
    [options, query, value]
  );

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

  const toggleDropdown = useCallback(() => {
    if (!isOpen) void read.load();
    setIsOpen((prevIsOpen) => !prevIsOpen);
    if (isOpen && onClose) onClose();
  }, [read, onClose, isOpen, setIsOpen]);

  const handleClose = () => {
    if (!isOpen) return;
    setIsOpen(false);
    setQuery("");
    if (onClose) onClose();
  };

  const handleAddLabel = async (labelName: string) => {
    if (!projectId || pending.current || !labelName.trim()) return;
    pending.current = true;
    setSubmitting(true);
    setNameError(undefined);
    try {
      await create(
        projectId,
        { name: labelName.trim(), color: getRandomLabelColor() },
        (_field, message) => setNameError(message),
        async (label) => {
          try {
            await onChange([...value, label.id]);
            setQuery("");
          } catch (error) {
            toastRefusal(error);
          }
        }
      );
    } finally {
      pending.current = false;
      setSubmitting(false);
    }
  };

  const searchInputKeyDown = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    e.stopPropagation();
    if (query !== "" && e.key === "Escape") {
      setQuery("");
      e.preventDefault();
    }

    if (
      !read.loading &&
      !read.error &&
      query !== "" &&
      e.key === "Enter" &&
      !isComposing(e.nativeEvent) &&
      canCreateLabel
    ) {
      e.preventDefault();
      await handleAddLabel(query);
    }
  };
  const handleKeyDown = useDropdownKeyDown(toggleDropdown, handleClose);

  const handleOnClick = useCallback(
    (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
      e.stopPropagation();
      e.preventDefault();
      toggleDropdown();
    },
    [toggleDropdown]
  );

  useEffect(() => {
    if (isOpen && inputRef.current && !isMobile) {
      inputRef.current.focus();
    }
  }, [isOpen, isMobile]);

  useOutsideClickDetector(dropdownRef, handleClose);

  const comboButton = useMemo(
    () => (
      <button
        ref={setReferenceElement}
        type="button"
        className={`clickable flex h-full w-full items-center justify-center gap-1 text-caption-sm-regular ${fullWidth && "hover:bg-layer-1"} ${
          disabled
            ? "cursor-not-allowed text-secondary"
            : value.length <= maxRender
              ? "cursor-pointer"
              : "cursor-pointer hover:bg-layer-1"
        } ${buttonClassName}`}
        onClick={handleOnClick}
        disabled={disabled}
      >
        {buttonLabel}
        {!hideDropdownArrow && !disabled && <ChevronDownOutline className="h-3 w-3" aria-hidden="true" />}
      </button>
    ),
    [
      buttonClassName,
      disabled,
      fullWidth,
      handleOnClick,
      hideDropdownArrow,
      buttonLabel,
      maxRender,
      value.length,
      setReferenceElement,
    ]
  );

  return (
    <div className={`${fullHeight ? "h-full" : "h-5"}`} onClick={preventPropagation} role="presentation">
      <ComboDropDown
        as="div"
        role="presentation"
        ref={dropdownRef}
        className={`h-full w-auto max-w-full flex-shrink-0 text-left ${className}`}
        value={value}
        onChange={onChange}
        disabled={disabled}
        onKeyDown={handleKeyDown}
        button={comboButton}
        renderByDefault={renderByDefault}
        multiple
      >
        {isOpen && (
          <Combobox.Options as="ul" className="fixed z-10" static>
            <div
              className={`z-10 my-1 h-auto w-48 rounded-sm border border-strong bg-surface-1 px-2 py-2.5 text-caption-sm-regular whitespace-nowrap shadow-raised-200 focus:outline-none ${optionsClassName}`}
              ref={setPopperElement}
              style={styles.popper}
              {...attributes.popper}
            >
              <Field name="name" invalid={!!nameError}>
                <div className="flex w-full items-center justify-start rounded-sm border border-subtle bg-surface-2 px-2">
                  <SearchOutline className="h-3.5 w-3.5 text-tertiary" />
                  <Combobox.Input
                    aria-invalid={!!nameError}
                    aria-describedby={nameError ? errorId : undefined}
                    ref={inputRef}
                    className="w-full bg-transparent px-2 py-1 text-caption-sm-regular text-secondary placeholder:text-placeholder focus:outline-none"
                    value={query}
                    onChange={(e) => {
                      setQuery(e.target.value);
                      setNameError(undefined);
                    }}
                    placeholder={t("common.search.label")}
                    displayValue={(assigned: any) => assigned?.name || ""}
                    onKeyDown={searchInputKeyDown}
                  />
                </div>
                {nameError && (
                  <FieldError match size="lg" id={errorId}>
                    {nameError}
                  </FieldError>
                )}
              </Field>
              <div className="mt-2 max-h-48 overflow-y-scroll">
                {read.error ? (
                  <DirectoryReadError error={read.error} retry={read.load} />
                ) : read.loading ? (
                  <p className="text-center text-secondary">{t("common.loading")}</p>
                ) : filteredOptions && filteredOptions.length > 0 ? (
                  <ul className="space-y-1">
                    {filteredOptions.map((option) => (
                      <Combobox.Option
                        as="li"
                        key={option.value}
                        value={option.value}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            e.preventDefault();
                            e.stopPropagation();
                          }
                        }}
                        className={({ active, selected }) =>
                          `flex cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none hover:bg-layer-1 ${
                            active ? "bg-layer-1" : ""
                          } ${selected ? "text-primary" : "text-secondary"}`
                        }
                      >
                        {({ selected }) => (
                          <>
                            {option.content}
                            {selected && (
                              <div className="flex-shrink-0">
                                <TickOutline className="h-3.5 w-3.5" />
                              </div>
                            )}
                          </>
                        )}
                      </Combobox.Option>
                    ))}
                  </ul>
                ) : submitting ? (
                  <LoadingOutline className="h-3.5 w-3.5 animate-spin" />
                ) : canCreateLabel ? (
                  <button
                    type="button"
                    onClick={() => {
                      if (!query.length) return;
                      handleAddLabel(query);
                    }}
                    className={`text-left text-secondary ${query.length ? "cursor-pointer" : "cursor-default"}`}
                  >
                    {/* TODO: translate here */}
                    {query.length ? (
                      <>
                        + Add <span className="text-primary">&quot;{query}&quot;</span> to labels
                      </>
                    ) : (
                      t("label.create.type")
                    )}
                  </button>
                ) : (
                  <p className="text-left text-secondary">{t("common.search.no_matching_results")}</p>
                )}
              </div>
            </div>
          </Combobox.Options>
        )}
      </ComboDropDown>
    </div>
  );
});
