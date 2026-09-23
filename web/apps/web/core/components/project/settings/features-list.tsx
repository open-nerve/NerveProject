/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { setPromiseToast } from "@plane/propel/toast";
import type { IProject } from "@plane/types";
// components
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
import { SettingsHeading } from "@/components/settings/heading";
// hooks
import { useProject } from "@/hooks/store/use-project";
// local imports
import { ProjectFeatureToggle } from "./helper";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

const PROJECT_FEATURES_LIST = {
  cycles: {
    i18n_label: "cycles",
    i18n_description: "cycles_description",
    property: "cycle_view",
  },
  modules: {
    i18n_label: "modules",
    i18n_description: "modules_description",
    property: "module_view",
  },
  views: {
    i18n_label: "views",
    i18n_description: "views_description",
    property: "issue_views_view",
  },
  inbox: {
    i18n_label: "intake",
    i18n_description: "intake_description",
    property: "inbox_view",
  },
};

export const ProjectFeaturesList = observer(function ProjectFeaturesList(props: Props) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const { t } = useTranslation();
  const { getProjectById, updateProject } = useProject();
  // derived values
  const currentProjectDetails = getProjectById(projectId);

  const handleSubmit = (featureProperty: string) => {
    if (!workspaceSlug || !projectId || !currentProjectDetails) return;

    // making the request to update the project feature
    const settingsPayload = {
      [featureProperty]: !currentProjectDetails?.[featureProperty as keyof IProject],
    };
    const updateProjectPromise = updateProject(workspaceSlug, projectId, settingsPayload);

    setPromiseToast(updateProjectPromise, {
      loading: "Updating project feature...",
      success: {
        title: "Success!",
        message: () => "Project feature updated successfully.",
      },
      error: {
        title: "Error!",
        message: () => "Something went wrong while updating project feature. Please try again.",
      },
    });
    void updateProjectPromise.then(() => {
      return undefined;
    });
  };

  return (
    <div>
      <SettingsHeading title={t("projects_and_issues")} description={t("projects_and_issues_description")} />
      <div className="mt-6 flex flex-col gap-y-4">
        {Object.entries(PROJECT_FEATURES_LIST).map(([featureItemKey, featureItem]) => (
          <div key={featureItemKey}>
            <SettingsBoxedControlItem
              title={t(featureItem.i18n_label)}
              description={t(featureItem.i18n_description)}
              control={
                <ProjectFeatureToggle
                  featureItem={featureItem}
                  value={Boolean(currentProjectDetails?.[featureItem.property as keyof IProject])}
                  handleSubmit={handleSubmit}
                />
              }
            />
          </div>
        ))}
      </div>
    </div>
  );
});
