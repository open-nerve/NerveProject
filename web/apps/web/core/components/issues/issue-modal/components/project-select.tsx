/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueFormValues } from "../types";
import { useId } from "react";
import { observer } from "mobx-react";
import type { Control } from "react-hook-form";
import { Controller } from "react-hook-form";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Field, FieldError } from "@makeplane/propel/components/field";
// components
import { ProjectDropdown } from "@/components/dropdowns/project/dropdown";
// hooks
import { useIssueModal } from "@/hooks/context/use-issue-modal";

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
  const errorId = useId();
  // context hooks
  const { allowedProjectIds } = useIssueModal();

  return (
    <Controller
      control={control}
      name="project_id"
      rules={{
        required: isDraft ? false : t("project_id_is_required"),
      }}
      render={({ field: { value, onChange }, fieldState: { error } }) => (
        <Field name="project_id" invalid={!!error}>
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
              invalid={!!error}
              describedBy={error ? errorId : undefined}
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
          {error?.message && (
            <FieldError id={errorId} size="lg" match>
              {error.message}
            </FieldError>
          )}
        </Field>
      )}
    />
  );
});
