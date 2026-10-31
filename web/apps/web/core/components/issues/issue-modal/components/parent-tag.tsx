/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueFormValues } from "../types";
import React from "react";
import { observer } from "mobx-react";
import type { Control } from "react-hook-form";
import { Controller } from "react-hook-form";
import { useTranslation } from "@nerve/i18n";
import { CloseOutline } from "@makeplane/propel/icons";
// types
import type { Issue } from "@nerve/api-client";
// helpers
// hooks
// components
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
import { useProjectState } from "@/hooks/store/use-project-state";

type TIssueParentTagProps = {
  control: Control<IssueFormValues>;
  selectedParentIssue: Issue;
  handleFormChange: () => void;
  setSelectedParentIssue: (issue: Issue | null) => void;
};

export const IssueParentTag = observer(function IssueParentTag(props: TIssueParentTagProps) {
  const { control, selectedParentIssue, handleFormChange, setSelectedParentIssue } = props;
  // store hooks
  const { t } = useTranslation();
  const { getStateById } = useProjectState();

  return (
    <Controller
      control={control}
      name="parent_id"
      render={({ field: { onChange } }) => (
        <div className="flex w-min items-center gap-2 rounded-sm bg-surface-2 p-2 text-caption-sm-regular whitespace-nowrap">
          <div className="flex items-center gap-2">
            <span
              className="block h-1.5 w-1.5 rounded-full bg-icon-placeholder"
              style={{
                backgroundColor: getStateById(selectedParentIssue.state_id)?.color,
              }}
            />
            <span className="flex-shrink-0 text-secondary">
              {selectedParentIssue?.project_id && (
                <IssueIdentifier
                  projectId={selectedParentIssue.project_id}
                  issueId={selectedParentIssue.id}
                  size="xs"
                />
              )}
            </span>
            <span className="truncate font-medium">{selectedParentIssue.name.substring(0, 50)}</span>
            <button
              type="button"
              className="grid place-items-center"
              onClick={() => {
                onChange(null);
                handleFormChange();
                setSelectedParentIssue(null);
              }}
              aria-label={t("remove_parent_issue")}
            >
              <CloseOutline className="h-3 w-3 cursor-pointer" />
            </button>
          </div>
        </div>
      )}
    />
  );
});
