/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import type { WorkspaceRole } from "@nerve/api-client";
import { ROLE } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { ChevronDownOutline, DeleteOutline, LinkOutline } from "@makeplane/propel/icons";
import type { TContextMenuItem } from "@nerve/ui";
import { CustomSelect, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useCopyInvitationLink } from "@/hooks/use-copy-invitation-link";
// local imports
import { WORKSPACE_ROLES } from "../workspace-roles";
import { useInvitationChanges } from "./use-invitation-changes";

type Props = {
  invitationId: string;
};

export const WorkspaceInvitationsListItem = observer(function WorkspaceInvitationsListItem(props: Props) {
  const { invitationId } = props;
  // router
  const { workspaceSlug } = useParams();
  // states
  const [removeMemberModal, setRemoveMemberModal] = useState(false);
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const {
    workspace: { getWorkspaceInvitationDetails },
  } = useMember();
  const { changeRole, remove } = useInvitationChanges();
  const copyLink = useCopyInvitationLink();
  // derived values: the row shows only to an admin (the members page's gate, decision 4), who may change, copy and
  // delete any pending invitation, and delete a declined one, which nerve keeps from being changed or accepted
  const invitationDetails = getWorkspaceInvitationDetails(invitationId);

  if (!workspaceSlug || !invitationDetails) return null;

  const declined = invitationDetails.responded_at !== null;

  const MENU_ITEMS: TContextMenuItem[] = [
    {
      key: "copy-link",
      action: () => void copyLink(invitationDetails),
      title: t("common.actions.copy_link"),
      icon: LinkOutline,
      shouldRender: !declined,
    },
    {
      key: "remove",
      action: () => {
        setRemoveMemberModal(true);
      },
      title: t("common.remove"),
      icon: DeleteOutline,
      className: "text-danger-primary",
      iconClassName: "text-danger-primary",
    },
  ];

  return (
    <>
      <ConfirmWorkspaceMemberRemove
        isOpen={removeMemberModal}
        onClose={() => setRemoveMemberModal(false)}
        userDetails={{
          id: invitationDetails.id,
          display_name: `${invitationDetails.email}`,
        }}
        onSubmit={() => remove(workspaceSlug, invitationDetails.id)}
      />
      <div className="group flex h-full w-full items-center justify-between px-3 py-4 hover:bg-layer-transparent-hover">
        <div className="flex items-center gap-x-4 gap-y-2">
          <span className="relative flex h-10 w-10 items-center justify-center rounded-sm bg-layer-3 p-4 text-tertiary capitalize">
            {(invitationDetails.email ?? "?")[0]}
          </span>
          <div>
            <h4 className="cursor-default text-body-xs-regular">{invitationDetails.email}</h4>
          </div>
        </div>
        <div className="flex items-center gap-2 text-11">
          <div className="flex items-center justify-center rounded-sm bg-label-yellow-bg-strong/20 px-2.5 py-1 text-center text-caption-sm-medium text-label-yellow-text">
            <p>{declined ? t("workspace_settings.settings.members.declined") : t("common.pending")}</p>
          </div>
          {declined ? (
            <span className="px-2 py-0.5 text-caption-sm-medium">{ROLE[invitationDetails.role]}</span>
          ) : (
            <CustomSelect
              customButton={
                <div className="item-center flex gap-1 rounded-sm px-2 py-0.5">
                  <span className="flex items-center rounded-sm text-caption-sm-medium">
                    {ROLE[invitationDetails.role]}
                  </span>
                  <span className="grid place-items-center">
                    <ChevronDownOutline className="h-3 w-3" />
                  </span>
                </div>
              }
              value={invitationDetails.role}
              // the select gives the picked option's value: a role's number, of WORKSPACE_ROLES
              onChange={(role: WorkspaceRole) => void changeRole(workspaceSlug, invitationDetails.id, role)}
              placement="bottom-end"
            >
              {WORKSPACE_ROLES.map((role) => (
                <CustomSelect.Option key={role} value={role}>
                  {ROLE[role]}
                </CustomSelect.Option>
              ))}
            </CustomSelect>
          )}
          <CustomMenu ellipsis placement="bottom-end" closeOnSelect>
            {MENU_ITEMS.filter((item) => item.shouldRender !== false).map((item) => (
              <CustomMenu.MenuItem
                key={item.key}
                onClick={() => {
                  item.action();
                }}
                className={cn(
                  "flex items-center gap-2",
                  {
                    "text-placeholder": item.disabled,
                  },
                  item.className
                )}
                disabled={item.disabled}
              >
                {item.icon && <item.icon className={cn("h-3 w-3", item.iconClassName)} />}
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
        </div>
      </div>
    </>
  );
});
