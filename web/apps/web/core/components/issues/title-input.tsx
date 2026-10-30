/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useId, useState } from "react";
import { AutosaveError, useConfirmedAutosave } from "@/components/editor/use-confirmed-autosave";
import { observer } from "mobx-react";
import { Field } from "@makeplane/propel/components/field";
import { TextArea } from "@makeplane/propel/components/text-area";
import { useTranslation } from "@nerve/i18n";
import type { TNameDescriptionLoader } from "@nerve/types";
// types
import { cn } from "@nerve/utils";
import type { TIssueOperations } from "./issue-detail";

export type IssueTitleInputProps = {
  disabled?: boolean;
  value: string | undefined | null;
  workspaceSlug: string;
  setIsSubmitting: (value: TNameDescriptionLoader) => void;
  issueOperations: Pick<TIssueOperations<{ name: string }>, "update" | "prepareUpdate">;
  projectId: string;
  issueId: string;
  containerClassName?: string;
};

export function IssueTitleInput(props: IssueTitleInputProps) {
  return (
    <IssueTitleInputContent key={JSON.stringify([props.workspaceSlug, props.projectId, props.issueId])} {...props} />
  );
}
const sameTitle = (left: string, right: string) => left === right;
const IssueTitleInputContent = observer(function IssueTitleInputContent(props: IssueTitleInputProps) {
  const { disabled, value, workspaceSlug, setIsSubmitting, issueId, issueOperations, projectId, containerClassName } =
    props;
  const { t } = useTranslation();
  const [isLengthVisible, setIsLengthVisible] = useState(false);
  const inputId = useId();
  const {
    value: title,
    change,
    failure,
    retry,
  } = useConfirmedAutosave({
    initial: value ?? "",
    same: sameTitle,
    save: (name) => issueOperations.update(workspaceSlug, projectId, issueId, { name }),
    prepareSave: issueOperations.prepareUpdate
      ? () => {
          const update = issueOperations.prepareUpdate!();
          return (name) => update(workspaceSlug, projectId, issueId, { name });
        }
      : undefined,
    status: setIsSubmitting,
    disabled,
  });

  if (disabled) return <div className="text-20 font-medium whitespace-pre-line">{title}</div>;

  return (
    <div className="flex flex-col gap-1.5">
      <div className={cn("relative", containerClassName)}>
        <Field name="title" invalid={title?.length === 0}>
          <div className="flex w-full">
            <TextArea
              size="2xl"
              surface="inline"
              autoResize
              rows={1}
              id={inputId}
              disabled={disabled}
              value={title}
              onChange={(event) => change(event.target.value)}
              maxLength={255}
              placeholder={t("issue.title.label")}
              onFocus={() => setIsLengthVisible(true)}
              onBlur={() => {
                setIsLengthVisible(false);
                const trimmed = title.trim();
                if (trimmed && trimmed !== title) change(trimmed);
              }}
            />
          </div>
        </Field>
        <div
          className={cn(
            "pointer-events-none absolute right-1 bottom-1 z-2 rounded-sm bg-surface-1 p-0.5 text-11 text-secondary opacity-0 transition-opacity",
            {
              "opacity-100": isLengthVisible,
            }
          )}
        >
          <span className={`${title.length === 0 || title.length > 255 ? "text-danger-primary" : ""}`}>
            {title.length}
          </span>
          /255
        </div>
      </div>
      <AutosaveError failure={failure} retry={retry} />
      {title?.length === 0 && (
        <span className="text-13 font-medium text-danger-primary">{t("form.title.required")}</span>
      )}
    </div>
  );
});
