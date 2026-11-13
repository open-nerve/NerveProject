/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { CloseOutline, SearchOutline } from "@makeplane/propel/icons";
// nerve imports
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
import { IconButton } from "@nerve/propel/icon-button";

type Props = {
  /** The search of the list, the caller's: the session's filters keep it for the project's list. */
  query: string;
  /** Keeps the search typed, or "" as it is cleared. */
  onSearch: (query: string) => void;
};

/**
 * The search of a project's cycles or modules list or archive, in its header (P5a spec 2.3, P6a spec 2.4). It opens on
 * its button, the caret going into it, and stays open on a search kept from before; Escape clears it, then closes it,
 * and so does a click elsewhere on an empty one; its clear button empties and closes it.
 */
export function PlanningListSearch({ query, onSearch }: Props) {
  // refs
  const inputRef = useRef<HTMLInputElement>(null);
  // hooks
  const { t } = useTranslation();
  // states
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  // outside click detector hook
  useOutsideClickDetector(inputRef, () => {
    if (isSearchOpen && query.trim() === "") setIsSearchOpen(false);
  });
  // the search opened on its button: the caret goes into it
  useEffect(() => {
    if (isSearchOpen) inputRef.current?.focus();
  }, [isSearchOpen]);

  const closeSearch = () => {
    onSearch("");
    setIsSearchOpen(false);
  };
  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Escape") return;
    if (query.trim() !== "") onSearch("");
    else {
      setIsSearchOpen(false);
      inputRef.current?.blur();
    }
  };

  if (!isSearchOpen && query.trim() === "")
    return (
      <IconButton
        variant="ghost"
        size="lg"
        onClick={() => setIsSearchOpen(true)}
        icon={SearchOutline}
        aria-label={t("common.search.label")}
      />
    );
  return (
    <div className="ml-auto flex w-64 items-center justify-start gap-1 overflow-hidden rounded-md border border-subtle bg-surface-1 px-2.5 py-1.5 text-placeholder opacity-100 transition-[width] ease-linear">
      <SearchOutline className="h-3.5 w-3.5" aria-hidden="true" />
      <input
        ref={inputRef}
        className="w-full max-w-[234px] border-none bg-transparent text-13 text-primary placeholder:text-placeholder focus:outline-none"
        placeholder={t("common.search.label")}
        aria-label={t("common.search.label")}
        value={query}
        onChange={(e) => onSearch(e.target.value)}
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
  );
}
