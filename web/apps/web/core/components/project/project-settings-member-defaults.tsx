/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { ProjectUpdate } from "@nerve/api-client";
import { Switch } from "@makeplane/propel/components/switch";
import { Loader } from "@nerve/ui";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { MemberSelect } from "./member-select";

/** A change of one of the project's member defaults: the one field changed (ProjectUpdate is a partial change). */
type TMemberDefault = Pick<ProjectUpdate, "project_lead_id"> | Pick<ProjectUpdate, "default_assignee_id">;

/** The member a select names, or none for its "none". */
const chosen = (value: string) => (value === "none" ? null : value);

type TDefaultSettingItemProps = {
  title: string;
  description: string;
  children: ReactNode;
};

function DefaultSettingItem({ title, description, children }: TDefaultSettingItemProps) {
  return (
    <div className="flex items-center justify-between gap-x-2">
      <div className="flex flex-col gap-0.5">
        <h4 className="text-13 font-medium">{title}</h4>
        <p className="text-11 text-tertiary">{description}</p>
      </div>
      <div className="w-full max-w-48 sm:max-w-64">{children}</div>
    </div>
  );
}

type TProjectSettingsMemberDefaultsProps = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The project's lead, its default assignee (each one of its members who is not its guest, M3 design 7.6) and its
 * guests' view of every work item, as nerve last answered them: a change shows once nerve has made it. The page
 * follows each change only in the session it was sent in (M3 design 7.1), a refusal with nerve's reason in a toast.
 */
export const ProjectSettingsMemberDefaults = observer(function ProjectSettingsMemberDefaults(
  props: TProjectSettingsMemberDefaultsProps
) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { getProjectById, updateProject, toggleProject } = useProject();
  const toastRefusal = useRefusalToast();
  // derived values
  const project = getProjectById(projectId);
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, projectId);

  const followers = {
    done: () =>
      setToast({
        title: `${t("success")}!`,
        type: TOAST_TYPE.SUCCESS,
        message: t("project_settings.general.toast.success"),
      }),
    failed: toastRefusal,
  };

  /**
   * Changes the project's lead or its default assignee: data is the one field changed, as ProjectUpdate is a partial
   * change and nerve keeps the other as it has it (v0 design 7.7).
   */
  const submitChanges = (data: TMemberDefault) => followInSession(() => updateProject(projectId, data), followers);

  /** Turns the guests' view of every work item the other way, from nerve's last answer, in the change's turn. */
  const toggleGuestViewAllIssues = () =>
    followInSession(() => toggleProject(projectId, "guest_view_all_features"), followers);

  return (
    <div className="my-6 flex flex-col gap-y-6">
      <DefaultSettingItem title="Project Lead" description="Select the project lead for the project.">
        {project ? (
          <MemberSelect
            value={project.project_lead_id}
            onChange={(val: string) => void submitChanges({ project_lead_id: chosen(val) })}
            isDisabled={!isAdmin}
          />
        ) : (
          <Loader className="h-9 w-full">
            <Loader.Item width="100%" height="100%" />
          </Loader>
        )}
      </DefaultSettingItem>
      <DefaultSettingItem title="Default Assignee" description="Select the default assignee for the project.">
        {project ? (
          <MemberSelect
            value={project.default_assignee_id}
            onChange={(val: string) => void submitChanges({ default_assignee_id: chosen(val) })}
            isDisabled={!isAdmin}
          />
        ) : (
          <Loader className="h-9 w-full">
            <Loader.Item width="100%" height="100%" />
          </Loader>
        )}
      </DefaultSettingItem>
      {project && (
        <DefaultSettingItem
          title="Guest access"
          description="This will allow guests to have view access to all the project work items."
        >
          <div className="flex items-center justify-end">
            <Switch
              size="sm"
              checked={project.guest_view_all_features}
              onCheckedChange={toggleGuestViewAllIssues}
              disabled={!isAdmin}
              aria-label="Guest access"
            />
          </div>
        </DefaultSettingItem>
      )}
    </div>
  );
});
