/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate, useParams } from "react-router";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";

/**
 * Why the project's pages do not show (M3 design 3.19, 7.6): the caller sees the project and is no member of it,
 * which he may join; it is archived; or it is not found to him (it does not exist, is deleted, or he does not see it).
 */
type TProps =
  | { kind: "not-member"; handleJoinProject: () => void; isJoinButtonDisabled: boolean }
  | { kind: "archived" }
  | { kind: "not-found" };

export const ProjectAccessRestriction = observer(function ProjectAccessRestriction(props: TProps) {
  // router
  const navigate = useNavigate();
  const { workspaceSlug } = useParams();
  // nerve hooks
  const { t } = useTranslation();

  // the caller sees the project and is no member of it: he may join it
  if (props.kind === "not-member")
    return (
      <div className="grid h-full w-full place-items-center bg-surface-1">
        <EmptyStateDetailed
          title={t("project_empty_state.no_access.title")}
          description={t("project_empty_state.no_access.join_description")}
          assetKey="no-access"
          assetClassName="size-40"
          actions={[
            {
              label: props.isJoinButtonDisabled
                ? t("project_empty_state.no_access.cta_loading")
                : t("project_empty_state.no_access.cta_primary"),
              onClick: props.handleJoinProject,
              disabled: props.isJoinButtonDisabled,
            },
          ]}
        />
      </div>
    );

  // the project is archived: its pages show again once restored, from the workspace's archived projects
  if (props.kind === "archived")
    return (
      <div className="grid h-full w-full place-items-center bg-surface-1">
        <EmptyStateDetailed
          title={t("project_empty_state.archived.title")}
          description={t("project_empty_state.archived.description")}
          assetKey="project"
          assetClassName="size-40"
          actions={[
            {
              label: t("project_empty_state.archived.cta_primary"),
              onClick: () => void navigate(`/${workspaceSlug}/projects/archives`),
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
