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
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { ETabIndices } from "@nerve/constants";
// types
import { getTabIndex } from "@nerve/utils";
// components
import { ProjectDropdown } from "@/components/dropdowns/project/dropdown";
// hooks
import { useIssueModal } from "@/hooks/context/use-issue-modal";
import { usePlatformOS } from "@/hooks/use-platform-os";

type TIssueProjectSelectProps = {
  control: Control<IssueFormValues>;
  disabled?: boolean;
  isDraft?: boolean;
  handleFormChange: () => void;
};

export const IssueProjectSelect = observer(function IssueProjectSelect(props: TIssueProjectSelectProps) {
  const { control, disabled = false, isDraft = false, handleFormChange } = props;
  // store hooks
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  // context hooks
  const { allowedProjectIds } = useIssueModal();

  const { getIndex } = getTabIndex(ETabIndices.ISSUE_FORM, isMobile);

  return (
    <Controller
      control={control}
      name="project_id"
      rules={{
        required: !isDraft,
      }}
      render={({ field: { value, onChange } }) => (
        <div className="flex h-7 items-center gap-1">
          <ProjectDropdown
            value={value}
            onChange={(projectId) => {
              onChange(projectId);
              handleFormChange();
            }}
            multiple={false}
            buttonVariant="border-with-text"
            renderCondition={(projectId) => allowedProjectIds.includes(projectId)}
            tabIndex={getIndex("project_id")}
            disabled={disabled}
          />
          {isDraft && value && (
            <button
              type="button"
              disabled={disabled}
              onClick={() => {
                onChange(null);
                handleFormChange();
              }}
            >
              {t("workspace_draft_issues.clear_project")}
            </button>
          )}
        </div>
      )}
    />
  );
});
