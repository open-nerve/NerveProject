/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueComment } from "@nerve/api-client";

import { useIssueDetailInput } from "@/hooks/use-issue-detail-input";
import { useRef, useState } from "react";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { observer } from "mobx-react";
// nerve imports
import type { EditorRefApi } from "@nerve/editor";
import type { TCommentsOperations } from "@nerve/types";
// local imports
import { CommentQuickActions } from "../quick-actions";
import { CommentBlock } from "../comment-block";
import { CommentCardDisplay } from "./display";

type TCommentCard = {
  workspaceSlug: string;
  comment: IssueComment | undefined;
  activityOperations: TCommentsOperations;
  ends: "top" | "bottom" | undefined;
  showCopyLinkOption: boolean;
  disabled?: boolean;
  projectId?: string;
};

export const CommentCard = observer(function CommentCard(props: TCommentCard) {
  const { workspaceSlug, comment, activityOperations, ends, showCopyLinkOption, disabled = false, projectId } = props;
  // states
  const [localEditing, setLocalEditing] = useState(false);
  const retained = useIssueDetailInput()?.form(`edit:${comment?.id}`);
  const isEditing = retained?.read()?.editing ?? localEditing;
  const setIsEditing = (editing: boolean) => {
    if (comment)
      retained?.store({
        ...(retained.read() ?? {
          html: comment.comment_html,
          baseHTML: comment.comment_html,
          attachmentIDs: [],
          revision: 0,
        }),
        editing,
      });
    setLocalEditing(editing);
  };
  // refs
  const readOnlyEditorRef = useRef<EditorRefApi>(null);
  // derived values
  const workspaceId = useWorkspace().getWorkspaceBySlug(workspaceSlug)?.id;

  if (!comment || !workspaceId) return null;

  return (
    <CommentBlock comment={comment} ends={ends}>
      <CommentCardDisplay
        activityOperations={activityOperations}
        comment={comment}
        disabled={disabled}
        projectId={projectId}
        readOnlyEditorRef={readOnlyEditorRef}
        workspaceId={workspaceId}
        workspaceSlug={workspaceSlug}
        isEditing={isEditing}
        setIsEditing={setIsEditing}
        renderQuickActions={(displayName) => (
          <CommentQuickActions
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            displayName={displayName}
            activityOperations={activityOperations}
            comment={comment}
            setEditMode={() => setIsEditing(true)}
            showCopyLinkOption={showCopyLinkOption}
          />
        )}
      />
    </CommentBlock>
  );
});
