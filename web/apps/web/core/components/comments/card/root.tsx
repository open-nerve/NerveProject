/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueComment } from "@nerve/api-client";

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
  entityId: string;
  comment: IssueComment | undefined;
  activityOperations: TCommentsOperations;
  ends: "top" | "bottom" | undefined;
  showCopyLinkOption: boolean;
  enableReplies: boolean;
  disabled?: boolean;
  projectId?: string;
};

export const CommentCard = observer(function CommentCard(props: TCommentCard) {
  const {
    workspaceSlug,
    entityId,
    comment,
    activityOperations,
    ends,
    showCopyLinkOption,
    disabled = false,
    projectId,
  } = props;
  // states
  const [isEditing, setIsEditing] = useState(false);
  // refs
  const readOnlyEditorRef = useRef<EditorRefApi>(null);
  // derived values
  const workspaceId = useWorkspace().getWorkspaceBySlug(workspaceSlug)?.id;

  if (!comment || !workspaceId) return null;

  return (
    <CommentBlock comment={comment} ends={ends}>
      <CommentCardDisplay
        activityOperations={activityOperations}
        entityId={entityId}
        comment={comment}
        disabled={disabled}
        projectId={projectId}
        readOnlyEditorRef={readOnlyEditorRef}
        workspaceId={workspaceId}
        workspaceSlug={workspaceSlug}
        isEditing={isEditing}
        setIsEditing={setIsEditing}
        renderQuickActions={() => (
          <CommentQuickActions
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
