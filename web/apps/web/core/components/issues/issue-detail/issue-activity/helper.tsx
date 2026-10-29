/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EFileAssetType, type TCommentsOperations } from "@nerve/types";
import { copyUrlToClipboard, generateWorkItemLink } from "@nerve/utils";
import { useEditorAsset } from "@/hooks/store/use-editor-asset";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useUser } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { followInSession } from "@/lib/in-session";
import { useDetailScope } from "../../issue-detail-widgets/use-detail-scope";

export const useWorkItemCommentOperations = (
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  issueId: string | undefined
): TCommentsOperations => {
  const detail = useIssueDetail();
  const { getProjectById } = useProject();
  const { getUserDetails } = useMember();
  const { uploadEditorAsset, duplicateEditorAsset } = useEditorAsset();
  const { data: currentUser } = useUser();
  const { t } = useTranslation();
  const refusal = useRefusalToast();
  const capture = useDetailScope(workspaceSlug ?? "", projectId, issueId ?? "");
  const scope = () => {
    if (!workspaceSlug || !projectId || !issueId || !capture()()) throw new SessionChangedError();
    return { ws: workspaceSlug, project: projectId, id: issueId };
  };
  const follow = async (change: () => Promise<unknown>, success?: string) => {
    const current = capture();
    if (!current()) return;
    await followInSession(change, {
      done: () => {
        if (current() && success)
          setToast({ type: TOAST_TYPE.SUCCESS, title: t("common.success"), message: t(success) });
      },
      failed: (error) => {
        if (current() && !(error instanceof SessionChangedError)) refusal(error);
      },
    });
  };
  const ops: TCommentsOperations = {
    copyCommentLink: (id) => {
      void follow(async () => {
        const owner = scope();
        const issue = detail.issue.getIssueById(owner.id);
        if (!issue) throw new SessionChangedError();
        const link = generateWorkItemLink({
          workspaceSlug: owner.ws,
          projectId: issue.project_id,
          issueId: owner.id,
          projectIdentifier: getProjectById(owner.project)?.identifier,
          sequenceId: issue.sequence_id,
        });
        await copyUrlToClipboard(`${link}#comment-${id}`);
      }, "issue.comments.copy_link.success");
    },
    // Forms own confirmation, clearing and refusal; preserve the native command result.
    createComment: async (body) => {
      const { ws, project, id } = scope();
      return detail.createComment(ws, project, id, body);
    },
    updateComment: async (commentId, body) => {
      const { ws, project, id } = scope();
      return detail.updateComment(ws, project, id, commentId, body);
    },
    removeComment: (commentId) =>
      follow(async () => {
        const { ws, project, id } = scope();
        await detail.removeComment(ws, project, id, commentId);
      }, "issue.comments.remove.success"),
    uploadCommentAsset: async (blockId, file, commentId) => {
      const { ws, project } = scope();
      const current = capture();
      const answer = await uploadEditorAsset({
        blockId,
        data: { entity_identifier: commentId ?? "", entity_type: EFileAssetType.COMMENT_DESCRIPTION },
        file,
        projectId: project,
        workspaceSlug: ws,
      });
      if (!current()) throw new SessionChangedError();
      return answer;
    },
    duplicateCommentAsset: async (assetId, commentId) => {
      const { ws, project } = scope();
      const current = capture();
      const answer = await duplicateEditorAsset({
        assetId,
        entityId: commentId || undefined,
        entityType: EFileAssetType.COMMENT_DESCRIPTION,
        projectId: project,
        workspaceSlug: ws,
      });
      if (!current()) throw new SessionChangedError();
      return answer;
    },
    addCommentReaction: (commentId, reaction) =>
      follow(async () => {
        const { ws, project, id } = scope();
        await detail.createCommentReaction(ws, project, id, commentId, reaction);
      }),
    deleteCommentReaction: (commentId, reaction) =>
      follow(async () => {
        const { ws, project, id } = scope();
        await detail.removeCommentReaction(ws, project, id, commentId, reaction);
      }),
    react: async (commentId, reaction, mine) => {
      if (mine.includes(reaction)) await ops.deleteCommentReaction(commentId, reaction);
      else await ops.addCommentReaction(commentId, reaction);
    },
    reactionIds: (commentId) =>
      issueId ? detail.commentReaction.getCommentReactionsByCommentId(issueId, commentId) : undefined,
    userReactions: (commentId) =>
      issueId && currentUser
        ? detail.commentReaction.commentReactionsByUser(issueId, commentId, currentUser.id).map((row) => row.reaction)
        : [],
    getReactionUsers: (commentId, reaction) => {
      if (!issueId) return [];
      const ids = detail.commentReaction.getCommentReactionsByCommentId(issueId, commentId)?.[reaction] ?? [];
      return ids.map((id) => {
        const row = detail.commentReaction.getCommentReactionById(issueId, commentId, id);
        return (row && getUserDetails(row.actor_id)?.display_name) || t("common.deactivated_user");
      });
    },
  };
  return ops;
};
