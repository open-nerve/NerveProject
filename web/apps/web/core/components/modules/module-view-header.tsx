/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useCallback, useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { CloseOutline, FilterOutline, SearchOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// nerve imports
import { MODULE_VIEW_LAYOUTS } from "@nerve/constants";
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";
import type { TModuleFilters } from "@nerve/types";
import { cn, calculateTotalFilters } from "@nerve/utils";
// components
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useModuleFilter } from "@/hooks/store/use-module-filter";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { ModuleFiltersSelection } from "./dropdowns/filters";
import { ModuleOrderByDropdown } from "./dropdowns/order-by";
import { ModuleLayoutIcon } from "./module-layout-icon";
import { useModuleDisplay } from "./use-module-display";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The view of a project's modules list, in its header (P6a spec 2.4, 2.6, R4, R14), for everyone who sees the list:
 * its search, its order, its filters and its layout. The order and the layout are the browser's, kept for the project
 * (useModuleDisplay), which the list follows; the order's menu is the keyboard's too. The filters' lead and members
 * are the members of the route's workspace (getWorkspaceMemberIds, by the address's slug). The search and the filters
 * themselves are still the old store's (T11 moves them to the session's filters, which the list reads).
 */
export const ModuleViewHeader = observer(function ModuleViewHeader({ workspaceSlug, projectId }: Props) {
  // refs
  const inputRef = useRef<HTMLInputElement>(null);
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const { layout, orderBy, setLayout, setOrderBy } = useModuleDisplay(projectId);
  // store hooks
  const {
    workspace: { getWorkspaceMemberIds },
  } = useMember();
  const {
    currentProjectDisplayFilters: displayFilters,
    currentProjectFilters: filters,
    searchQuery,
    updateDisplayFilters,
    updateFilters,
    updateSearchQuery,
  } = useModuleFilter();

  // states
  const [isSearchOpen, setIsSearchOpen] = useState(searchQuery !== "");

  // handlers
  const handleFilters = useCallback(
    (key: keyof TModuleFilters, value: string | string[]) => {
      const newValues = filters?.[key] ?? [];

      if (Array.isArray(value))
        value.forEach((val) => {
          if (!newValues.includes(val)) newValues.push(val);
          else newValues.splice(newValues.indexOf(val), 1);
        });
      else {
        if (filters?.[key]?.includes(value)) newValues.splice(newValues.indexOf(value), 1);
        else newValues.push(value);
      }

      updateFilters(projectId, { [key]: newValues });
    },
    [filters, projectId, updateFilters]
  );

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      if (searchQuery && searchQuery.trim() !== "") updateSearchQuery("");
      else {
        setIsSearchOpen(false);
        inputRef.current?.blur();
      }
    }
  };

  // outside click detector hook
  useOutsideClickDetector(inputRef, () => {
    if (isSearchOpen && searchQuery.trim() === "") setIsSearchOpen(false);
  });

  useEffect(() => {
    if (searchQuery.trim() !== "") setIsSearchOpen(true);
  }, [searchQuery]);

  const isFiltersApplied = calculateTotalFilters(filters ?? {}) !== 0 || displayFilters?.favorites;

  return (
    <div className="hidden h-full items-center gap-2 self-end sm:flex">
      <div className="flex items-center">
        {!isSearchOpen && (
          <IconButton
            variant="ghost"
            size="lg"
            className="p- -mr-1"
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
            placeholder="Search"
            value={searchQuery}
            onChange={(e) => updateSearchQuery(e.target.value)}
            onKeyDown={handleInputKeyDown}
          />
          {isSearchOpen && (
            <button
              type="button"
              className="grid place-items-center"
              onClick={() => {
                setIsSearchOpen(false);
              }}
            >
              <CloseOutline className="h-3 w-3" />
            </button>
          )}
        </div>
      </div>
      <ModuleOrderByDropdown value={orderBy} onChange={setOrderBy} />
      <FiltersDropdown
        icon={<FilterOutline className="h-3 w-3" />}
        title="Filters"
        placement="bottom-end"
        isFiltersApplied={isFiltersApplied}
      >
        <ModuleFiltersSelection
          displayFilters={displayFilters ?? {}}
          filters={filters ?? {}}
          handleDisplayFiltersUpdate={(val) => updateDisplayFilters(projectId, val)}
          handleFiltersUpdate={handleFilters}
          memberIds={getWorkspaceMemberIds(workspaceSlug)}
        />
      </FiltersDropdown>
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
    </div>
  );
});
