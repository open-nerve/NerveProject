/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { PillButton } from "@makeplane/propel/components/pill";
import { CloseOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import type { ModuleFilters } from "@nerve/utils";
// components
import { AppliedDateFilters } from "@/components/planning/applied-date-filters";
// hooks
import { useModuleFilters } from "@/hooks/store/use-module-filters";
// store
import type { ModuleListKind } from "@/store/planning/module_filter.store";
// local imports
import { AppliedMembersFilters } from "./members";
import { AppliedStatusFilters } from "./status";

type Props = {
  projectId: string;
  /** The project's list the filters are of: its modules list, or its archive. */
  list: ModuleListKind;
};

/** The filters of a project's modules list, each by its name in the page, and what its options are. */
const FILTERS = [
  { key: "status", i18n_title: "project_module.status", kind: "status" },
  { key: "lead", i18n_title: "lead", kind: "members" },
  { key: "members", i18n_title: "members", kind: "members" },
  { key: "start_date", i18n_title: "start_date", kind: "days" },
  { key: "target_date", i18n_title: "due_date", kind: "days" },
] as const satisfies { key: keyof ModuleFilters; i18n_title: string; kind: string }[];

/**
 * The filters applied to a project's modules list or archive (P6a spec 2.4), the session's of the project and the
 * list: each filter by its name, each of its options removable, each filter as a whole too, and all of them at once;
 * nothing while none is applied. They are the viewer's own, whatever his role.
 */
export const ModuleAppliedFiltersList = observer(function ModuleAppliedFiltersList(props: Props) {
  const { projectId, list } = props;
  // hooks
  const { t } = useTranslation();
  const { getFilters, updateFilters, clearAllFilters } = useModuleFilters();
  const filters = getFilters(projectId, list);
  const applied = FILTERS.filter(({ key }) => (filters[key]?.length ?? 0) > 0);

  if (applied.length === 0) return null;

  return (
    <div className="flex flex-wrap items-stretch gap-2 bg-surface-1">
      {applied.map((filter) => {
        const title = t(filter.i18n_title);
        return (
          <div
            key={filter.key}
            className="my-auto flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border border-subtle p-1.5 text-11 text-tertiary"
          >
            <span className="text-11 text-tertiary">{title}</span>
            <div className="flex flex-wrap items-center gap-1">
              {filter.kind === "status" ? (
                <AppliedStatusFilters
                  values={filters.status ?? []}
                  handleRemove={(status) =>
                    updateFilters(projectId, list, { status: filters.status?.filter((one) => one !== status) })
                  }
                />
              ) : filter.kind === "members" ? (
                <AppliedMembersFilters
                  values={filters[filter.key] ?? []}
                  handleRemove={(member) =>
                    updateFilters(projectId, list, {
                      [filter.key]: filters[filter.key]?.filter((one) => one !== member),
                    })
                  }
                />
              ) : (
                <AppliedDateFilters
                  values={filters[filter.key] ?? []}
                  handleRemove={(day) =>
                    updateFilters(projectId, list, { [filter.key]: filters[filter.key]?.filter((one) => one !== day) })
                  }
                />
              )}
              <button
                type="button"
                className="grid place-items-center text-tertiary hover:text-secondary"
                aria-label={t("common.remove_item", { name: title })}
                onClick={() => updateFilters(projectId, list, { [filter.key]: [] })}
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
