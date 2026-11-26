/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// icons
import { ChevronDownOutline, FilterOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Row } from "@nerve/ui";
// components
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { ViewFiltersSelection } from "@/components/views/filters/filter-selection";
import { ViewOrderByDropdown } from "@/components/views/filters/order-by";
import { useViewsPageState } from "@/components/views/page-state";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { viewEntryOn } from "@/lib/views-entry";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The small screen's header of a project's views list (P6a spec 2.3), of the route's project: the list's order and its
 * filters, whose owners are the project's members, its guests too (getProjectMemberIds); the page's own state, as the
 * wide header's (useViewsPageState); for everyone the ways into the project's views are offered to (viewEntryOn).
 */
export const ViewMobileHeader = observer(function ViewMobileHeader(props: Props) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { choices, setChoices } = useViewsPageState();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const {
    project: { getProjectMemberIds },
  } = useMember();

  if (!viewEntryOn(getProjectById(projectId), getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)))
    return null;

  return (
    <div className="z-[13] flex justify-evenly border-b border-subtle bg-surface-1 py-2 md:hidden">
      <Row className="flex flex-grow items-center justify-center border-l border-subtle text-13 text-secondary">
        <ViewOrderByDropdown
          sortBy={choices.sortBy}
          sortKey={choices.sortKey}
          onChange={({ key, order }) =>
            setChoices((current) => ({ ...current, sortKey: key ?? current.sortKey, sortBy: order ?? current.sortBy }))
          }
          isMobile
        />
      </Row>
      <div className="flex flex-grow items-center justify-center border-l border-subtle text-13 text-secondary">
        <FiltersDropdown
          icon={<FilterOutline className="h-3 w-3" />}
          title={t("common.filters")}
          placement="bottom-end"
          isFiltersApplied={false}
          menuButton={
            <Row className="flex items-center text-13 text-secondary">
              {t("common.filters")}
              <ChevronDownOutline className="ml-2 h-4 w-4 text-secondary" />
            </Row>
          }
        >
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
