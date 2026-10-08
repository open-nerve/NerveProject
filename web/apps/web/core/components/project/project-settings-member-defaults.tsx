/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useEffect } from "react";
import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import useSWR from "swr";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { Project } from "@nerve/api-client";
import { Switch } from "@makeplane/propel/components/switch";
import { Loader } from "@nerve/ui";
// constants
import { PROJECT_DETAILS } from "@nerve/constants";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { MemberSelect } from "./member-select";

type TMemberDefaults = Pick<Project, "project_lead_id" | "default_assignee_id">;

const defaultValues: TMemberDefaults = {
  project_lead_id: null,
  default_assignee_id: null,
};

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

export const ProjectSettingsMemberDefaults = observer(function ProjectSettingsMemberDefaults(
  props: TProjectSettingsMemberDefaultsProps
) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { allowPermissions } = useUserPermissions();

  const { currentProjectDetails, fetchProject, updateProject } = useProject();
  // derived values
  const isAdmin = allowPermissions(
    [EUserPermissions.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    currentProjectDetails?.id
  );
  // form info
  const { reset, control } = useForm<TMemberDefaults>({ defaultValues });
  // fetching user members
  useSWR(
    workspaceSlug && projectId ? PROJECT_DETAILS(workspaceSlug, projectId) : null,
    workspaceSlug && projectId ? () => fetchProject(projectId) : null
  );

  useEffect(() => {
    if (!currentProjectDetails) return;

    reset({
      project_lead_id: currentProjectDetails.project_lead_id,
      default_assignee_id: currentProjectDetails.default_assignee_id,
    });
  }, [currentProjectDetails, reset]);

  const submitChanges = async (formData: Partial<TMemberDefaults>) => {
    if (!workspaceSlug || !projectId) return;

    reset({
      project_lead_id: currentProjectDetails?.project_lead_id ?? null,
      default_assignee_id: currentProjectDetails?.default_assignee_id ?? null,
      ...formData,
    });

    try {
      await updateProject(projectId, {
        default_assignee_id:
          formData.default_assignee_id === "none"
            ? null
            : (formData.default_assignee_id ?? currentProjectDetails?.default_assignee_id),
        project_lead_id:
          formData.project_lead_id === "none"
            ? null
            : (formData.project_lead_id ?? currentProjectDetails?.project_lead_id),
      });
      setToast({
        title: `${t("success")}!`,
        type: TOAST_TYPE.SUCCESS,
        message: t("project_settings.general.toast.success"),
      });
    } catch (err) {
      console.error(err);
    }
  };

  const toggleGuestViewAllIssues = async (value: boolean) => {
    if (!workspaceSlug || !projectId) return;

    try {
      await updateProject(projectId, {
        guest_view_all_features: value,
      });
      setToast({
        title: `${t("success")}!`,
        type: TOAST_TYPE.SUCCESS,
        message: t("project_settings.general.toast.success"),
      });
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <div className="my-6 flex flex-col gap-y-6">
      <DefaultSettingItem title="Project Lead" description="Select the project lead for the project.">
        {currentProjectDetails ? (
          <Controller
            control={control}
            name="project_lead_id"
            render={({ field: { value } }) => (
              <MemberSelect
                value={value}
                onChange={(val: string) => {
                  submitChanges({ project_lead_id: val });
                }}
                isDisabled={!isAdmin}
              />
            )}
          />
        ) : (
          <Loader className="h-9 w-full">
            <Loader.Item width="100%" height="100%" />
          </Loader>
        )}
      </DefaultSettingItem>
      <DefaultSettingItem title="Default Assignee" description="Select the default assignee for the project.">
        {currentProjectDetails ? (
          <Controller
            control={control}
            name="default_assignee_id"
            render={({ field: { value } }) => (
              <MemberSelect
                value={value}
                onChange={(val: string) => {
                  submitChanges({ default_assignee_id: val });
                }}
                isDisabled={!isAdmin}
              />
            )}
          />
        ) : (
          <Loader className="h-9 w-full">
            <Loader.Item width="100%" height="100%" />
          </Loader>
        )}
      </DefaultSettingItem>
      {currentProjectDetails && (
        <DefaultSettingItem
          title="Guest access"
          description="This will allow guests to have view access to all the project work items."
        >
          <div className="flex items-center justify-end">
            <Switch
              size="sm"
              checked={!!currentProjectDetails?.guest_view_all_features}
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
