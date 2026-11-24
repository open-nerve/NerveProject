/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@nerve/i18n";
import { PillButton } from "@makeplane/propel/components/pill";
import { CloseOutline } from "@makeplane/propel/icons";
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

/** The filters of the list, in the order they show, each by the name its panel gives it (ViewFiltersSelection). */
const FILTERS = [
  { key: "owned_by", i18n_title: "common.created_by" },
  { key: "created_at", i18n_title: "inbox_issue.filters.created_at" },
] as const satisfies { key: keyof ViewListFilters; i18n_title: string }[];

/**
 * The filters applied to a project's views list (P6a spec 2.3): each filter applied, with its values, each of which
 * the person removes, or the filter whole; and a clear of them all. None applied, it shows nothing.
 */
export function ViewAppliedFiltersList(props: Props) {
  const { filters, onChange } = props;
  const { t } = useTranslation();

  const applied = FILTERS.filter(({ key }) => filters[key].length > 0);
  if (applied.length === 0) return null;

  /** The filters without value of the filter of key, or without the filter whole. */
  const remove = (key: keyof ViewListFilters, value: string | null) =>
    onChange({ ...filters, [key]: value === null ? [] : filters[key].filter((each) => each !== value) });

  return (
    <div className="flex flex-wrap items-stretch gap-2 bg-surface-1">
      {applied.map(({ key, i18n_title }) => (
        <div
          key={key}
          className="my-auto flex min-h-9 cursor-pointer flex-wrap items-center gap-1.5 rounded-md border border-subtle p-1.5 text-11 text-tertiary hover:text-secondary"
        >
          <span className="text-11 text-tertiary">{t(i18n_title)}</span>
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
