/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { Command } from "cmdk";
import type { Issue, Project } from "@nerve/api-client";
import { ProjectsOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { useNavigate } from "react-router";
import { PowerKModalCommandItem } from "./command-item";

type Props = {
  workspaceSlug: string;
  issues: { issue: Issue; project: Project }[];
  projects: Project[];
  closePalette: () => void;
};

/** The sequence of issues is the native service's relevance order, without another client-side rank or filter. */
export function PowerKModalSearchResults({ workspaceSlug, issues, projects, closePalette }: Props) {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const open = (path: string) => {
    closePalette();
    navigate(path);
  };
  return (
    <>
      {issues.length > 0 && (
        <Command.Group heading={t("work_items")}>
          {issues.map(({ issue, project }) => (
            <PowerKModalCommandItem
              key={issue.id}
              value={`issue-${issue.id}`}
              label={
                <div className="flex gap-2">
                  <span className="text-tertiary">
                    {project.identifier}-{issue.sequence_id}
                  </span>
                  {issue.name}
                </div>
              }
              onSelect={() => open(`/${workspaceSlug}/browse/${project.identifier}-${issue.sequence_id}`)}
            />
          ))}
        </Command.Group>
      )}
      {projects.length > 0 && (
        <Command.Group heading={t("common.projects")}>
          {projects.map((project) => (
            <PowerKModalCommandItem
              key={project.id}
              value={`project-${project.id}`}
              icon={ProjectsOutline}
              label={project.name}
              onSelect={() => open(`/${workspaceSlug}/projects/${project.id}/issues`)}
            />
          ))}
        </Command.Group>
      )}
    </>
  );
}
