/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueComment, IssueCommentCreate } from "@nerve/api-client";

import { useEffect, useRef } from "react";
import { observer } from "mobx-react";
import { useForm } from "react-hook-form";
import type { EditorRefApi } from "@nerve/editor";
import { CloseOutline, TickOutline } from "@makeplane/propel/icons";
// nerve imports
import type { TCommentsOperations } from "@nerve/types";
import { cn, isCommentEmpty } from "@nerve/utils";
// components
import { LiteTextEditor } from "@/components/editor/lite-text";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
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
  const capture = useDetailScope(workspaceSlug, projectId, `${comment.issue_id}:${comment.id}:${isEditing}`);
  const refusal = useRefusalToast();
  const { t } = useTranslation();
  // form info
  const {
    formState: { isSubmitting },
    handleSubmit,
    setFocus,
    watch,
    setValue,
  } = useForm<IssueCommentCreate>({
    defaultValues: { comment_html: comment?.comment_html },
  });
  const commentHTML = watch("comment_html");

  const isEmpty = isCommentEmpty(commentHTML);
  const isEditorReadyToDiscard = editorRef.current?.isEditorReadyToDiscard();
  const isSubmitButtonDisabled = isSubmitting || !isEditorReadyToDiscard;
  const isDisabled = isSubmitting || isEmpty || isSubmitButtonDisabled;

  const onEnter = async (formData: IssueCommentCreate) => {
    const current = capture();
    if (!current() || sending.current) return;
    sending.current = true;
    await followInSession(() => activityOperations.updateComment(comment.id, formData), {
      done: (answer) => {
        if (!current()) return;
        editorRef.current?.setEditorValue(answer.comment_html);
        readOnlyEditorRef?.setEditorValue(answer.comment_html);
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("common.success"), message: t("issue.comments.update.success") });
        setIsEditing(false);
      },
      failed: (error) => {
        if (current() && !(error instanceof SessionChangedError)) refusal(error);
      },
    });
    if (current()) sending.current = false;
  };

  useEffect(() => {
    if (isEditing) {
      setFocus("comment_html");
    }
  }, [isEditing, setFocus]);

  return (
    <form
      className="flex flex-col gap-2"
      onSubmit={handleSubmit(onEnter)}
      onKeyDown={(e) => {
        if (e.key === "Enter" && !e.shiftKey && !e.ctrlKey && !e.metaKey && !isEmpty) handleSubmit(onEnter)(e);
      }}
    >
      <div>
        <LiteTextEditor
          editable
          workspaceId={workspaceId}
          workspaceSlug={workspaceSlug}
          ref={editorRef}
          id={comment.id}
          initialValue={commentHTML ?? ""}
          value={null}
          onChange={(_json, comment_html) => setValue("comment_html", comment_html)}
          onEnterKeyPress={(e) => {
            if (!isEmpty && !isSubmitting) {
              handleSubmit(onEnter)(e);
            }
          }}
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
      </div>
      <div className="flex gap-2 self-end">
        {!isEmpty && (
          <button
            type="button"
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
