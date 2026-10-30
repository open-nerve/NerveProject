import { Field, FieldError } from "@makeplane/propel/components/field";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { DirectoryReadError, useDirectoryRead } from "@/components/issues/issue-detail-widgets/directory-read";
import { Fragment, useState, useId, useRef } from "react";
import { observer } from "mobx-react";
import { usePopper } from "react-popper";
import { AddOutline, LoadingOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
import { Combobox } from "@headlessui/react";
// nerve imports
import { EUserPermissionsLevel, getRandomLabelColor } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { IssueLabelCreation } from "@/components/labels/use-label-changes";
import { EUserProjectRoles } from "@nerve/types";
// helpers
import { getTabIndex, isComposing } from "@nerve/utils";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useUserPermissions } from "@/hooks/store/user";
import { usePlatformOS } from "@/hooks/use-platform-os";
//constants
export interface IIssueLabelSelect {
  disabled?: boolean;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  values: string[];
  onSelect: (_labelIds: string[]) => Promise<void>;
  onAddLabel: IssueLabelCreation;
}

export const IssueLabelSelect = observer(function IssueLabelSelect(props: IIssueLabelSelect) {
  const { disabled, workspaceSlug, projectId, issueId, values, onSelect, onAddLabel } = props;
  const { t } = useTranslation();
  // store hooks
  const { isMobile } = usePlatformOS();
  const { fetchProjectLabels, getProjectLabels } = useLabel();
  const { allowPermissions } = useUserPermissions();
  // states
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const [query, setQuery] = useState("");
  const [nameError, setNameError] = useState<string>();
  const errorId = useId();
  const pending = useRef(false);
  const [submitting, setSubmitting] = useState<boolean>(false);

  const canCreateLabel =
    projectId && allowPermissions([EUserProjectRoles.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, projectId);

  const projectLabels = getProjectLabels(projectId);
  const read = useDirectoryRead(
    workspaceSlug,
    projectId ?? undefined,
    projectId ? () => fetchProjectLabels(projectId) : undefined
  );

  const { baseTabIndex } = getTabIndex(undefined, isMobile);

  const options = (projectLabels ?? []).map((label) => ({
    value: label.id,
    query: label.name,
    content: (
      <div className="flex items-center justify-start gap-2 overflow-hidden">
        <span
          className="h-2.5 w-2.5 flex-shrink-0 rounded-full"
          style={{
            backgroundColor: label.color,
          }}
        />
        <div className="line-clamp-1 inline-block truncate">{label.name}</div>
      </div>
    ),
  }));

  const filteredOptions =
    query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase()));

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: "bottom-start",
    modifiers: [
      {
        name: "preventOverflow",
        options: {
          padding: 12,
        },
      },
    ],
  });

  const issueLabels = values ?? [];

  const label = <span className="text-body-xs-medium text-placeholder">{t("label.select")}</span>;

  const searchInputKeyDown = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (query !== "" && e.key === "Escape") {
      e.stopPropagation();
      setQuery("");
    }

    if (
      !read.loading &&
      !read.error &&
      query !== "" &&
      e.key === "Enter" &&
      !isComposing(e.nativeEvent) &&
      canCreateLabel
    ) {
      e.stopPropagation();
      e.preventDefault();
      await handleAddLabel(query);
    }
  };

  const handleAddLabel = async (labelName: string) => {
    if (disabled || pending.current || !labelName.trim()) return;
    pending.current = true;
    setSubmitting(true);
    setNameError(undefined);
    try {
      await onAddLabel(
        { name: labelName.trim(), color: getRandomLabelColor() },
        (_field, message) => setNameError(message),
        async (created) => {
          try {
            await onSelect([...values, created.id]);
            setQuery("");
          } catch {
            // Attachment reports its own refusal; keep the query and confirmed label available.
          }
        }
      );
    } finally {
      pending.current = false;
      setSubmitting(false);
    }
  };

  if (!issueId || !values) return <></>;

  return (
    <>
      <Combobox
        disabled={disabled || submitting}
        as="div"
        className="size-full flex-shrink-0 text-left"
        value={issueLabels}
        onChange={(value) => void onSelect(value).catch(() => undefined)}
        multiple
      >
        <Combobox.Button as={Fragment}>
          <Button
            ref={setReferenceElement}
            type="button"
            variant="tertiary"
            size="sm"
            prependIcon={<AddOutline />}
            onClick={read.load}
          >
            {label}
          </Button>
        </Combobox.Button>

        <Combobox.Options as="ul" className="fixed z-10">
          <div
            className="z-10 my-1 w-48 rounded-sm border border-strong bg-surface-1 py-2.5 text-11 whitespace-nowrap shadow-raised-200 focus:outline-none"
            ref={setPopperElement}
            style={styles.popper}
            {...attributes.popper}
          >
            <div className="px-2">
              <Field name="name" invalid={!!nameError}>
                <div className="flex w-full items-center justify-start rounded-sm border border-subtle bg-surface-2 px-2">
                  <SearchOutline className="h-3.5 w-3.5 text-tertiary" />
                  <Combobox.Input
                    aria-invalid={!!nameError}
                    aria-describedby={nameError ? errorId : undefined}
                    className="w-full bg-transparent px-2 py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                    value={query}
                    onChange={(e) => {
                      setQuery(e.target.value);
                      setNameError(undefined);
                    }}
                    placeholder={t("common.search.label")}
                    displayValue={(assigned: any) => assigned?.name}
                    onKeyDown={searchInputKeyDown}
                    tabIndex={baseTabIndex}
                  />
                </div>
                {nameError && (
                  <FieldError match size="lg" id={errorId}>
                    {nameError}
                  </FieldError>
                )}
              </Field>
            </div>
            <div className="vertical-scrollbar mt-2 scrollbar-sm max-h-48 overflow-y-scroll px-2 pr-0">
              {read.error ? (
                <DirectoryReadError error={read.error} retry={read.load} />
              ) : read.loading ? (
                <p className="text-center text-secondary">{t("common.loading")}</p>
              ) : filteredOptions.length > 0 ? (
                <ul className="space-y-1">
                  {filteredOptions.map((option) => (
                    <Combobox.Option
                      as="li"
                      key={option.value}
                      value={option.value}
                      className={({ selected }) =>
                        `flex cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none hover:bg-layer-1 ${
                          selected ? "text-primary" : "text-secondary"
                        }`
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
                <LoadingOutline className="spin h-3.5 w-3.5" />
              ) : canCreateLabel ? (
                <div className="space-y-1">
                  <button
                    disabled={disabled || submitting}
                    type="button"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      if (!query.length) return;
                      handleAddLabel(query);
                    }}
                    className={`text-left text-secondary ${query.length ? "cursor-pointer" : "cursor-default"}`}
                  >
                    {query.length ? (
                      <>
                        {/* TODO: Translate here */}+ Add <span className="text-primary">&quot;{query}&quot;</span> to
                        labels
                      </>
                    ) : (
                      t("label.create.type")
                    )}
                  </button>
                </div>
              ) : (
                <p className="text-left text-secondary">{t("common.search.no_matching_results")}</p>
              )}
            </div>
          </div>
        </Combobox.Options>
      </Combobox>
    </>
  );
});
