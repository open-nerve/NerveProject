/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Cycle } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { ContextMenu, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useCycleMenuItems } from "@/components/common/quick-actions-helper";
import { useCycles } from "@/hooks/store/use-cycles";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { useCopyLink } from "@/hooks/use-copy-link";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { useCyclePage } from "./actions-host";

type Props = {
  parentRef: React.RefObject<HTMLElement | null>;
  cycleId: string;
  projectId: string;
  workspaceSlug: string;
  customClassName?: string;
};

/**
 * A cycle's quick actions (P5a spec 2.5): its menu, by its button, named by the cycle, and by a right-click on its row
 * (parentRef); none while the store does not give the cycle. Its dialogs are its page's (CycleActionsHost), and what
 * follows a change is the page's while it is mounted, though the change takes the row away.
 */
export const CycleQuickActions = observer(function CycleQuickActions(props: Props) {
  const { getCycleById } = useCycles();
  const cycle = getCycleById(props.cycleId);
  return cycle ? <CycleMenu {...props} cycle={cycle} /> : null;
});

/**
 * The menu of the cycle of the project of projectId (useCycleMenuItems): what the caller may do with it, by his role in
 * the project, as nerve decides it (R8): its admins and members write it; its admins delete it, and so does its creator
 * while the project's admin or member.
 */
const CycleMenu = observer(function CycleMenu(props: Props & { cycle: Cycle }) {
  const { parentRef, cycle, projectId, workspaceSlug, customClassName } = props;
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { unarchive } = useCycles();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { data: currentUser } = useUser();
  const { openDialog, pageMounted } = useCyclePage();
  const copyLink = useCopyLink();
  const toastRefusal = useRefusalToast();
  // auth
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const canWrite = role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER;
  const canDelete = role === EUserPermissions.ADMIN || (canWrite && cycle.created_by_id === currentUser?.id);

  const link = `/${workspaceSlug}/projects/${projectId}/cycles/${cycle.id}`;
  // Restored, the cycle goes back to its project's list, and its row from the archive; the page says so and goes to the
  // archive, which closes a preview. Refused, it says nerve's reason, never success.
  const handleRestore = () =>
    void followInSession(() => unarchive(cycle.id), {
      done: () => {
        if (!pageMounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("project_cycles.action.restore.success.title"),
          message: t("project_cycles.action.restore.success.description"),
        });
        return navigate(`/${workspaceSlug}/projects/${projectId}/archives/cycles`);
      },
      failed: (error) => {
        if (pageMounted.current) toastRefusal(error);
      },
    });
  const MENU_ITEMS = useCycleMenuItems({
    cycle,
    canWrite,
    canDelete,
    handleEdit: () => openDialog("edit", cycle),
    handleArchive: () => openDialog("archive", cycle),
    handleRestore,
    handleDelete: () => openDialog("delete", cycle),
    handleCopyLink: () => void copyLink(link, t("common.link_copied_to_clipboard")),
    handleOpenInNewTab: () => window.open(link, "_blank", "noopener,noreferrer"),
  });

  return (
    <>
      <ContextMenu parentRef={parentRef} items={MENU_ITEMS} />
      <CustomMenu
        ariaLabel={t("common.item_options", { name: cycle.name })}
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
