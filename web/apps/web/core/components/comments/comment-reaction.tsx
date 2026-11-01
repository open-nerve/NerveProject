/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueComment } from "@nerve/api-client";

import { useMemo, useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { ReactionOutline } from "@makeplane/propel/icons";
import { emojiToString, stringToEmoji } from "@nerve/propel/emoji-icon-picker";
import { EmojiReactionGroup, EmojiReactionPicker } from "@nerve/propel/emoji-reaction";
import type { EmojiReactionType } from "@nerve/propel/emoji-reaction";
import type { TCommentsOperations } from "@nerve/types";

export type TProps = {
  comment: IssueComment;
  disabled?: boolean;
  activityOperations: TCommentsOperations;
  pending: boolean;
  displayName: string;
  onReact: (value: string) => Promise<void>;
};

export const CommentReactions = observer(function CommentReactions(props: TProps) {
  const { comment, activityOperations, disabled = false, pending, displayName, onReact } = props;
  const { t } = useTranslation();
  // state
  const [isPickerOpen, setIsPickerOpen] = useState(false);

  const userReactions = activityOperations.userReactions(comment.id);
  const reactionIds = activityOperations.reactionIds(comment.id);

  // Transform reactions data to Propel EmojiReactionType format
  const reactions: EmojiReactionType[] = useMemo(() => {
    if (!reactionIds) return [];

    return Object.keys(reactionIds)
      .filter((reaction) => reactionIds[reaction]?.length > 0)
      .map((reaction) => {
        const users = activityOperations.getReactionUsers(comment.id, reaction);

        return {
          emoji: stringToEmoji(reaction),
          count: reactionIds[reaction].length,
          reacted: userReactions?.includes(reaction) || false,
          users: users,
        };
      });
  }, [reactionIds, userReactions, activityOperations, comment.id]);

  if (!userReactions) return null;

  // Don't render anything if there are no reactions and it's disabled
  if (reactions.length === 0 && disabled) return null;

  const label = t("aria_labels.comment_add_reaction", { name: displayName });
  return (
    <fieldset disabled={disabled || pending} className="relative flex min-w-0 items-center gap-2">
      <EmojiReactionGroup
        reactions={reactions}
        onReactionClick={(emoji) => void onReact(emojiToString(emoji))}
        showAddButton={false}
        addReactionLabel={label}
      />
      {!disabled && (
        <EmojiReactionPicker
          isOpen={isPickerOpen}
          handleToggle={setIsPickerOpen}
          onChange={(value) => void onReact(value)}
          disabled={pending}
          searchPlaceholder={t("search")}
          buttonClassName="inline-flex size-6 items-center justify-center rounded hover:bg-layer-1"
          label={
            <>
              <ReactionOutline aria-hidden="true" className="size-4" />
              <span className="sr-only">{label}</span>
            </>
          }
          placement="bottom-start"
        />
      )}
    </fieldset>
  );
});
