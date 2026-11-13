/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "react-router";
import { LinkOutline, MembersOutline } from "@makeplane/propel/icons";
// nerve imports
import { MODULE_STATUS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { ModuleStatusIcon } from "@nerve/propel/icons";
// components
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { useUserPermissions } from "@/hooks/store/user";
import { useCopyLink } from "@/hooks/use-copy-link";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * The commands on a module's page (P6a spec 2.9, R5, R6): its members and its status, changed while nerve takes the
 * caller's writes of the route's project's modules (canWriteModules) and the module is not archived (nerve refuses
 * any change of an archived one). A choice of a member adds him or removes him: the store is given whom, and makes
 * nerve's whole set of members in its turn, so that a change sent before it is not lost. nerve's answer the store
 * shows; its refusal is said, while the tab is in the session the change was sent in (followInSession). The copy of
 * the link of the route's module, through the web app's one copy, is anyone's who sees the page. The modules'
 * favorites are gone until M7 brings them back (M6 design 15 #13).
 */
export const usePowerKModuleContextBasedActions = (): TPowerKCommandConfig[] => {
  // navigation
  const { workspaceSlug, projectId, moduleId } = useParams();
  // store hooks
  const { getModuleById, update, changeMembers } = useModules();
  const { canWriteModules } = useUserPermissions();
  // hooks
  const copyLink = useCopyLink();
  const toastRefusal = useRefusalToast();
  const { t } = useTranslation();
  // derived values
  const module = getModuleById(moduleId);
  const editable = () => !!module && !module.archived_at && canWriteModules(workspaceSlug ?? "", projectId);

  /** Sends change of the module the page shows, while the caller may change it. */
  const send = (change: (id: string) => Promise<unknown>) => {
    if (!module || !editable()) return;
    const id = module.id;
    void followInSession(() => change(id), { failed: toastRefusal });
  };

  /** Adds the member, or removes him if he is one, as the menu shows him chosen. */
  const toggleMember = (memberId: unknown) => {
    if (!module || typeof memberId !== "string" || !memberId) return;
    const change = module.member_ids.includes(memberId)
      ? { add: [], remove: [memberId] }
      : { add: [memberId], remove: [] };
    send((id) => changeMembers(id, change));
  };

  return [
    {
      id: "add_remove_module_members",
      i18n_title: "power_k.contextual_actions.module.add_remove_members",
      icon: MembersOutline,
      group: "contextual",
      contextType: "module",
      type: "change-page",
      page: "update-module-member",
      onSelect: toggleMember,
      shortcut: "m",
      isEnabled: editable,
      isVisible: editable,
      closeOnSelect: false,
    },
    {
      id: "change_module_status",
      i18n_title: "power_k.contextual_actions.module.change_status",
      iconNode: <ModuleStatusIcon status="backlog" className="size-3.5 shrink-0" />,
      group: "contextual",
      contextType: "module",
      type: "change-page",
      page: "update-module-status",
      onSelect: (data) => {
        const status = MODULE_STATUS.find((option) => option.value === data)?.value;
        if (status) send((id) => update(id, { status }));
      },
      shortcut: "s",
      isEnabled: editable,
      isVisible: editable,
      closeOnSelect: true,
    },
    {
      id: "copy_module_url",
      i18n_title: "power_k.contextual_actions.module.copy_url",
      icon: LinkOutline,
      group: "contextual",
      contextType: "module",
      type: "action",
      action: () =>
        copyLink(
          `/${workspaceSlug}/projects/${projectId}/modules/${moduleId}`,
          t("power_k.contextual_actions.module.copy_url_toast_success")
        ),
      modifierShortcut: "cmd+shift+,",
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
  ];
};
