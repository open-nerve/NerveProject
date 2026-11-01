/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueComment } from "@nerve/api-client";

import { useMemo, useRef, useState } from "react";
import { observer } from "mobx-react";
import { DeleteOutline, EditOutline, LinkOutline, MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { TCommentsOperations } from "@nerve/types";
import type { TContextMenuItem } from "@nerve/ui";
import { CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useUser, useUserPermissions } from "@/hooks/store/user";

type TCommentCard = {
  activityOperations: TCommentsOperations;
  comment: IssueComment;
  setEditMode: () => void;
  showCopyLinkOption: boolean;
  workspaceSlug: string;
  projectId: string | undefined;
  displayName: string;
};

export const CommentQuickActions = observer(function CommentQuickActions(props: TCommentCard) {
  const { activityOperations, comment, setEditMode, showCopyLinkOption, workspaceSlug, projectId, displayName } = props;
  // store hooks
  const { data: currentUser } = useUser();
  const { allowPermissions } = useUserPermissions();
  const sending = useRef(false);
  const [pending, setPending] = useState(false);
  // derived values
  const isAuthor = currentUser?.id === comment.actor_id;
  const canEdit = isAuthor;
  const canDelete =
    isAuthor ||
    (!!projectId &&
      allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, projectId));
  // translation
  const { t } = useTranslation();

  const MENU_ITEMS = useMemo(
    function MENU_ITEMS(): TContextMenuItem[] {
      return [
        {
          key: "edit",
          action: setEditMode,
          title: t("common.actions.edit"),
          icon: EditOutline,
          shouldRender: canEdit,
        },
        {
          key: "copy_link",
          action: () => activityOperations.copyCommentLink(comment.id),
          title: t("common.actions.copy_link"),
          icon: LinkOutline,
          shouldRender: showCopyLinkOption,
        },
        {
          key: "delete",
          action: async () => {
            if (sending.current) return;
            sending.current = true;
            setPending(true);
            try {
              // The operation owns session checks and refusal feedback; the menu owns its busy lifetime.
              await activityOperations.removeComment(comment.id);
            } finally {
              sending.current = false;
              setPending(false);
            }
          },
          title: t("common.actions.delete"),
          icon: DeleteOutline,
          shouldRender: canDelete,
        },
      ].filter((item) => item.shouldRender !== false);
    },
    [t, setEditMode, canEdit, showCopyLinkOption, activityOperations, comment, canDelete]
  );

  if (MENU_ITEMS.length === 0) return null;

  return (
    <CustomMenu
      disabled={pending}
      ariaLabel={t("aria_labels.comment_options", { name: displayName })}
      customButton={<MoreHorizontalOutline className="size-4" aria-hidden="true" />}
      customButtonClassName="inline-flex size-6 items-center justify-center rounded-md text-secondary hover:bg-layer-1 focus-visible:outline-offset-2"
      closeOnSelect
    >
      {MENU_ITEMS.map((item) => (
        <CustomMenu.MenuItem
          key={item.key}
          onClick={() => item.action()}
          className={cn(
            "flex items-center gap-2",
            {
              "text-placeholder": item.disabled,
            },
            item.className
          )}
          disabled={item.disabled}
        >
          {item.icon && <item.icon className={cn("size-3 shrink-0", item.iconClassName)} />}
          <div>
            <h5>{item.title}</h5>
            {item.description && (
              <p
                className={cn("whitespace-pre-line text-tertiary", {
                  "text-placeholder": item.disabled,
                })}
              >
                {item.description}
              </p>
            )}
          </div>
        </CustomMenu.MenuItem>
      ))}
    </CustomMenu>
  );
});
