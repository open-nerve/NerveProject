/** Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only. See the LICENSE file for details. */
import { observer } from "mobx-react";
import { AddOutline } from "@makeplane/propel/icons";
import { EUserPermissionsLevel } from "@nerve/constants";
import { EUserProjectRoles } from "@nerve/types";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { useTranslation } from "@nerve/i18n";
import type { IssueLabelCreation } from "@/components/labels/use-label-changes";
import { useLabel } from "@/hooks/store/use-label";
import { useUserPermissions } from "@/hooks/store/user";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { DirectoryReadError, useDirectoryRead } from "@/components/issues/issue-detail-widgets/directory-read";
import { WorkItemLabelSelectBase } from "@/components/issues/select/base";

export interface IIssueLabelSelect {
  disabled?: boolean;
  showCreationToast?: boolean;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  values: string[];
  onSelect: (_labelIds: string[]) => Promise<void>;
  onAddLabel: IssueLabelCreation;
}

export const IssueLabelSelect = observer(function IssueLabelSelect(props: IIssueLabelSelect) {
  const {
    showCreationToast = false,
    disabled,
    workspaceSlug,
    projectId,
    issueId,
    values,
    onSelect,
    onAddLabel,
  } = props;
  const { t } = useTranslation();
  const { fetchProjectLabels, getProjectLabels, getLabelById } = useLabel();
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const { allowPermissions } = useUserPermissions();
  const read = useDirectoryRead(workspaceSlug, projectId, () => fetchProjectLabels(projectId));
  const canCreate = allowPermissions(
    [EUserProjectRoles.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );
  if (!issueId || !values) return null;
  return (
    <WorkItemLabelSelectBase
      grouped={false}
      issueName={getIssueById(issueId)?.name}
      disabled={disabled}
      value={values}
      onChange={onSelect}
      onCreateLabel={onAddLabel}
      onCreated={
        showCreationToast
          ? () => setToast({ title: t("toast.success"), type: TOAST_TYPE.SUCCESS, message: t("label.create.success") })
          : undefined
      }
      createLabelEnabled={canCreate}
      getLabelById={getLabelById}
      labelIds={getProjectLabels(projectId)?.map((label) => label.id) ?? []}
      onDropdownOpen={read.load}
      isInitializing={read.loading}
      readStatus={read.error ? <DirectoryReadError error={read.error} retry={read.load} /> : undefined}
      buttonContainerClassName="gap-1 rounded-sm px-2 py-1 text-secondary hover:bg-layer-1"
      label={
        <>
          <AddOutline className="size-3" />
          <span className="text-body-xs-medium text-placeholder">{t("label.select")}</span>
        </>
      }
    />
  );
});
