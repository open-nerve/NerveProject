/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { PlusCircleOutline } from "@makeplane/propel/icons";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { SettingsSidebarItem } from "@/components/settings/sidebar/item";
import { WorkspaceLogo } from "@/components/workspace/logo";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useWorkspacesFetch } from "@/hooks/use-workspaces-fetch";

export const ProfileSettingsSidebarWorkspaceOptions = observer(function ProfileSettingsSidebarWorkspaceOptions() {
  // store hooks
  const { workspaces } = useWorkspace();
  // outside a workspace no wrapper has fetched the caller's workspaces: the settings fetch them as they mount (M3
  // design 7.1), for this list and power-K's
  useWorkspacesFetch();
  // translation
  const { t } = useTranslation();

  return (
    <div className="shrink-0">
      <div className="p-2 text-caption-md-medium text-tertiary capitalize">{t("common.workspace")}</div>
      <div className="flex flex-col">
        {(workspaces ?? []).map((workspace) => (
          <SettingsSidebarItem
            key={workspace.id}
            as="link"
            href={`/${workspace.slug}`}
            iconNode={<WorkspaceLogo logo={workspace.logo_url} name={workspace.name} classNames="shrink-0" />}
            label={workspace.name}
            isActive={false}
          />
        ))}
        <div className="mt-1.5">
          <SettingsSidebarItem
            as="link"
            href="/create-workspace"
            icon={PlusCircleOutline}
            label={t("create_workspace")}
            isActive={false}
          />
        </div>
      </div>
    </div>
  );
});
