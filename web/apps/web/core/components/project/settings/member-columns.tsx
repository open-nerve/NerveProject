/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Link } from "react-router";
import { CircleMinus } from "lucide-react";
import { Disclosure } from "@headlessui/react";
// nerve imports
import { ROLE_DETAILS } from "@nerve/constants";
import type { ProjectRole } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { CustomMenu, CustomSelect } from "@nerve/ui";
import { getFileURL } from "@nerve/utils";
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
// local imports
import { useProjectMembershipChanges } from "./use-project-membership-changes";

type NameProps = {
  rowData: IProjectMemberDetails;
  workspaceSlug: string;
  /** Whether the membership is the caller's own, which he may leave. */
  own: boolean;
  /** Whether the caller may remove the membership (canRemove). */
  removable: boolean;
  setRemoveMemberModal: (rowData: IProjectMemberDetails) => void;
};

type AccountTypeProps = {
  rowData: IProjectMemberDetails;
  /** The roles the caller may give the membership (roleChoices): none shows its role alone. */
  choices: ProjectRole[];
  workspaceSlug: string;
  projectId: string;
};

export function NameColumn(props: NameProps) {
  const { rowData, workspaceSlug, own, removable, setRemoveMemberModal } = props;
  // derived values
  const { avatar_url, display_name, email, first_name, id, last_name } = rowData.member;

  return (
    <Disclosure>
      {() => (
        <div className="group relative">
          <div className="flex w-72 items-center gap-2">
            <div className="flex flex-1 items-center gap-x-2 gap-y-2">
              {avatar_url && avatar_url.trim() !== "" ? (
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
                  <span className="relative flex size-6 items-center justify-center rounded-full bg-layer-3 text-11 text-on-color capitalize">
                    {(email ?? display_name ?? "?")[0]}
                  </span>
                </Link>
              )}
              {first_name} {last_name}
            </div>
            {(own || removable) && (
              <CustomMenu
                ellipsis
                buttonClassName="p-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
                optionsClassName="p-1.5"
                placement="bottom-end"
              >
                <CustomMenu.MenuItem onClick={() => setRemoveMemberModal(rowData)}>
                  <div className="flex items-center gap-x-1 font-medium text-danger-primary">
                    <CircleMinus className="size-3.5 flex-shrink-0" />
                    {own ? "Leave " : "Remove "}
                  </div>
                </CustomMenu.MenuItem>
              </CustomMenu>
            )}
          </div>
        </div>
      )}
    </Disclosure>
  );
}

export const AccountTypeColumn = observer(function AccountTypeColumn(props: AccountTypeProps) {
  const { rowData, choices, projectId, workspaceSlug } = props;
  // store hooks
  const { changeRole } = useProjectMembershipChanges(workspaceSlug, projectId);
  // translation
  const { t } = useTranslation();
  // derived values
  const roleLabel = t(ROLE_DETAILS[rowData.role].i18n_title);

  return choices.length > 0 ? (
    <CustomSelect
      value={rowData.role}
      // the select gives the picked option's value: a role's number, of the choices
      onChange={(role: ProjectRole) => void changeRole(rowData.member.id, role)}
      label={
        <div className="flex">
          <span>{roleLabel}</span>
        </div>
      }
      buttonClassName="!px-0 !justify-start hover:bg-surface-1 border-none"
      className="w-32 rounded-md p-0"
      input
    >
      {choices.map((role) => (
        <CustomSelect.Option key={role} value={role}>
          {t(ROLE_DETAILS[role].i18n_title)}
        </CustomSelect.Option>
      ))}
    </CustomSelect>
  ) : (
    <div className="flex w-32">
      <span>{roleLabel}</span>
    </div>
  );
});
