/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams } from "react-router";
// components
import type { TPowerKPageType } from "@/components/power-k/core/types";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useMember } from "@/hooks/store/use-member";
// local imports
import { PowerKMembersMenu } from "../../../../menus/members";
import { PowerKWorkItemCyclesMenu } from "./cycles-menu";
import { PowerKWorkItemLabelsMenu } from "./labels-menu";
import { PowerKWorkItemModulesMenu } from "./modules-menu";
import { PowerKWorkItemPrioritiesMenu } from "./priorities-menu";
import { PowerKProjectStatesMenu } from "./states-menu";

type Props = {
  activePage: TPowerKPageType | null;
  handleSelection: (data: unknown) => void;
};

export const PowerKWorkItemContextBasedPages = observer(function PowerKWorkItemContextBasedPages(props: Props) {
  const { activePage, handleSelection } = props;
  // navigation
  const { workItem: entityIdentifier } = useParams();
  // store hooks
  const {
    issue: { getIssueById, getIssueIdByIdentifier },
  } = useIssueDetail();
  const {
    project: { getProjectMemberIds },
  } = useMember();
  // derived values
  const entityId = entityIdentifier ? getIssueIdByIdentifier(entityIdentifier) : null;
  const entityDetails = entityId ? getIssueById(entityId) : null;
  const projectMemberIds = entityDetails?.project_id ? getProjectMemberIds(entityDetails.project_id, false) : [];

  if (!entityDetails) return null;

  return (
    <>
      {/* states menu */}
      {activePage === "update-work-item-state" && (
        <PowerKProjectStatesMenu handleSelect={handleSelection} workItemDetails={entityDetails} />
      )}
      {/* priority menu */}
      {activePage === "update-work-item-priority" && (
        <PowerKWorkItemPrioritiesMenu handleSelect={handleSelection} workItemDetails={entityDetails} />
      )}
      {/* members menu */}
      {activePage === "update-work-item-assignee" && (
        <PowerKMembersMenu
          handleSelect={handleSelection}
          userIds={projectMemberIds ?? undefined}
          value={entityDetails.assignee_ids}
        />
      )}
      {/* cycles menu */}
      {activePage === "update-work-item-cycle" && (
        <PowerKWorkItemCyclesMenu handleSelect={handleSelection} workItemDetails={entityDetails} />
      )}
      {/* modules menu */}
      {activePage === "update-work-item-module" && (
        <PowerKWorkItemModulesMenu handleSelect={handleSelection} workItemDetails={entityDetails} />
      )}
      {/* labels menu */}
      {activePage === "update-work-item-labels" && (
        <PowerKWorkItemLabelsMenu handleSelect={handleSelection} workItemDetails={entityDetails} />
      )}
    </>
  );
});
