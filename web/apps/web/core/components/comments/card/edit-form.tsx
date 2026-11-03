/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueComment, IssueCommentUpdate } from "@nerve/api-client";

import { useId, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useForm } from "react-hook-form";
import type { EditorRefApi } from "@nerve/editor";
import { CloseOutline, TickOutline } from "@makeplane/propel/icons";
// nerve imports
import type { TCommentsOperations } from "@nerve/types";
import { cn, isCommentEmpty, isComposing } from "@nerve/utils";
// components
import { LiteTextEditor } from "@/components/editor/lite-text";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useFormRefusal } from "@/lib/form-refusal";
import { useRefusedFocus } from "@/lib/refused-focus";
import { useDetailScope } from "../../issues/issue-detail-widgets/use-detail-scope";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";

type Props = {
  activityOperations: TCommentsOperations;
  comment: IssueComment;
  isEditing: boolean;
  projectId?: string;
  readOnlyEditorRef: EditorRefApi | null;
  setIsEditing: (isEditing: boolean) => void;
  workspaceId: string;
  workspaceSlug: string;
};

export const CommentCardEditForm = observer(function CommentCardEditForm(props: Props) {
  const {
    activityOperations,
    comment,
    isEditing,
    projectId,
    readOnlyEditorRef,
    setIsEditing,
    workspaceId,
    workspaceSlug,
  } = props;
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const sending = useRef(false);
  const [isSubmitting, setSubmitting] = useState(false);
  const capture = useDetailScope(workspaceSlug, projectId, `${comment.issue_id}:${comment.id}:${isEditing}`);
  const refusal = useFormRefusal(["comment_html"], {});
  const errorId = useId();
  const { t } = useTranslation();
  // form info
  const {
    formState: { errors, submitCount },
    handleSubmit,
    setError,
    clearErrors,
    watch,
    setValue,
  } = useForm<IssueCommentUpdate>({
    defaultValues: { comment_html: comment?.comment_html },
    shouldFocusError: false,
  });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  const commentHTML = watch("comment_html");

  const isEmpty = isCommentEmpty(commentHTML);
  const isEditorReadyToDiscard = editorRef.current?.isEditorReadyToDiscard();
  const isSubmitButtonDisabled = isSubmitting || !isEditorReadyToDiscard;
  const isDisabled = isSubmitting || isEmpty || isSubmitButtonDisabled;

  const onEnter = async (formData: IssueCommentUpdate) => {
    const current = capture();
    if (
      !current() ||
      sending.current ||
      isCommentEmpty(formData.comment_html) ||
      !editorRef.current?.isEditorReadyToDiscard()
    )
      return;
    sending.current = true;
    setSubmitting(true);
    clearErrors("comment_html");
    await followInSession(() => activityOperations.updateComment(comment.id, formData), {
      done: (answer) => {
        if (!current()) return;
        editorRef.current?.setEditorValue(answer.comment_html);
        readOnlyEditorRef?.setEditorValue(answer.comment_html);
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("common.success"), message: t("issue.comments.update.success") });
        setIsEditing(false);
      },
      failed: (error) => {
        if (current() && !(error instanceof SessionChangedError))
          refusal(error, (field, message) => setError(field, { type: "server", message }));
      },
    });
    sending.current = false;
    setSubmitting(false);
  };

  return (
    <form
      ref={formRef}
      className="flex flex-col gap-2"
      onSubmit={handleSubmit(onEnter)}
      onKeyDownCapture={(e) => {
        if (e.key === "Enter" && isComposing(e.nativeEvent)) {
          e.preventDefault();
          e.stopPropagation();
        }
      }}
    >
      <div inert={isSubmitting}>
        <LiteTextEditor
          editable
          editorProps={{
            attributes: {
              role: "textbox",
              "aria-multiline": "true",
              "aria-label": t("common.comment"),
              "aria-invalid": String(!!errors.comment_html),
              "aria-describedby": errors.comment_html ? errorId : "",
            },
          }}
          autofocus
          workspaceId={workspaceId}
          workspaceSlug={workspaceSlug}
          ref={editorRef}
          id={comment.id}
          initialValue={commentHTML ?? ""}
          value={null}
          onChange={(_json, comment_html) => setValue("comment_html", comment_html)}
          onEnterKeyPress={(e) => void handleSubmit(onEnter)(e)}
          isSubmitting={isSubmitting}
          showSubmitButton={false}
          uploadFile={async (blockId, file) => {
            const { asset_id } = await activityOperations.uploadCommentAsset(blockId, file, comment.id);
            return asset_id;
          }}
          duplicateFile={async (assetId: string) => {
            const { asset_id } = await activityOperations.duplicateCommentAsset(assetId, comment.id);
            return asset_id;
          }}
          projectId={projectId}
          parentClassName="p-2 bg-surface-1"
          displayConfig={{
            fontSize: "small-font",
          }}
        />
        {errors.comment_html && (
          <p id={errorId} className="text-13 text-danger-primary">
            {errors.comment_html.message}
          </p>
        )}
      </div>
      <div className="flex gap-2 self-end">
        {!isEmpty && (
          <button
            type="button"
            aria-label={t("save")}
            onClick={handleSubmit(onEnter)}
            disabled={isDisabled}
            className={cn(
              "group grid size-7 place-items-center rounded-lg border border-success-subtle bg-success-subtle shadow-raised-100 duration-300",
              isDisabled ? "" : "hover:bg-success-subtle-1"
            )}
          >
            <TickOutline className="size-4 text-success-primary" />
          </button>
        )}
        <button
          type="button"
          aria-label={t("common.cancel")}
          disabled={isSubmitting}
          className={cn(
            "group grid size-7 place-items-center rounded-lg border border-danger-subtle bg-danger-subtle shadow-raised-100 duration-300",
            isSubmitting ? "" : "hover:bg-danger-subtle-hover"
          )}
          onClick={() => {
            setIsEditing(false);
            editorRef.current?.setEditorValue(comment.comment_html ?? "<p></p>");
          }}
        >
          <CloseOutline className="size-4 text-danger-primary" />
        </button>
      </div>
    </form>
  );
});
