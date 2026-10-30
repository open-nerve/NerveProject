/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import type { TLabelOperations } from "../root";
import { IssueLabelSelect } from "./label-select";

type TIssueLabelSelectRoot = {
  disabled?: boolean;
  showCreationToast?: boolean;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  values: string[];
  labelOperations: TLabelOperations;
};

export function IssueLabelSelectRoot(props: TIssueLabelSelectRoot) {
  const { showCreationToast, disabled, workspaceSlug, projectId, issueId, values, labelOperations } = props;

  const handleLabel = async (_labelIds: string[]) => {
    await labelOperations.updateIssue(workspaceSlug, projectId, issueId, { label_ids: _labelIds });
  };

  return (
    <IssueLabelSelect
      key={JSON.stringify([workspaceSlug, projectId, issueId])}
      disabled={disabled}
      showCreationToast={showCreationToast}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      issueId={issueId}
      values={values}
      onSelect={handleLabel}
      onAddLabel={labelOperations.create}
    />
  );
}
