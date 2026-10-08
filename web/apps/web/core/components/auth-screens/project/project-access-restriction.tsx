/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";

type TProps = {
  /** Whether the caller sees the project and is no member of it (M3 design 3.19): he may join it. */
  canJoin: boolean;
  handleJoinProject: () => void;
  isJoinButtonDisabled: boolean;
};

export const ProjectAccessRestriction = observer(function ProjectAccessRestriction(props: TProps) {
  const { canJoin, handleJoinProject, isJoinButtonDisabled } = props;
  // nerve hooks
  const { t } = useTranslation();

  // the caller sees the project and is no member of it: he may join it
  if (canJoin)
    return (
      <div className="grid h-full w-full place-items-center bg-surface-1">
        <EmptyStateDetailed
          title={t("project_empty_state.no_access.title")}
          description={t("project_empty_state.no_access.join_description")}
          assetKey="no-access"
          assetClassName="size-40"
          actions={[
            {
              label: isJoinButtonDisabled
                ? t("project_empty_state.no_access.cta_loading")
                : t("project_empty_state.no_access.cta_primary"),
              onClick: handleJoinProject,
              disabled: isJoinButtonDisabled,
            },
          ]}
        />
      </div>
    );

  // the project is not found to him: it does not exist, is deleted, or he does not see it
  return (
    <div className="grid h-full w-full place-items-center bg-surface-1">
      <EmptyStateDetailed
        title={t("project_empty_state.invalid_project.title")}
        description={t("project_empty_state.invalid_project.description")}
        assetKey="project"
        assetClassName="size-40"
      />
    </div>
  );
});
