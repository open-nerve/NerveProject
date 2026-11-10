/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { CloseOutline, FilterOutline, SearchOutline } from "@makeplane/propel/icons";
// nerve imports
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
import { calculateTotalFilters } from "@nerve/utils";
// components
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
// hooks
import { useCycleFilters } from "@/hooks/store/use-cycle-filters";
// store
import type { CycleList } from "@/store/planning/cycle_filter.store";
// local imports
import { CycleFiltersSelection } from "./dropdowns";

type Props = {
  projectId: string;
  /** The project's list the search and the filters are of: its cycles list, or its archive. */
  list: CycleList;
};

/**
 * The search and the filters' menu of a project's cycles list or archive (P5a spec 2.3, 2.5), kept by the project and
 * the list in the session's filters. The search opens on its button, and stays open on a search kept from before;
 * Escape clears it, then closes it, and so does a click elsewhere on an empty one.
 */
export const CyclesViewHeader = observer(function CyclesViewHeader(props: Props) {
  const { projectId, list } = props;
  // refs
  const inputRef = useRef<HTMLInputElement>(null);
  // hooks
  const { t } = useTranslation();
  const { getFilters, getSearchQuery, updateSearchQuery } = useCycleFilters();
  const searchQuery = getSearchQuery(projectId, list);
  // states
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const searchShown = isSearchOpen || searchQuery.trim() !== "";
  // outside click detector hook
  useOutsideClickDetector(inputRef, () => {
    if (isSearchOpen && searchQuery.trim() === "") setIsSearchOpen(false);
  });
  // the search opened on its button: the caret goes into it
  useEffect(() => {
    if (isSearchOpen) inputRef.current?.focus();
  }, [isSearchOpen]);

  const search = (query: string) => updateSearchQuery(projectId, list, query);
  const closeSearch = () => {
    search("");
    setIsSearchOpen(false);
  };
  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Escape") return;
    if (searchQuery.trim() !== "") search("");
    else {
      setIsSearchOpen(false);
      inputRef.current?.blur();
    }
  };

  return (
    <div className="flex items-center gap-2">
      {!searchShown ? (
        <IconButton
          variant="ghost"
          size="lg"
          onClick={() => setIsSearchOpen(true)}
          icon={SearchOutline}
          aria-label={t("common.search.label")}
        />
      ) : (
        <div className="ml-auto flex w-64 items-center justify-start gap-1 overflow-hidden rounded-md border border-subtle bg-surface-1 px-2.5 py-1.5 text-placeholder opacity-100 transition-[width] ease-linear">
          <SearchOutline className="h-3.5 w-3.5" aria-hidden="true" />
          <input
            ref={inputRef}
            className="w-full max-w-[234px] border-none bg-transparent text-13 text-primary placeholder:text-placeholder focus:outline-none"
            placeholder={t("common.search.label")}
            aria-label={t("common.search.label")}
            value={searchQuery}
            onChange={(e) => search(e.target.value)}
            onKeyDown={handleInputKeyDown}
          />
          <button
            type="button"
            className="grid place-items-center"
            aria-label={t("common.search.clear")}
            onClick={closeSearch}
          >
            <CloseOutline className="h-3 w-3" aria-hidden="true" />
          </button>
        </div>
      )}

      <FiltersDropdown
        icon={<FilterOutline className="h-3 w-3" />}
        title={t("common.filters")}
        placement="bottom-end"
        isFiltersApplied={calculateTotalFilters(getFilters(projectId, list)) !== 0}
      >
        <CycleFiltersSelection projectId={projectId} list={list} />
      </FiltersDropdown>
    </div>
  );
});
