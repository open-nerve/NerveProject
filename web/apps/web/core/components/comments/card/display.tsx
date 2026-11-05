/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueComment } from "@nerve/api-client";

import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useLocation } from "react-router";
// nerve imports
import { ProfileAvatar } from "@/components/assets/profile-image";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { EditorRefApi } from "@nerve/editor";
import { useHashScroll } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
import type { TCommentsOperations } from "@nerve/types";
import { calculateTimeAgo, cn, renderFormattedDate, renderFormattedTime } from "@nerve/utils";
// components
import { LiteTextEditor } from "@/components/editor/lite-text";
import { useCommentView } from "../use-comment-content";
import { useEditorAssets } from "@/hooks/editor/use-editor-assets";
// local imports
import { CommentAttachments } from "../attachments";
import { CommentReactions } from "../comment-reaction";
import { CommentCardEditForm } from "./edit-form";
import { EmojiReactionPicker } from "@nerve/propel/emoji-reaction";
import { ReactionOutline } from "@makeplane/propel/icons";
import { useMember } from "@/hooks/store/use-member";

export type TCommentCardDisplayProps = {
  activityOperations: TCommentsOperations;
  comment: IssueComment;
  disabled: boolean;
  projectId?: string;
  readOnlyEditorRef: React.RefObject<EditorRefApi | null>;
  workspaceId: string;
  workspaceSlug: string;
  isEditing?: boolean;
  setIsEditing?: (isEditing: boolean) => void;
  renderQuickActions?: (displayName: string) => ReactNode;
};

export const CommentCardDisplay = observer(function CommentCardDisplay(props: TCommentCardDisplayProps) {
  const {
    activityOperations,
    comment,
    disabled,
    projectId,
    readOnlyEditorRef,
    workspaceId,
    workspaceSlug,
    isEditing = false,
    setIsEditing,
    renderQuickActions,
  } = props;
  const { t } = useTranslation();
  const view = useCommentView(workspaceSlug, projectId, comment.issue_id, comment.id);
  const images = useEditorAssets(view, true);
  // states
  const [highlightClassName, setHighlightClassName] = useState("");
  // state
  const [isPickerOpen, setIsPickerOpen] = useState(false);
  const [reactionPending, setReactionPending] = useState(false);
  const reacting = useRef(false);
  // store hooks
  const { getUserDetails } = useMember();
  // derived values
  const userDetails = getUserDetails(comment.actor_id);
  const displayName = userDetails?.display_name ?? t("common.deactivated_user");

  const userReactions = activityOperations.userReactions(comment.id);

  // navigation
  const { pathname } = useLocation();
  // derived values
  const commentBlockId = `comment-${comment?.id}`;
  // Check if there are any reactions to determine if we should render the footer
  const reactionIds = activityOperations.reactionIds(comment.id);
  const hasReactions = reactionIds && Object.keys(reactionIds).some((key) => reactionIds[key]?.length > 0);

  // scroll to comment
  const { isHashMatch } = useHashScroll({
    elementId: commentBlockId,
    pathname,
  });

  useEffect(() => {
    if (!isHashMatch) return;
    setHighlightClassName("border-accent-strong");
    const timeout = setTimeout(() => {
      setHighlightClassName("");
    }, 8000);

    return () => clearTimeout(timeout);
  }, [isHashMatch]);

  const handleEmojiSelect = async (emoji: string) => {
    if (disabled || reacting.current || !userReactions) return;
    reacting.current = true;
    setReactionPending(true);
    try {
      // One lock covers the header picker and the footer. Operations own scope and refusal feedback.
      await activityOperations.react(comment.id, emoji, userReactions);
    } finally {
      reacting.current = false;
      setReactionPending(false);
    }
  };

  return (
    <div className="relative flex flex-col gap-2">
      <div className="relative mb-3 flex w-full items-center gap-2">
        <ProfileAvatar
          alt={displayName}
          fallback={displayName?.[0]?.toUpperCase()}
          size="2xs"
          source={{ kind: "member", workspaceID: workspaceId, memberID: comment.actor_id }}
        />
        <div className="flex flex-1 flex-wrap items-center gap-1">
          <div className="text-caption-sm-medium">{displayName}</div>
          <div className="text-caption-sm-regular text-tertiary">
            commented{" "}
            <Tooltip
              label={`${renderFormattedDate(comment.created_at)} at ${renderFormattedTime(comment.created_at)}`}
              side="bottom"
              layout="single"
              delay={200}
            >
              <span className="text-tertiary">
                {calculateTimeAgo(comment.created_at)}
                {comment.edited_at && " (edited)"}
              </span>
            </Tooltip>
          </div>
        </div>
        {!disabled && (
          <div className="flex shrink-0 items-center gap-1">
            <EmojiReactionPicker
              isOpen={isPickerOpen}
              handleToggle={setIsPickerOpen}
              onChange={(value) => void handleEmojiSelect(value)}
              disabled={reactionPending}
              searchPlaceholder={t("search")}
              buttonClassName="inline-flex size-6 items-center justify-center rounded hover:bg-layer-1"
              label={
                <>
                  <ReactionOutline aria-hidden="true" className="size-4" />
                  <span className="sr-only">{t("aria_labels.comment_add_reaction", { name: displayName })}</span>
                </>
              }
              placement="bottom-start"
            />
            {renderQuickActions ? renderQuickActions(displayName) : null}
          </div>
        )}
      </div>
      {!disabled && isEditing && setIsEditing ? (
        <CommentCardEditForm
          activityOperations={activityOperations}
          comment={comment}
          isEditing={isEditing}
          readOnlyEditorRef={readOnlyEditorRef.current}
          setIsEditing={setIsEditing}
          projectId={projectId}
          workspaceId={workspaceId}
          workspaceSlug={workspaceSlug}
        />
      ) : (
        <>
          <LiteTextEditor
            images={images}
            editable={false}
            ref={readOnlyEditorRef}
            id={comment.id}
            initialValue={comment.comment_html ?? ""}
            workspaceId={workspaceId}
            workspaceSlug={workspaceSlug}
            containerClassName={cn("!py-1 transition-[border-color] duration-500", highlightClassName)}
            projectId={projectId}
            displayConfig={{
              fontSize: "small-font",
            }}
            parentClassName="border-none"
          />
          {hasReactions && (
            <CommentReactions
              comment={comment}
              disabled={disabled}
              pending={reactionPending}
              displayName={displayName}
              onReact={handleEmojiSelect}
              activityOperations={activityOperations}
            />
          )}
        </>
      )}
      <CommentAttachments
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={comment.issue_id}
        commentId={comment.id}
        disabled={disabled}
      />
    </div>
  );
});
