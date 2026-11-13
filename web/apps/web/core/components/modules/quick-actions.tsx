/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Module } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { ContextMenu, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useModuleMenuItems } from "@/components/common/quick-actions-helper";
import { useModules } from "@/hooks/store/use-modules";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { useCopyLink } from "@/hooks/use-copy-link";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { useModulePage } from "./actions-host";

type Props = {
  parentRef: React.RefObject<HTMLElement | null>;
  moduleId: string;
  projectId: string;
  workspaceSlug: string;
  customClassName?: string;
};

/**
 * A module's quick actions (P6a spec 2.6): its menu, by its button, named by the module, and by a right-click on its
 * row (parentRef); none while the store does not give the module. Its dialogs are its page's (ModuleActionsHost), and
 * what follows a change is the page's while it is mounted, though the change takes the row away.
 */
export const ModuleQuickActions = observer(function ModuleQuickActions(props: Props) {
  const { getModuleById } = useModules();
  const module = getModuleById(props.moduleId);
  return module ? <ModuleMenu {...props} module={module} /> : null;
});

/**
 * The menu of the module of the project of projectId (useModuleMenuItems): what the caller may do with it, as nerve
 * decides it (R5; canWriteModules, canDeleteModule): its admins and members write it; its admins delete it, and so does
 * its creator while the project's admin or member; neither one whom nerve shows only his own work items, nor anyone
 * while the project's modules are turned off.
 */
const ModuleMenu = observer(function ModuleMenu(props: Props & { module: Module }) {
  const { parentRef, module, projectId, workspaceSlug, customClassName } = props;
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { unarchive } = useModules();
  const { canWriteModules, canDeleteModule } = useUserPermissions();
  const { data: currentUser } = useUser();
  const { openDialog, pageMounted } = useModulePage();
  const copyLink = useCopyLink();
  const toastRefusal = useRefusalToast();
  // states
  const restoring = useRef(false);
  // auth
  const canWrite = canWriteModules(workspaceSlug, projectId);
  const canDelete = canDeleteModule(workspaceSlug, projectId, module.created_by_id === currentUser?.id);

  const link = `/${workspaceSlug}/projects/${projectId}/modules/${module.id}`;
  // Restored, the module goes back to its project's list, and its row from the archive; the page says so and goes to
  // the archive, which closes a preview. Refused, it says nerve's reason, never success. While its restore is out, the
  // module's restore asked again (by its button's menu or its right-click menu, which share the items) sends nothing.
  const handleRestore = async () => {
    if (restoring.current) return;
    restoring.current = true;
    await followInSession(() => unarchive(module.id), {
      done: () => {
        if (!pageMounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("project_module.action.restore.success.title"),
          message: t("project_module.action.restore.success.description"),
        });
        return navigate(`/${workspaceSlug}/projects/${projectId}/archives/modules`);
      },
      failed: (error) => {
        if (pageMounted.current) toastRefusal(error);
      },
    });
    restoring.current = false;
  };
  const MENU_ITEMS = useModuleMenuItems({
    module,
    canWrite,
    canDelete,
    handleEdit: () => openDialog("edit", module),
    handleArchive: () => openDialog("archive", module),
    handleRestore: () => void handleRestore(),
    handleDelete: () => openDialog("delete", module),
    handleCopyLink: () => void copyLink(link, t("common.link_copied_to_clipboard")),
    handleOpenInNewTab: () => window.open(link, "_blank", "noopener,noreferrer"),
  });

  return (
    <>
      <ContextMenu parentRef={parentRef} items={MENU_ITEMS} />
      <CustomMenu
        ariaLabel={t("common.item_options", { name: module.name })}
        customButton={<MoreHorizontalOutline className="size-4" aria-hidden="true" />}
        customButtonClassName={cn(getIconButtonStyling("tertiary", "lg"), customClassName)}
        placement="bottom-end"
        closeOnSelect
        maxHeight="lg"
      >
        {MENU_ITEMS.map((item) => (
          <CustomMenu.MenuItem
            key={item.key}
            onClick={() => item.action()}
            className={cn("flex items-center gap-2", { "text-placeholder": item.disabled }, item.className)}
            disabled={item.disabled}
          >
            {item.icon && <item.icon className={cn("h-3 w-3 flex-shrink-0", item.iconClassName)} />}
            <div>
              <h5>{item.title}</h5>
              {item.description && (
                <p className={cn("whitespace-pre-line text-tertiary", { "text-placeholder": item.disabled })}>
                  {item.description}
                </p>
              )}
            </div>
          </CustomMenu.MenuItem>
        ))}
      </CustomMenu>
    </>
  );
});
