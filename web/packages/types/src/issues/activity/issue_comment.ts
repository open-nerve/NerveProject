/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { JSONContent } from "../../editor";
import type { TFileSignedURLResponse } from "../../file";
import type {
  TIssueActivityWorkspaceDetail,
  TIssueActivityProjectDetail,
  TIssueActivityIssueDetail,
  TIssueActivityUserDetail,
} from "./base";

export type TIssueComment = {
  id: string;
  workspace: string;
  workspace_detail: TIssueActivityWorkspaceDetail;
  project: string;
  project_detail: TIssueActivityProjectDetail;
  issue: string;
  issue_detail: TIssueActivityIssueDetail;
  actor: string;
  actor_detail: TIssueActivityUserDetail;
  created_at: string;
  edited_at?: string | undefined;
  updated_at: string;
  created_by: string | undefined;
  updated_by: string | undefined;
  attachments: any[];
  comment_reactions: any[];
  comment_stripped: string;
  comment_html: string;
  comment_json: JSONContent;
  external_id: string | undefined;
  external_source: string | undefined;
};

export type TCommentsOperations = {
  copyCommentLink: (commentId: string) => void;
  createComment: (data: Partial<TIssueComment>) => Promise<Partial<TIssueComment> | undefined>;
  updateComment: (commentId: string, data: Partial<TIssueComment>) => Promise<void>;
  removeComment: (commentId: string) => Promise<void>;
  uploadCommentAsset: (blockId: string, file: File, commentId?: string) => Promise<TFileSignedURLResponse>;
  duplicateCommentAsset: (assetId: string, commentId?: string) => Promise<{ asset_id: string }>;
  addCommentReaction: (commentId: string, reactionEmoji: string) => Promise<void>;
  deleteCommentReaction: (commentId: string, reactionEmoji: string) => Promise<void>;
  react: (commentId: string, reactionEmoji: string, userReactions: string[]) => Promise<void>;
  reactionIds: (commentId: string) =>
    | {
        [reaction: string]: string[];
      }
    | undefined;
  userReactions: (commentId: string) => string[] | undefined;
  getReactionUsers: (reaction: string, reactionIds: Record<string, string[]>) => string;
};

export type TIssueCommentMap = {
  [issue_id: string]: TIssueComment;
};

export type TIssueCommentIdMap = {
  [issue_id: string]: string[];
};
