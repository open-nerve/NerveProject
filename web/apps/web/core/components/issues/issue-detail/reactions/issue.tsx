/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useRef, useState } from "react";
import { observer } from "mobx-react";
import type { User } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { stringToEmoji } from "@nerve/propel/emoji-icon-picker";
import { EmojiReactionGroup, EmojiReactionPicker } from "@nerve/propel/emoji-reaction";
import type { EmojiReactionType } from "@nerve/propel/emoji-reaction";
import { cn } from "@nerve/utils";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useMember } from "@/hooks/store/use-member";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";
import { DetailResourceError } from "@/components/issues/issue-detail-widgets/resource-error";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";

type IssueReactionProps = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  currentUser: User;
  disabled?: boolean;
  className?: string;
};

export function IssueReaction(props: IssueReactionProps) {
  return (
    <IssueReactionContent
      key={`${props.workspaceSlug}/${props.projectId}/${props.issueId}/${props.currentUser.id}`}
      {...props}
    />
  );
}

const IssueReactionContent = observer(function IssueReactionContent(props: IssueReactionProps) {
  const { workspaceSlug, projectId, issueId, currentUser, disabled = false, className = "" } = props;
  const { t } = useTranslation();
  const [isPickerOpen, setIsPickerOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const sending = useRef(false);
  const detail = useIssueDetail();
  const { reaction } = detail;
  const { getUserDetails } = useMember();
  const refuse = useRefusalToast();
  const scope = useDetailScope(workspaceSlug, projectId, issueId);
  const reactionIds = reaction.getReactionsByIssueId(issueId);
  const userReactions = new Set(reaction.reactionsByUser(issueId, currentUser.id).map((row) => row.reaction));

  const react = async (value: string) => {
    const current = scope();
    if (disabled || sending.current || !current()) return;
    sending.current = true;
    setPending(true);
    await followInSession(
      async () =>
        userReactions.has(value)
          ? detail.removeReaction(workspaceSlug, projectId, issueId, value)
          : detail.createReaction(workspaceSlug, projectId, issueId, value),
      {
        failed: (error) => {
          if (current()) refuse(error);
        },
      }
    );
    // Busy state belongs to this keyed host, even if its permission proof expired.
    sending.current = false;
    setPending(false);
  };
  const reactions: EmojiReactionType[] = Object.entries(reactionIds ?? {}).map(([value, ids]) => ({
    emoji: stringToEmoji(value),
    count: ids.length,
    reacted: userReactions.has(value),
    users: ids.map((id) => {
      const row = reaction.getReactionById(issueId, id);
      return (row && getUserDetails(row.actor_id)?.display_name) || t("common.deactivated_user");
    }),
  }));
  return (
    <div className={cn("relative mt-4", className)}>
      <DetailResourceError
        error={reaction.getError(issueId)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        retry={() => detail.fetchReactions(workspaceSlug, projectId, issueId)}
      />
      <EmojiReactionPicker
        isOpen={isPickerOpen}
        handleToggle={setIsPickerOpen}
        onChange={(value) => void react(value)}
        disabled={disabled || pending}
        label={
          <EmojiReactionGroup
            reactions={reactions}
            onReactionClick={(emoji) => void react(Array.from(emoji, (char) => char.codePointAt(0)).join("-"))}
            showAddButton={!disabled}
            onAddReaction={() => {
              if (!pending) setIsPickerOpen(true);
            }}
            addReactionLabel={t("aria_labels.add_reaction")}
          />
        }
        placement="bottom-start"
      />
    </div>
  );
});
