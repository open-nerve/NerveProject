/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { UserAltOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { ProfileAvatar } from "@/components/assets/profile-image";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";

/**
 * A module's lead in its row and its card (P6a spec 2.6): the account's avatar, named by the account as the workspace's
 * members give it (getUserDetails, never the router's address), or as a former member once it is not there; else
 * that the module has no lead.
 */
export const ModuleLeadAvatar = observer(function ModuleLeadAvatar({ leadId }: { leadId: string | null }) {
  // hooks
  const { t } = useTranslation();
  const imageWorkspace = useRouteWorkspace();
  const { getUserDetails } = useMember();

  if (leadId === null)
    return (
      <Tooltip label={t("project_module.no_lead")}>
        <span role="img" aria-label={t("project_module.no_lead")} className="flex">
          <UserAltOutline className="h-4 w-4 text-tertiary" aria-hidden="true" />
        </span>
      </Tooltip>
    );

  const name = getUserDetails(leadId)?.display_name ?? t("distribution.former_member");
  return (
    <span className="flex cursor-default" title={name}>
      <ProfileAvatar
        source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: leadId }}
        alt={name}
        fallback={name[0]?.toUpperCase()}
        size="xs"
      />
    </span>
  );
});
