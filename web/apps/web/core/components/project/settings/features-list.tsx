/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
import { SettingsHeading } from "@/components/settings/heading";
// hooks
import { useProject } from "@/hooks/store/use-project";
import type { ProjectToggleField } from "@/store/project/project.store";
// local imports
import { ProjectFeatureToggle } from "./helper";
import { useFeatureToggle } from "./use-feature-toggle";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

const PROJECT_FEATURES_LIST: Record<
  string,
  { i18n_label: string; i18n_description: string; property: ProjectToggleField }
> = {
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
    property: "intake_view",
  },
};

export const ProjectFeaturesList = observer(function ProjectFeaturesList(props: Props) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const { t } = useTranslation();
  const { getProjectById } = useProject();
  // the feature turns the other way from nerve's last answer, in the change's turn (v0 design 7.7)
  const toggleFeature = useFeatureToggle(workspaceSlug, projectId);
  // derived values
  const currentProjectDetails = getProjectById(projectId);

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
                  value={Boolean(currentProjectDetails?.[featureItem.property])}
                  handleSubmit={toggleFeature}
                />
              }
            />
          </div>
        ))}
      </div>
    </div>
  );
});
