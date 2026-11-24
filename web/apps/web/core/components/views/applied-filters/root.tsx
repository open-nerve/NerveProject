/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@nerve/i18n";
import { PillButton } from "@makeplane/propel/components/pill";
import { CloseOutline } from "@makeplane/propel/icons";
// nerve imports
import { replaceUnderscoreIfSnakeCase } from "@nerve/utils";
// components
import { AppliedDateFilters } from "@/components/common/applied-filters/date";
import { AppliedMembersFilters } from "@/components/common/applied-filters/members";
// local imports
import type { ViewListFilters } from "../view-list-order";
import { NO_VIEW_FILTERS } from "../view-list-order";

type Props = {
  /** The filters applied to the list. */
  filters: ViewListFilters;
  /** Applies these filters instead. */
  onChange: (filters: ViewListFilters) => void;
};

/** The filters of the list, in the order they show. */
const FILTERS: (keyof ViewListFilters)[] = ["owned_by", "created_at"];

/**
 * The filters applied to a project's views list (P6a spec 2.3): each filter applied, with its values, each of which
 * the person removes, or the filter whole; and a clear of them all. None applied, it shows nothing.
 */
export function ViewAppliedFiltersList(props: Props) {
  const { filters, onChange } = props;
  const { t } = useTranslation();

  const applied = FILTERS.filter((key) => filters[key].length > 0);
  if (applied.length === 0) return null;

  /** The filters without value of the filter of key, or without the filter whole. */
  const remove = (key: keyof ViewListFilters, value: string | null) =>
    onChange({ ...filters, [key]: value === null ? [] : filters[key].filter((each) => each !== value) });

  return (
    <div className="flex flex-wrap items-stretch gap-2 bg-surface-1">
      {applied.map((key) => (
        <div
          key={key}
          className="my-auto flex min-h-9 cursor-pointer flex-wrap items-center gap-1.5 rounded-md border border-subtle p-1.5 text-11 text-tertiary capitalize hover:text-secondary"
        >
          <span className="text-11 text-tertiary">{replaceUnderscoreIfSnakeCase(key)}</span>
          {key === "created_at" ? (
            <AppliedDateFilters editable handleRemove={(value) => remove(key, value)} values={filters[key]} />
          ) : (
            <AppliedMembersFilters editable handleRemove={(value) => remove(key, value)} values={filters[key]} />
          )}
          <button
            type="button"
            className="grid place-items-center text-tertiary hover:text-secondary"
            onClick={() => remove(key, null)}
          >
            <CloseOutline height={12} width={12} />
          </button>
        </div>
      ))}
      <PillButton
        type="button"
        size="md"
        variant="outline"
        label={t("common.clear_all")}
        endIcon={<CloseOutline height={12} width={12} />}
        onClick={() => onChange(NO_VIEW_FILTERS)}
      />
    </div>
  );
}
