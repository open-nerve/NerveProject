/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { setPromiseToast } from "@nerve/propel/toast";
import { Switch } from "@makeplane/propel/components/switch";
// components
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
// hooks
import { useProject } from "@/hooks/store/use-project";
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
  // store hooks
  const { toggleProject } = useProject();

  // the feature turns the other way from nerve's last answer, in the change's turn (v0 design 7.7)
  const handleSubmit = () => {
    if (!workspaceSlug || !projectId) return;

    setPromiseToast(toggleProject(projectId, featureProperty), {
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
  };

  return (
    <SettingsBoxedControlItem
      title={title}
      description={description}
      control={
        <Switch
          size="sm"
          checked={value}
          onCheckedChange={handleSubmit}
          aria-label={typeof title === "string" ? title : "Toggle project feature"}
        />
      }
    />
  );
});
