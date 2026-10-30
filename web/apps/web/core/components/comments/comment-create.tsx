/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueCommentCreate } from "@nerve/api-client";

import { useRef, useState } from "react";
import { observer } from "mobx-react";
import { useForm, Controller } from "react-hook-form";
// nerve imports
import type { EditorRefApi } from "@nerve/editor";
import type { TCommentsOperations } from "@nerve/types";
import { cn, isCommentEmpty } from "@nerve/utils";
// components
import { LiteTextEditor } from "@/components/editor/lite-text";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// services
import { FileService } from "@/services/file.service";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
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

// services
const fileService = new FileService();

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
  const [uploadedAssetIds, setUploadedAssetIds] = useState<string[]>([]);
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const sending = useRef(false);
  const capture = useDetailScope(workspaceSlug, projectId, entityId);
  const refusal = useRefusalToast();
  const { t } = useTranslation();
  // store hooks
  const workspaceStore = useWorkspace();
  // derived values
  const workspaceId = workspaceStore.getWorkspaceBySlug(workspaceSlug)?.id;
  // form info
  const {
    handleSubmit,
    control,
    watch,
    formState: { isSubmitting },
    reset,
  } = useForm<IssueCommentCreate>({
    defaultValues: {
      comment_html: "<p></p>",
    },
  });

  const onSubmit = async (formData: IssueCommentCreate) => {
    const current = capture();
    if (!current() || sending.current) return;
    sending.current = true;
    await followInSession(() => activityOperations.createComment(formData), {
      done: async (comment) => {
        if (!current()) return;
        reset({ comment_html: "<p></p>" });
        editorRef.current?.clearEditor();
        setUploadedAssetIds([]);
        // Asset bookkeeping is a separate M5 follow-up; it cannot turn a confirmed comment into a failed create.
        if (uploadedAssetIds.length)
          await followInSession(
            () =>
              fileService.updateBulkProjectAssetsUploadStatus(workspaceSlug, projectId, entityId, {
                asset_ids: uploadedAssetIds,
              }),
            {
              failed: (error) => {
                if (current() && !(error instanceof SessionChangedError)) refusal(error);
              },
            }
          );
        if (!current()) return;
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("common.success"), message: t("issue.comments.create.success") });
        onSubmitCallback?.(comment.id);
      },
      failed: (error) => {
        if (current() && !(error instanceof SessionChangedError)) refusal(error);
      },
    });
    if (current()) sending.current = false;
  };

  const commentHTML = watch("comment_html");
  const isEmpty = isCommentEmpty(commentHTML);

  if (!workspaceId) return null;

  return (
    <form
      className={cn("sticky bottom-0 z-[4] bg-surface-1 sm:static")}
      onSubmit={handleSubmit(onSubmit)}
      onKeyDown={(e) => {
        if (
          e.key === "Enter" &&
          !e.shiftKey &&
          !e.ctrlKey &&
          !e.metaKey &&
          !isEmpty &&
          !isSubmitting &&
          editorRef.current?.isEditorReadyToDiscard()
        )
          handleSubmit(onSubmit)(e);
      }}
    >
      <Controller
        name="comment_html"
        control={control}
        render={({ field: { value, onChange } }) => (
          <LiteTextEditor
            editable
            workspaceId={workspaceId}
            id={"add_comment_" + entityId}
            value="<p></p>"
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            onEnterKeyPress={(e) => {
              if (!isEmpty && !isSubmitting) {
                handleSubmit(onSubmit)(e);
              }
            }}
            ref={editorRef}
            initialValue={value ?? "<p></p>"}
            containerClassName="min-h-min"
            onChange={(_json, comment_html) => onChange(comment_html)}
            isSubmitting={isSubmitting}
            uploadFile={async (blockId, file) => {
              const { asset_id } = await activityOperations.uploadCommentAsset(blockId, file);
              setUploadedAssetIds((prev) => [...prev, asset_id]);
              return asset_id;
            }}
            duplicateFile={async (assetId: string) => {
              const { asset_id } = await activityOperations.duplicateCommentAsset(assetId);
              setUploadedAssetIds((prev) => [...prev, asset_id]);
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
    </form>
  );
});
