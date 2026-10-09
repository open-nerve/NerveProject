/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Link } from "react-router";

import { Disclosure } from "@headlessui/react";
// nerve imports
import { ROLE, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { DeactivatedUserOutline, DeleteOutline } from "@makeplane/propel/icons";
import { Pill, EPillVariant, EPillSize } from "@nerve/propel/pill";
import type { User, WorkspaceMember, WorkspaceRole } from "@nerve/api-client";
// nerve ui
import { CustomSelect, PopoverMenu } from "@nerve/ui";
// helpers
import { getFileURL } from "@nerve/utils";
// hooks
import { useUser, useUserPermissions } from "@/hooks/store/user";
// local imports
import { WORKSPACE_ROLES } from "../workspace-roles";
import { useMembershipChanges } from "./use-membership-changes";

type NameProps = {
  rowData: WorkspaceMember;
  workspaceSlug: string;
  isAdmin: boolean;
  currentUser: User | undefined;
  setRemoveMemberModal: (rowData: WorkspaceMember) => void;
};

type AccountTypeProps = {
  rowData: WorkspaceMember;
  workspaceSlug: string;
};

export function NameColumn(props: NameProps) {
  const { rowData, workspaceSlug, isAdmin, currentUser, setRemoveMemberModal } = props;
  // derived values
  const { avatar_url, display_name, email, first_name, id, last_name } = rowData.member;
  const isSuspended = rowData.is_active === false;

  return (
    <Disclosure>
      {() => (
        <div className="group relative">
          <div className="flex w-72 items-center justify-between gap-x-4 gap-y-2">
            <div className="flex flex-1 items-center gap-x-2 gap-y-2">
              {isSuspended ? (
                <div className="rounded-full bg-layer-1">
                  <DeactivatedUserOutline className="size-6 text-placeholder" />
                </div>
              ) : avatar_url && avatar_url.trim() !== "" ? (
                <Link to={`/${workspaceSlug}/profile/${id}`}>
                  <span className="relative flex size-6 items-center justify-center rounded-full text-on-color capitalize">
                    <img
                      src={getFileURL(avatar_url)}
                      className="absolute top-0 left-0 h-full w-full rounded-full object-cover"
                      alt={display_name || (email ?? undefined)}
                    />
                  </span>
                </Link>
              ) : (
                <Link to={`/${workspaceSlug}/profile/${id}`}>
                  <span className="relative flex size-6 items-center justify-center rounded-full bg-layer-3 text-11 text-tertiary capitalize">
                    {(email ?? display_name ?? "?")[0]}
                  </span>
                </Link>
              )}
              <span className={isSuspended ? "text-placeholder" : ""}>
                {first_name} {last_name}
              </span>
            </div>

            {!isSuspended && (isAdmin || id === currentUser?.id) && (
              <PopoverMenu
                data={[""]}
                keyExtractor={(item) => item}
                popoverClassName="justify-end"
                buttonClassName="outline-none	origin-center rotate-90 size-8 aspect-square flex-shrink-0 grid place-items-center opacity-0 group-hover:opacity-100 transition-opacity"
                render={() => (
                  <button
                    type="button"
                    className="flex cursor-pointer items-center gap-x-3"
                    onClick={() => setRemoveMemberModal(rowData)}
                  >
                    <DeleteOutline className="size-3.5 align-middle" /> {id === currentUser?.id ? "Leave " : "Remove "}
                  </button>
                )}
              />
            )}
          </div>
        </div>
      )}
    </Disclosure>
  );
}

export const AccountTypeColumn = observer(function AccountTypeColumn(props: AccountTypeProps) {
  const { rowData, workspaceSlug } = props;
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { data: currentUser } = useUser();
  const { changeRole } = useMembershipChanges(workspaceSlug);

  // derived values
  const isCurrentUser = currentUser?.id === rowData.member.id;
  const isAdminRole = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);
  const isRoleNonEditable = isCurrentUser || !isAdminRole;
  const isSuspended = rowData.is_active === false;

  return (
    <>
      {isSuspended ? (
        <div className="flex w-32">
          <Pill variant={EPillVariant.DEFAULT} size={EPillSize.SM} className="border-none">
            Suspended
          </Pill>
        </div>
      ) : isRoleNonEditable ? (
        <div className="flex w-32">
          <span>{ROLE[rowData.role]}</span>
        </div>
      ) : (
        <CustomSelect
          value={rowData.role}
          // the select gives the picked option's value: a role's number, of WORKSPACE_ROLES
          onChange={(role: WorkspaceRole) => void changeRole(rowData.member.id, role)}
          label={
            <div className="flex">
              <span>{ROLE[rowData.role]}</span>
            </div>
          }
          buttonClassName="!px-0 !justify-start hover:bg-surface-1 border-none"
          className="w-32 rounded-md p-0"
          input
        >
          {WORKSPACE_ROLES.map((role) => (
            <CustomSelect.Option key={role} value={role}>
              {ROLE[role]}
            </CustomSelect.Option>
          ))}
        </CustomSelect>
      )}
    </>
  );
});
