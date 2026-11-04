/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueCommentCreate } from "@nerve/api-client";

import { useId, useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useForm, Controller } from "react-hook-form";
// nerve imports
import type { EditorRefApi } from "@nerve/editor";
import type { TCommentsOperations } from "@nerve/types";
import { cn, isCommentEmpty, isComposing } from "@nerve/utils";
// components
import { LiteTextEditor } from "@/components/editor/lite-text";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// services
import { useCommentView, useCommentPreparations } from "./use-comment-content";
import { useCommentFieldError } from "./use-comment-field-error";
import { useCommentInput } from "./use-comment-input";
import { PreparedCommentFiles } from "./prepared-files";
import { CommentInputFeedback } from "./input-feedback";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useFormRefusal } from "@/lib/form-refusal";
import { useRefusedFocus } from "@/lib/refused-focus";
import { useDetailScope } from "../issues/issue-detail-widgets/use-detail-scope";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";

type TCommentCreate = {
  entityId: string;
  workspaceSlug: string;
  activityOperations: TCommentsOperations;
  showToolbarInitially?: boolean;
  projectId: string;
  onSubmitCallback?: (elementId: string) => void;
};

export const CommentCreate = observer(function CommentCreate(props: TCommentCreate) {
  return <CommentCreateForm key={`${props.workspaceSlug}:${props.projectId}:${props.entityId}`} {...props} />;
});

const CommentCreateForm = observer(function CommentCreateForm(props: TCommentCreate) {
  const {
    workspaceSlug,
    entityId,
    activityOperations,
    showToolbarInitially = false,
    projectId,
    onSubmitCallback,
  } = props;
  // states
  const input = useCommentInput("new-comment", "<p></p>");
  const view = useCommentView(workspaceSlug, projectId, entityId);
  const preparations = useCommentPreparations(view, input.value.attachmentIDs, input.select);
  const detail = useIssueDetail();
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const sending = useRef(false);
  const [isSubmitting, setSubmitting] = useState(false);
  const capture = useDetailScope(workspaceSlug, projectId, entityId);
  const formRefusal = useFormRefusal(["comment_html"], {});
  const errorId = useId();
  const { t } = useTranslation();
  // store hooks
  const workspaceStore = useWorkspace();
  // derived values
  const workspaceId = workspaceStore.getWorkspaceBySlug(workspaceSlug)?.id;
  // form info
  const {
    handleSubmit,
    control,
    formState: { errors, submitCount },
    setError,
    clearErrors,
    reset,
  } = useForm<IssueCommentCreate>({
    defaultValues: {
      comment_html: input.value.html,
    },
    shouldFocusError: false,
  });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  useCommentFieldError(input.value.error, setError);
  const { focused } = input;
  const focusAfterRead = input.value.focusAfterRead;
  useLayoutEffect(() => {
    if (view?.current && focusAfterRead && editorRef.current?.isEditorReadyToDiscard()) {
      editorRef.current.focus("end");
      focused();
    }
  }, [view, focusAfterRead, focused]);

  const onSubmit = async (formData: IssueCommentCreate) => {
    const current = capture();
    if (
      !current() ||
      sending.current ||
      !view?.canWrite("attachment") ||
      !preparations.ready ||
      input.value.unknown ||
      isCommentEmpty(formData.comment_html) ||
      !editorRef.current?.isEditorReadyToDiscard()
    )
      return;
    const attachment_ids = preparations.files!.payload();
    sending.current = true;
    input.begin();
    setSubmitting(true);
    clearErrors("comment_html");
    await followInSession(
      () =>
        activityOperations.createComment({ ...formData, attachment_ids }, (comment) => {
          if (!current() || !view.current) return;
          input.confirmed("<p></p>", true);
          preparations.files!.clear();
          preparations.upload?.cancel();
          reset({ comment_html: "<p></p>" });
          editorRef.current?.clearEditor();
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("common.success"),
            message: t("issue.comments.create.success"),
          });
          onSubmitCallback?.(comment.id);
        }),
      {
        failed: (error) => {
          if (current() && !(error instanceof SessionChangedError)) {
            input.failed(error);
            formRefusal(error, (field, message) => setError(field, { type: "server", message }));
          }
        },
      }
    );
    sending.current = false;
    if (current()) setSubmitting(false);
  };

  if (!workspaceId) return null;

  return (
    <form
      ref={formRef}
      className={cn("sticky bottom-0 z-[4] bg-surface-1 sm:static")}
      onSubmit={handleSubmit(onSubmit)}
      onKeyDownCapture={(e) => {
        if (e.key === "Enter" && isComposing(e.nativeEvent)) {
          e.preventDefault();
          e.stopPropagation();
        }
      }}
    >
      <div inert={isSubmitting || !view?.canWrite("attachment")}>
        <Controller
          name="comment_html"
          control={control}
          render={({ field: { value, onChange } }) => (
            <LiteTextEditor
              editable={!!view?.canWrite("description")}
              editorProps={{
                attributes: {
                  role: "textbox",
                  "aria-multiline": "true",
                  "aria-label": t("common.comment"),
                  "aria-invalid": String(!!errors.comment_html),
                  "aria-describedby": errors.comment_html ? errorId : "",
                },
              }}
              workspaceId={workspaceId}
              id={"add_comment_" + entityId + "_" + errorId}
              value="<p></p>"
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              onEnterKeyPress={(e) => void handleSubmit(onSubmit)(e)}
              ref={editorRef}
              initialValue={value ?? "<p></p>"}
              containerClassName="min-h-min"
              onChange={(_json, comment_html) => {
                input.change(comment_html);
                clearErrors("comment_html");
                onChange(comment_html);
              }}
              isSubmitting={isSubmitting}
              submitDisabled={!preparations.ready || input.value.unknown}
              uploadFile={async (blockId, file) => {
                const { asset_id } = await activityOperations.uploadCommentAsset(blockId, file);
                return asset_id;
              }}
              duplicateFile={async (assetId: string) => {
                const { asset_id } = await activityOperations.duplicateCommentAsset(assetId);
                return asset_id;
              }}
              showToolbarInitially={showToolbarInitially}
              parentClassName="p-2"
              displayConfig={{
                fontSize: "small-font",
              }}
            />
          )}
        />
        {errors.comment_html && (
          <p id={errorId} className="text-13 text-danger-primary">
            {errors.comment_html.message}
          </p>
        )}
      </div>
      <div inert={isSubmitting}>
        {preparations.files && <PreparedCommentFiles files={preparations.files} upload={preparations.upload} />}
      </div>
      {!isSubmitting && (
        <CommentInputFeedback
          value={input.value}
          current={() => capture()() && !!view?.current}
          verify={async () => {
            await detail.comment.fetchComments(workspaceSlug, projectId, entityId);
            if (view?.current) await preparations.files?.load();
          }}
          verified={input.verify}
          allowResend={input.allowResend}
        />
      )}
    </form>
  );
});
