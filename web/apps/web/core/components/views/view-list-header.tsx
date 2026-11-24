/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
// icons
import { CloseOutline, FilterOutline, SearchOutline } from "@makeplane/propel/icons";
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
// helpers
import { cn } from "@nerve/utils";
// hooks
import { useMember } from "@/hooks/store/use-member";
// local imports
import { FiltersDropdown } from "../issues/issue-layouts/filters";
import { ViewFiltersSelection } from "./filters/filter-selection";
import { ViewOrderByDropdown } from "./filters/order-by";
import { useViewsPageState } from "./page-state";

type Props = {
  projectId: string;
};

/**
 * The view of a project's views list in its header (P6a spec 2.3): the search of their names, their order, and their
 * filters, whose owners are the project's members, its guests too (getProjectMemberIds); the page's own
 * (useViewsPageState).
 */
export const ViewListHeader = observer(function ViewListHeader(props: Props) {
  const { projectId } = props;
  const { t } = useTranslation();
  // states
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  // refs
  const inputRef = useRef<HTMLInputElement>(null);
  // store hooks
  const { choices, setChoices } = useViewsPageState();
  const {
    project: { getProjectMemberIds },
  } = useMember();
  const search = (text: string) => setChoices((current) => ({ ...current, search: text }));

  // handlers
  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      if (choices.search.trim() !== "") {
        search("");
      } else {
        setIsSearchOpen(false);
        inputRef.current?.blur();
      }
    }
  };

  // outside click detector hook
  useOutsideClickDetector(inputRef, () => {
    if (isSearchOpen && choices.search.trim() === "") setIsSearchOpen(false);
  });

  useEffect(() => {
    if (choices.search.trim() !== "") setIsSearchOpen(true);
  }, [choices.search]);

  return (
    <div className="flex h-full items-center gap-2">
      <div className="flex items-center">
        {!isSearchOpen && (
          <IconButton
            variant="ghost"
            size="lg"
            className="-mr-1"
            onClick={() => {
              setIsSearchOpen(true);
              inputRef.current?.focus();
            }}
            icon={SearchOutline}
            aria-label={t("common.search.label")}
          />
        )}
        <div
          className={cn(
            "ml-auto flex w-0 items-center justify-start gap-1 overflow-hidden rounded-md border border-transparent bg-surface-1 text-placeholder opacity-0 transition-[width] ease-linear",
            {
              "w-64 border-subtle px-2.5 py-1.5 opacity-100": isSearchOpen,
            }
          )}
        >
          <SearchOutline className="h-3.5 w-3.5" />
          <input
            ref={inputRef}
            className="w-full max-w-[234px] border-none bg-transparent text-13 text-primary placeholder:text-placeholder focus:outline-none"
            placeholder={t("common.search.label")}
            value={choices.search}
            onChange={(e) => search(e.target.value)}
            onKeyDown={handleInputKeyDown}
          />
          {isSearchOpen && (
            <button
              type="button"
              className="grid place-items-center"
              onClick={() => {
                search("");
                setIsSearchOpen(false);
              }}
            >
              <CloseOutline className="h-3 w-3" />
            </button>
          )}
        </div>
      </div>
      <div className="hidden items-center gap-2 md:flex">
        <ViewOrderByDropdown
          sortBy={choices.sortBy}
          sortKey={choices.sortKey}
          onChange={({ key, order }) =>
            setChoices((current) => ({ ...current, sortKey: key ?? current.sortKey, sortBy: order ?? current.sortBy }))
          }
        />
        <FiltersDropdown icon={<FilterOutline className="h-3 w-3" />} placement="bottom-end" isFiltersApplied={false}>
          <ViewFiltersSelection
            filters={choices.filters}
            onChange={(filters) => setChoices((current) => ({ ...current, filters }))}
            memberIds={getProjectMemberIds(projectId, true) ?? undefined}
          />
        </FiltersDropdown>
      </div>
    </div>
  );
});
