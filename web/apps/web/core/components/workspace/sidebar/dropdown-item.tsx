/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams, Link } from "react-router";
import { SettingsOutline, TickOutline, UserPlusOutline } from "@makeplane/propel/icons";
import { Menu } from "@headlessui/react";
// nerve imports
import { EUserPermissions, ROLE_DETAILS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { Workspace } from "@nerve/api-client";
import { cn } from "@nerve/utils";

import { WorkspaceLogo } from "../logo";

type TProps = {
  workspace: Workspace;
  activeWorkspace: Workspace | null;
  handleItemClick: () => void;
  handleWorkspaceNavigation: (workspace: Workspace) => void;
  handleClose: () => void;
};
const SidebarDropdownItem = observer(function SidebarDropdownItem(props: TProps) {
  const { workspace, activeWorkspace, handleItemClick, handleWorkspaceNavigation, handleClose } = props;
  // router
  const { workspaceSlug } = useParams();
  // hooks
  const { t } = useTranslation();

  return (
    <Link
      key={workspace.id}
      to={`/${workspace.slug}`}
      onClick={() => {
        handleWorkspaceNavigation(workspace);
        handleItemClick();
      }}
      className="w-full"
      id={workspace.id}
    >
      {/* the entry shows as the menu marks it, by the arrows as by the pointer (Headless UI's data-focus): a step
          darker than it rests, the current workspace's too */}
      <Menu.Item
        as="div"
        className={cn("px-4 py-2", {
          "bg-layer-transparent-active data-focus:bg-layer-transparent-selected": workspace.id === activeWorkspace?.id,
          "data-focus:bg-layer-transparent-hover": workspace.id !== activeWorkspace?.id,
        })}
      >
        <div className="flex items-center justify-between gap-1 rounded-sm p-1 text-13 text-primary">
          <div className="relative flex w-[80%] items-center justify-start gap-2.5">
            <WorkspaceLogo
              source={{ kind: "workspace", workspaceID: workspace.id }}
              name={workspace.name}
              classNames="h-8 w-8 border-subtle text-14 font-medium"
            />
            <div className="w-[inherit]">
              <div
                className={`truncate text-left text-13 font-medium text-ellipsis ${workspaceSlug === workspace.slug ? "" : "text-secondary"}`}
              >
                {workspace.name}
              </div>
              <div className="flex w-fit gap-2 text-13 text-tertiary capitalize">
                <span>{t(ROLE_DETAILS[workspace.role].i18n_title)}</span>
                <div className="m-auto h-1 w-1 rounded-full bg-layer-1/50" />
                <span className="capitalize">{t("member", { count: workspace.total_members || 0 })}</span>
              </div>
            </div>
          </div>
          {workspace.id === activeWorkspace?.id ? (
            <span className="flex-shrink-0 p-1">
              <TickOutline className="h-5 w-5 text-primary" />
            </span>
          ) : null}
        </div>
        {workspace.id === activeWorkspace?.id && (
          <>
            <div className="mt-2 mb-1 flex gap-2">
              {[EUserPermissions.ADMIN, EUserPermissions.MEMBER].includes(workspace?.role) && (
                <Link
                  to={`/${workspace.slug}/settings`}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleClose();
                  }}
                  className="flex gap-1.5 rounded-md border border-strong bg-layer-2 px-2.5 py-1.5 text-secondary transition-colors hover:border-strong hover:text-secondary hover:shadow-raised-100"
                >
                  <SettingsOutline className="my-auto h-4 w-4 flex-shrink-0" />
                  <span className="my-auto text-13 font-medium whitespace-nowrap">{t("settings")}</span>
                </Link>
              )}
              {[EUserPermissions.ADMIN].includes(workspace?.role) && (
                <Link
                  to={`/${workspace.slug}/settings/members`}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleClose();
                  }}
                  className="flex gap-1.5 rounded-md border border-strong bg-layer-2 px-2.5 py-1.5 text-secondary transition-colors hover:border-strong hover:text-secondary hover:shadow-raised-100"
                >
                  <UserPlusOutline className="my-auto h-4 w-4 flex-shrink-0" />
                  <span className="my-auto text-13 font-medium whitespace-nowrap">
                    {t("workspace_settings.settings.members.invite_members")}
                  </span>
                </Link>
              )}
            </div>
          </>
        )}
      </Menu.Item>
    </Link>
  );
});

export default SidebarDropdownItem;
