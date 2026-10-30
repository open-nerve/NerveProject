/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueComment, IssueCommentCreate } from "@nerve/api-client";
import type { TFileSignedURLResponse } from "../../file";

export type TCommentsOperations = {
  copyCommentLink: (commentId: string) => void;
  createComment: (data: IssueCommentCreate) => Promise<IssueComment>;
  updateComment: (commentId: string, data: IssueCommentCreate) => Promise<IssueComment>;
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
  getReactionUsers: (commentId: string, reaction: string) => string[];
};
