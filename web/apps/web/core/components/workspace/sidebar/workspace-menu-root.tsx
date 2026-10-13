/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useState, useEffect } from "react";
import { observer } from "mobx-react";
import { Link } from "react-router";
// icons
import { ChevronDownOutline, LogOutOutline, PlusCircleOutline } from "@makeplane/propel/icons";
// ui
import { Menu, Transition } from "@headlessui/react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import type { Workspace } from "@nerve/api-client";
import { Loader } from "@nerve/ui";
import { orderWorkspacesList, cn } from "@nerve/utils";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserProfile } from "@/hooks/store/user";
import { useInstance } from "@/hooks/store/use-instance";
import { useSignOut } from "@/hooks/use-sign-out";
// components
import { LeaveWorkspaceModal } from "../leave-workspace-modal";
import { WorkspaceLogo } from "../logo";
import SidebarDropdownItem from "./dropdown-item";

/** The workspace menu of the top navigation: the account's workspaces, a new one, leaving this one, signing out. */
export const WorkspaceMenuRoot = observer(function WorkspaceMenuRoot() {
  // store hooks
  const { toggleSidebar, toggleAnySidebarDropdown } = useAppTheme();
  const { config } = useInstance();
  const { data: currentUser } = useUser();
  const signOut = useSignOut();
  const { updateUserProfile } = useUserProfile();
  const { currentWorkspace: activeWorkspace, workspaces } = useWorkspace();
  // derived values
  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;
  // translation
  const { t } = useTranslation();
  // local state
  const [isWorkspaceMenuOpen, setIsWorkspaceMenuOpen] = useState(false);
  // the leaving's dialog outlives the menu, which closes as its entry is picked
  const [isLeaveModalOpen, setIsLeaveModalOpen] = useState(false);

  // the workspace opened last is a best-effort preference: the link opens the workspace whether nerve saves it or not
  const handleWorkspaceNavigation = (workspace: Workspace) => {
    void updateUserProfile({ last_workspace_id: workspace?.id }).catch(() => undefined);
  };

  const handleSignOut = () => void signOut();

  const handleItemClick = () => {
    if (window.innerWidth < 768) {
      toggleSidebar();
    }
  };
  const workspacesList = workspaces && orderWorkspacesList(workspaces);

  // Toggle sidebar dropdown state when the menu is open
  useEffect(() => {
    toggleAnySidebarDropdown(isWorkspaceMenuOpen);
  }, [isWorkspaceMenuOpen, toggleAnySidebarDropdown]);

  return (
    <>
      <Menu
        as="div"
        className="relative flex h-full w-fit max-w-48 flex-grow justify-stretch truncate text-left whitespace-nowrap"
      >
        {({ open, close }: { open: boolean; close: () => void }) => {
          // Update local state directly
          if (isWorkspaceMenuOpen !== open) {
            setIsWorkspaceMenuOpen(open);
          }

          return (
            <>
              {/* as the keyboard focuses it (Headless UI's data-focus), the ring @nerve/ui's selects show on their button,
                  inside its edge: the menu's root clips what is outside it */}
              <Menu.Button
                className={cn(
                  "group/menu-button flex flex-grow items-center justify-between gap-1 truncate rounded-sm p-1 text-13 font-medium text-secondary hover:bg-layer-1 focus:outline-none data-focus:outline-1 data-focus:-outline-offset-1 data-focus:outline-accent-strong data-focus:outline-solid",
                  {
                    "bg-layer-1": open,
                  }
                )}
                aria-label={t("aria_labels.projects_sidebar.open_workspace_switcher")}
              >
                <div className="flex flex-grow items-center gap-2 truncate">
                  <WorkspaceLogo
                    logo={activeWorkspace?.logo_url}
                    name={activeWorkspace?.name}
                    classNames="border border-subtle rounded-md size-7"
                  />
                  <h4 className="truncate text-14 font-medium text-primary">{activeWorkspace?.name ?? t("loading")}</h4>
                </div>
                <ChevronDownOutline
                  className={cn("size-4 flex-shrink-0 text-placeholder duration-300", {
                    "rotate-180": open,
                  })}
                />
              </Menu.Button>
              <Transition
                as={Fragment}
                enter="transition ease-out duration-100"
                enterFrom="transform opacity-0 scale-95"
                enterTo="transform opacity-100 scale-100"
                leave="transition ease-in duration-75"
                leaveFrom="transform opacity-100 scale-100"
                leaveTo="transform opacity-0 scale-95"
              >
                <Menu.Items as={Fragment}>
                  <div className="fixed top-10 left-4 z-21 mt-1 flex w-[19rem] origin-top-left flex-col divide-y divide-subtle rounded-md border-[0.5px] border-strong bg-surface-1 shadow-raised-200 outline-none">
                    <div className="vertical-scrollbar scrollbar-sm flex max-h-96 flex-col items-start justify-start overflow-x-hidden overflow-y-scroll">
                      <span className="sticky top-0 z-21 h-full w-full flex-shrink-0 truncate rounded-md bg-surface-1 px-4 pt-3 pb-1 text-left text-13 font-medium text-placeholder">
                        {currentUser?.email}
                      </span>
                      {workspacesList ? (
                        <div className="flex size-full flex-col items-start justify-start">
                          {(activeWorkspace
                            ? [
                                activeWorkspace,
                                ...workspacesList.filter((workspace) => workspace.id !== activeWorkspace?.id),
                              ]
                            : workspacesList
                          ).map((workspace) => (
                            <SidebarDropdownItem
                              key={workspace.id}
                              workspace={workspace}
                              activeWorkspace={activeWorkspace}
                              handleItemClick={handleItemClick}
                              handleWorkspaceNavigation={handleWorkspaceNavigation}
                              handleClose={close}
                            />
                          ))}
                        </div>
                      ) : (
                        <div className="w-full">
                          <Loader className="space-y-2">
                            <Loader.Item height="30px" />
                            <Loader.Item height="30px" />
                          </Loader>
                        </div>
                      )}
                    </div>
                    {/* each item shows as the menu marks it, by the arrows as by the pointer (Headless UI's data-focus) */}
                    <div className="flex w-full flex-col items-start justify-start gap-2 px-4 py-2 text-13">
                      {!isWorkspaceCreationDisabled && (
                        <Link to="/create-workspace" className="w-full">
                          <Menu.Item
                            as="div"
                            className="flex items-center gap-2 rounded-sm px-2 py-1 text-13 font-medium text-secondary data-focus:bg-layer-transparent-hover"
                          >
                            <PlusCircleOutline className="size-4 flex-shrink-0" />
                            {t("create_workspace")}
                          </Menu.Item>
                        </Link>
                      )}
                      {/* every member may leave the workspace he is in; nerve refuses its only admin (workspace.sole_admin)
                          and the only admin of one of its projects that has other members (project.sole_admin), M3
                          design 3.7 */}
                      {activeWorkspace && (
                        <Menu.Item
                          as="button"
                          type="button"
                          className="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-13 font-medium text-secondary data-focus:bg-layer-transparent-hover"
                          onClick={() => setIsLeaveModalOpen(true)}
                        >
                          <LogOutOutline className="size-4 flex-shrink-0" />
                          {t("leave_workspace")}
                        </Menu.Item>
                      )}

                      <div className="w-full">
                        <Menu.Item
                          as="button"
                          type="button"
                          className="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-13 font-medium text-danger-primary data-focus:bg-layer-transparent-hover"
                          onClick={handleSignOut}
                        >
                          <LogOutOutline className="size-4 flex-shrink-0" />
                          {t("sign_out")}
                        </Menu.Item>
                      </div>
                    </div>
                  </div>
                </Menu.Items>
              </Transition>
            </>
          );
        }}
      </Menu>
      {activeWorkspace && (
        <LeaveWorkspaceModal
          workspaceSlug={activeWorkspace.slug}
          isOpen={isLeaveModalOpen}
          onClose={() => setIsLeaveModalOpen(false)}
        />
      )}
    </>
  );
});
