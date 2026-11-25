/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { Command } from "cmdk";
import type { Cycle, Issue, Module, Project, View } from "@nerve/api-client";
import { CyclesOutline, ModuleOutline, ProjectsOutline, ViewsOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { useNavigate } from "react-router";
import { PowerKModalCommandItem } from "./command-item";

type Props = {
  workspaceSlug: string;
  issues: { issue: Issue; project: Project }[];
  projects: Project[];
  cycles: { cycle: Cycle; project: Project }[];
  modules: { module: Module; project: Project }[];
  views: { view: View; project: Project }[];
  closePalette: () => void;
};

/** A cycle's, a module's or a view's row: its project's identifier, small and dimmed, then its name. */
function ProjectItemLabel({ project, name }: { project: Project; name: string }) {
  return (
    <p>
      <span className="text-11 text-tertiary">{project.identifier}</span> {name}
    </p>
  );
}

/**
 * The sequence of issues is the native service's relevance order, without another client-side rank or filter; the
 * cycles and modules (P7c spec 2.4) and the views (P6a spec 2.8) of the search's project follow, each going to its
 * page, in nerve's order.
 */
export function PowerKModalSearchResults({
  workspaceSlug,
  issues,
  projects,
  cycles,
  modules,
  views,
  closePalette,
}: Props) {
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
      {cycles.length > 0 && (
        <Command.Group heading={t("common.cycles")}>
          {cycles.map(({ cycle, project }) => (
            <PowerKModalCommandItem
              key={cycle.id}
              value={`cycle-${cycle.id}`}
              icon={CyclesOutline}
              label={<ProjectItemLabel project={project} name={cycle.name} />}
              onSelect={() => open(`/${workspaceSlug}/projects/${project.id}/cycles/${cycle.id}`)}
            />
          ))}
        </Command.Group>
      )}
      {modules.length > 0 && (
        <Command.Group heading={t("common.modules")}>
          {modules.map(({ module, project }) => (
            <PowerKModalCommandItem
              key={module.id}
              value={`module-${module.id}`}
              icon={ModuleOutline}
              label={<ProjectItemLabel project={project} name={module.name} />}
              onSelect={() => open(`/${workspaceSlug}/projects/${project.id}/modules/${module.id}`)}
            />
          ))}
        </Command.Group>
      )}
      {views.length > 0 && (
        <Command.Group heading={t("views")}>
          {views.map(({ view, project }) => (
            <PowerKModalCommandItem
              key={view.id}
              value={`view-${view.id}`}
              icon={ViewsOutline}
              label={<ProjectItemLabel project={project} name={view.name} />}
              onSelect={() => open(`/${workspaceSlug}/projects/${project.id}/views/${view.id}`)}
            />
          ))}
        </Command.Group>
      )}
    </>
  );
}
