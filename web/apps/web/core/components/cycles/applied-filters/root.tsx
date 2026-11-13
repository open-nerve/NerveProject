/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { PillButton } from "@makeplane/propel/components/pill";
import { CloseOutline } from "@makeplane/propel/icons";
import type { CycleFilters } from "@nerve/utils";
// components
import { AppliedDateFilters } from "@/components/planning/applied-date-filters";
// hooks
import { useCycleFilters } from "@/hooks/store/use-cycle-filters";
// store
import type { CycleList } from "@/store/planning/cycle_filter.store";
// local imports
import { AppliedStatusFilters } from "./status";

type Props = {
  projectId: string;
  /** The project's list the filters are of: its cycles list, or its archive. */
  list: CycleList;
};

/** The filters of a project's cycles list, each by its name in the page: nerve's status, and the two days. */
const FILTERS = [
  { key: "status", i18n_title: "project_cycles.filters.status" },
  { key: "start_date", i18n_title: "start_date" },
  { key: "end_date", i18n_title: "end_date" },
] as const satisfies { key: keyof CycleFilters; i18n_title: string }[];

/**
 * The filters applied to a project's cycles list or archive (P5a spec 2.3), the session's of the project and the list:
 * each filter by its name, each of its options removable, each filter as a whole too, and all of them at once; nothing
 * while none is applied. They are the viewer's own, whatever his role.
 */
export const CycleAppliedFiltersList = observer(function CycleAppliedFiltersList(props: Props) {
  const { projectId, list } = props;
  // hooks
  const { t } = useTranslation();
  const { getFilters, updateFilters, clearAllFilters } = useCycleFilters();
  const filters = getFilters(projectId, list);
  const applied = FILTERS.filter(({ key }) => (filters[key]?.length ?? 0) > 0);

  if (applied.length === 0) return null;

  return (
    <div className="flex flex-wrap items-stretch gap-2 bg-surface-1">
      {applied.map(({ key, i18n_title }) => {
        const title = t(i18n_title);
        return (
          <div
            key={key}
            className="my-auto flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border border-subtle p-1.5 text-11 text-tertiary"
          >
            <span className="text-11 text-tertiary">{title}</span>
            <div className="flex flex-wrap items-center gap-1">
              {key === "status" ? (
                <AppliedStatusFilters
                  values={filters.status ?? []}
                  handleRemove={(status) =>
                    updateFilters(projectId, list, { status: filters.status?.filter((one) => one !== status) })
                  }
                />
              ) : (
                <AppliedDateFilters
                  values={filters[key] ?? []}
                  handleRemove={(day) =>
                    updateFilters(projectId, list, { [key]: filters[key]?.filter((one) => one !== day) })
                  }
                />
              )}
              <button
                type="button"
                className="grid place-items-center text-tertiary hover:text-secondary"
                aria-label={t("common.remove_item", { name: title })}
                onClick={() => updateFilters(projectId, list, { [key]: [] })}
              >
                <CloseOutline height={12} width={12} aria-hidden="true" />
              </button>
            </div>
          </div>
        );
      })}
      <PillButton
        type="button"
        size="md"
        variant="outline"
        label={t("common.clear_all")}
        endIcon={<CloseOutline height={12} width={12} />}
        onClick={() => clearAllFilters(projectId, list)}
      />
    </div>
  );
});
