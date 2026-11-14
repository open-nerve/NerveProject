// oxlint-disable promise/always-return
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { pick } from "lodash-es";
import { ISSUE_FORM_FIELDS } from "./types";
import type { IssueFormData, IssueFormValues, IssueDescriptionLoadState, IssueFormSubmitOptions } from "./types";
import { useState } from "react";
import type { MutableRefObject, RefObject } from "react";
import type { UseFormReturn } from "react-hook-form";
// editor
import { DEFAULT_WORK_ITEM_FORM_VALUES } from "@nerve/constants";
import type { EditorRefApi } from "@nerve/editor";
// i18n
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { getChangedIssuefields } from "@nerve/utils";
import { useFormRefusal } from "@/lib/form-refusal";

type FormSubmissionInput = {
  methods: UseFormReturn<IssueFormValues>;
  data: IssueFormData | undefined;
  hasDraftSeed: boolean;
  isDraft: boolean;
  descriptionLoadState: IssueDescriptionLoadState;
  editorRef: RefObject<EditorRefApi | null>;
  /** Whether the description's value is a pure ID migration of the one loaded. */
  descriptionMigration: MutableRefObject<boolean>;
  isCreateMoreToggleEnabled: boolean;
  onSubmit: (
    values: Partial<IssueFormValues>,
    is_draft_issue?: boolean,
    options?: IssueFormSubmitOptions
  ) => Promise<void>;
  onPublish: ((values: IssueFormData) => Promise<void>) | undefined;
  isPending: boolean;
};

/** What the form sends when it saves or publishes, and nerve's refusal of it under the form's fields. */
export function useFormSubmission({
  methods,
  data,
  hasDraftSeed,
  isDraft,
  descriptionLoadState,
  editorRef,
  descriptionMigration,
  isCreateMoreToggleEnabled,
  onSubmit,
  onPublish,
  isPending,
}: FormSubmissionInput) {
  const { t } = useTranslation();
  // states
  const [isMoving, setIsMoving] = useState<boolean>(false);
  const {
    formState: { isSubmitting, dirtyFields },
    reset,
    getValues,
    setError,
  } = methods;
  const refuse = useFormRefusal(["name", "project_id", "start_date", "target_date"] as const, {});

  const submissionData = (formData: Partial<IssueFormValues>) => {
    if (!data?.id && !hasDraftSeed) return formData;
    // Seeded drafts retain wire fields that the display controls cannot resolve.
    // Only actual edits replace those fields; the command owner merges the seed.
    return {
      ...getChangedIssuefields(formData, dirtyFields as { [key: string]: boolean | undefined }),
      project_id: getValues("project_id"),
      ...(data?.id ? { id: data.id, description_html: formData.description_html ?? "<p></p>" } : {}),
    };
  };

  const handleFormSubmit = async (formData: Partial<IssueFormValues>, is_draft_issue = isDraft) => {
    if (descriptionLoadState !== "ready") return;
    // Check if the editor is ready to discard
    if (!editorRef.current?.isEditorReadyToDiscard()) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("error"),
        message: t("editor_is_not_ready_to_discard_changes"),
      });
      return;
    }

    await onSubmit(submissionData(formData), is_draft_issue, {
      isDescriptionMigration: descriptionMigration.current,
      onRefused: (error) => refuse(error, (field, message) => setError(field, { type: "server", message })),
    })
      .then(() => {
        descriptionMigration.current = false;
        reset({
          ...DEFAULT_WORK_ITEM_FORM_VALUES,
          ...(isCreateMoreToggleEnabled ? pick(data, ISSUE_FORM_FIELDS) : {}),
          project_id: getValues<"project_id">("project_id"),
          description_html: data?.description_html ?? "<p></p>",
        });
        editorRef?.current?.clearEditor();
      })
      .catch(() => undefined);
  };

  const handleMoveToProjects = async () => {
    if (!data?.id || !onPublish || isMoving || isSubmitting || isPending || descriptionLoadState !== "ready") return;
    if (!editorRef.current?.isEditorReadyToDiscard()) {
      setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("editor_is_not_ready_to_discard_changes") });
      return;
    }
    setIsMoving(true);
    try {
      await onPublish({
        ...getChangedIssuefields(getValues(), dirtyFields as { [key: string]: boolean | undefined }),
        project_id: getValues("project_id"),
        description_html: getValues("description_html"),
      });
    } catch {
      /* The command owner displays the current refusal; retain the form. */
    } finally {
      setIsMoving(false);
    }
  };

  return { isMoving, submissionData, handleFormSubmit, handleMoveToProjects };
}
