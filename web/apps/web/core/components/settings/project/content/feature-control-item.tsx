/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { Switch } from "@makeplane/propel/components/switch";
// components
import { useFeatureToggle } from "@/components/project/settings/use-feature-toggle";
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
import type { ProjectToggleField } from "@/store/project/project.store";

type Props = {
  description?: React.ReactNode;
  projectId: string;
  featureProperty: ProjectToggleField;
  title: React.ReactNode;
  value: boolean;
  workspaceSlug: string;
};

export const ProjectSettingsFeatureControlItem = observer(function ProjectSettingsFeatureControlItem(props: Props) {
  const { description, featureProperty, projectId, title, value, workspaceSlug } = props;
  // the feature turns the other way from nerve's last answer, in the change's turn (v0 design 7.7)
  const toggleFeature = useFeatureToggle(workspaceSlug, projectId);

  return (
    <SettingsBoxedControlItem
      title={title}
      description={description}
      control={
        <Switch
          size="sm"
          checked={value}
          onCheckedChange={() => toggleFeature(featureProperty)}
          aria-label={typeof title === "string" ? title : "Toggle project feature"}
        />
      }
    />
  );
});
