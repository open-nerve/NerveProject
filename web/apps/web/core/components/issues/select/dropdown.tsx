/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { DirectoryReadError, useDirectoryRead } from "@/components/issues/issue-detail-widgets/directory-read";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { EUserPermissionsLevel } from "@nerve/constants";
import { useLabelChanges, type IssueLabelCreation } from "@/components/labels/use-label-changes";
import { useDetailScope } from "../issue-detail-widgets/use-detail-scope";
import { useIssueLabels } from "../use-issue-labels";
import { EUserPermissions } from "@nerve/types";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import type { TWorkItemLabelSelectBaseProps } from "./base";
import { WorkItemLabelSelectBase } from "./base";

type TWorkItemLabelSelectProps = Omit<TWorkItemLabelSelectBaseProps, "labelIds" | "getLabelById" | "onDropdownOpen"> & {
  projectId: string | undefined;
};

export const IssueLabelSelect = observer(function IssueLabelSelect(props: TWorkItemLabelSelectProps) {
  const { projectId } = props;
  // router
  const { workspaceSlug } = useParams();
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { getLabelById, fetchProjectLabels } = useLabel();
  // derived values
  const labels = useIssueLabels(workspaceSlug, projectId);
  const capture = useDetailScope(workspaceSlug ?? "", projectId, "label-create", true);
  const { create } = useLabelChanges(capture);

  const canCreateLabel =
    projectId && allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, projectId);

  const read = useDirectoryRead(workspaceSlug, projectId, projectId ? () => fetchProjectLabels(projectId) : undefined);

  const handleCreateLabel: IssueLabelCreation = async (data, underField, done) => {
    if (!projectId) return;
    await create(projectId, data, underField, done);
  };

  return (
    <WorkItemLabelSelectBase
      key={JSON.stringify([workspaceSlug, projectId])}
      {...props}
      getLabelById={getLabelById}
      labelIds={labels?.map((label) => label.id) ?? []}
      onDropdownOpen={read.load}
      isInitializing={read.loading}
      readStatus={read.error ? <DirectoryReadError error={read.error} retry={read.load} /> : undefined}
      onCreateLabel={handleCreateLabel}
      createLabelEnabled={!!canCreateLabel}
    />
  );
});
