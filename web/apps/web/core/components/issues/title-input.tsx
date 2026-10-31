/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useId, useState } from "react";
import { AutosaveError, useConfirmedAutosave } from "@/components/editor/use-confirmed-autosave";
import { observer } from "mobx-react";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { TextArea } from "@makeplane/propel/components/text-area";
import { useTranslation } from "@nerve/i18n";
import type { TNameDescriptionLoader } from "@nerve/types";
// types
import { cn } from "@nerve/utils";
import type { TIssueOperations } from "./issue-detail";
import { fieldErrorKeys } from "@/lib/error-messages";
import { useRefusedFocus } from "@/lib/refused-focus";
import { SessionChangedError } from "@/lib/auth/token-manager";

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
const titleError = (name: string) => {
  const length = Array.from(name.trim()).length;
  return length === 0 ? "form.title.required" : length > 255 ? "errors.field.too_long" : undefined;
};
function validateTitle(name: string): void {
  const message = titleError(name);
  if (message) throw new Error(message);
}
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
    save: (name) => {
      validateTitle(name);
      return issueOperations.update(workspaceSlug, projectId, issueId, { name });
    },
    prepareSave: issueOperations.prepareUpdate
      ? () => {
          const update = issueOperations.prepareUpdate!();
          return (name) => {
            validateTitle(name);
            return update(workspaceSlug, projectId, issueId, { name });
          };
        }
      : undefined,
    status: setIsSubmitting,
    disabled,
  });
  const currentFailure = failure?.error instanceof SessionChangedError ? undefined : failure;
  const fieldError = titleError(title) ?? fieldErrorKeys(currentFailure?.error).name;
  const fieldRef = useRefusedFocus<HTMLDivElement>(currentFailure);
  const length = Array.from(title).length;

  if (disabled) return <div className="text-20 font-medium whitespace-pre-line">{title}</div>;

  return (
    <div className="flex flex-col gap-1.5" ref={fieldRef}>
      <div className={cn("relative", containerClassName)}>
        <Field name="title" invalid={!!fieldError}>
          <div className="flex w-full">
            <TextArea
              size="2xl"
              surface="inline"
              autoResize
              rows={1}
              id={inputId}
              aria-label={t("issue.title.label")}
              disabled={disabled}
              value={title}
              onChange={(event) => change(event.target.value)}
              placeholder={t("issue.title.label")}
              onFocus={() => setIsLengthVisible(true)}
              onBlur={() => {
                setIsLengthVisible(false);
                const trimmed = title.trim();
                if (trimmed && trimmed !== title) change(trimmed);
              }}
            />
          </div>
          {fieldError && (
            <FieldError size="2xl" match>
              {t(fieldError)}
            </FieldError>
          )}
        </Field>
        <div
          className={cn(
            "pointer-events-none absolute right-1 bottom-1 z-2 rounded-sm bg-surface-1 p-0.5 text-11 text-secondary opacity-0 transition-opacity",
            {
              "opacity-100": isLengthVisible,
            }
          )}
        >
          <span className={`${fieldError ? "text-danger-primary" : ""}`}>{length}</span>
          /255
        </div>
      </div>
      {fieldError ? (
        currentFailure && (
          <button type="button" onClick={retry}>
            {t("common.retry")}
          </button>
        )
      ) : (
        <AutosaveError failure={failure} retry={retry} />
      )}
    </div>
  );
});
