/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { CloseOutline, FilterOutline, SearchOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// nerve imports
import { MODULE_VIEW_LAYOUTS } from "@nerve/constants";
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
import { calculateTotalFilters, cn } from "@nerve/utils";
// components
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
// hooks
import { useModuleFilters } from "@/hooks/store/use-module-filters";
import { usePlatformOS } from "@/hooks/use-platform-os";
// store
import type { ModuleListKind } from "@/store/planning/module_filter.store";
// local imports
import { ModuleFiltersSelection } from "./dropdowns/filters/root";
import { ModuleOrderByDropdown } from "./dropdowns/order-by";
import { ModuleLayoutIcon } from "./module-layout-icon";
import { useModuleDisplay } from "./use-module-display";

type Props = {
  /** The workspace of the page's address. */
  workspaceSlug: string;
  projectId: string;
  /** The project's list the view is of: its modules list, or its archive. */
  list: ModuleListKind;
};

/**
 * The view of a project's modules list or archive, in its header (P6a spec 2.4, 2.6, R4, R14), for everyone who sees
 * it: its search, its order, its filters, and the list's layout. The search and the filters are the project's own of
 * the list, kept by the session's filters (ModuleFilterStore): the search opens on its button, and stays open on a
 * search kept from before; Escape clears it, then closes it, and so does a click elsewhere on an empty one. The order
 * and the layout are the browser's, kept for the project (useModuleDisplay): the list and the archive share the order,
 * which the order's menu sets, by the keyboard too; the archive has no layout to choose.
 */
export const ModuleViewHeader = observer(function ModuleViewHeader({ workspaceSlug, projectId, list }: Props) {
  // refs
  const inputRef = useRef<HTMLInputElement>(null);
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const { layout, orderBy, setLayout, setOrderBy } = useModuleDisplay(projectId);
  // store hooks
  const { getFilters, getSearchQuery, updateSearchQuery } = useModuleFilters();
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
    <div className={cn("h-full items-center gap-2 self-end", list === "default" ? "hidden sm:flex" : "flex")}>
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
      <ModuleOrderByDropdown value={orderBy} onChange={setOrderBy} />
      <FiltersDropdown
        icon={<FilterOutline className="h-3 w-3" />}
        title={t("common.filters")}
        placement="bottom-end"
        isFiltersApplied={calculateTotalFilters(getFilters(projectId, list)) !== 0}
      >
        <ModuleFiltersSelection workspaceSlug={workspaceSlug} projectId={projectId} list={list} />
      </FiltersDropdown>
      {list === "default" && (
        <div className="hidden items-center gap-1 rounded-sm bg-layer-3 p-1 md:flex">
          {MODULE_VIEW_LAYOUTS.map((one) => (
            <Tooltip key={one.key} label={t(one.i18n_title)} disabled={isMobile}>
              <button
                type="button"
                aria-label={t(one.i18n_title)}
                aria-pressed={layout === one.key}
                className={cn(
                  "group grid h-5.5 w-7 place-items-center overflow-hidden rounded-sm transition-all hover:bg-layer-transparent-hover",
                  { "bg-layer-transparent-active hover:bg-layer-transparent-active": layout === one.key }
                )}
                onClick={() => setLayout(one.key)}
              >
                <ModuleLayoutIcon layout={one.key} />
              </button>
            </Tooltip>
          ))}
        </div>
      )}
    </div>
  );
});
