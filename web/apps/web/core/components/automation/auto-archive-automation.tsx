/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { RestoreOutline } from "@makeplane/propel/icons";
// nerve imports
import { PROJECT_AUTOMATION_MONTHS, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { ProjectUpdate } from "@nerve/api-client";
import { Switch } from "@makeplane/propel/components/switch";
import { CustomSelect, Loader } from "@nerve/ui";
// component
import { SelectMonthModal } from "@/components/automation";
import { SettingsControlItem } from "@/components/settings/control-item";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

const initialValues: Pick<ProjectUpdate, "archive_in"> = { archive_in: 1 };

/**
 * The auto-archiving of the current project's closed work items: its switch turns it on, after a month, or off, from
 * the project as nerve last answered it, in the change's turn (ProjectStore.toggleAutoArchive; v0 design 7.7), not
 * from what the switch shows; its select and the custom range send the months picked. The page follows each only in
 * the session it was sent in (M3 design 7.1), a refusal with nerve's reason in a toast.
 */
export const AutoArchiveAutomation = observer(function AutoArchiveAutomation() {
  // router
  const { workspaceSlug, projectId } = useParams();
  // states
  const [monthModal, setmonthModal] = useState(false);
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();

  const { getProjectById, updateProject, toggleAutoArchive } = useProject();
  const project = projectId ? getProjectById(projectId) : undefined;

  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, project?.id);

  /**
   * Sends a change of the current project's auto-archiving, and follows it: done, once nerve has made it; a refusal,
   * with nerve's reason in a toast. Settles once the page has followed it.
   */
  const send = async (change: (id: string) => Promise<unknown>, done?: () => void) => {
    if (!project) return;

    await followInSession(() => change(project.id), { done, failed: toastRefusal });
  };
  const handleChange = (formData: Pick<ProjectUpdate, "archive_in">, done?: () => void) =>
    send((id) => updateProject(id, formData), done);
  const handleToggle = () => send((id) => toggleAutoArchive(id));

  const autoArchiveStatus = useMemo(() => {
    if (project?.archive_in === undefined) return false;
    return project.archive_in !== 0;
  }, [project]);

  return (
    <>
      <SelectMonthModal
        initialValues={initialValues}
        isOpen={monthModal}
        handleClose={() => setmonthModal(false)}
        handleChange={handleChange}
      />
      <div className="flex flex-col gap-4 border-b border-subtle py-2">
        <div className="flex items-center gap-3">
          <div className="grid size-10 shrink-0 place-items-center rounded-sm bg-layer-2">
            <RestoreOutline className="size-4 shrink-0 text-primary" />
          </div>
          <SettingsControlItem
            title={t("project_settings.automations.auto-archive.title")}
            description={t("project_settings.automations.auto-archive.description")}
            control={
              <Switch
                size="sm"
                checked={autoArchiveStatus}
                onCheckedChange={handleToggle}
                disabled={!isAdmin}
                aria-label={t("project_settings.automations.auto-archive.title")}
              />
            }
          />
        </div>
        {project ? (
          autoArchiveStatus && (
            <div className="ml-13">
              <div className="flex w-full items-center justify-between gap-2 rounded-sm border border-subtle bg-surface-2 px-5 py-4">
                <div className="w-1/2 text-13 font-medium">
                  {t("project_settings.automations.auto-archive.duration")}
                </div>
                <div className="w-1/2">
                  <CustomSelect
                    value={project?.archive_in}
                    label={`${project?.archive_in} ${project?.archive_in === 1 ? "month" : "months"}`}
                    onChange={(val: number) => void handleChange({ archive_in: val })}
                    input
                    disabled={!isAdmin}
                  >
                    <>
                      {PROJECT_AUTOMATION_MONTHS.map((month) => (
                        <CustomSelect.Option key={month.i18n_label} value={month.value}>
                          <span className="text-13">{t(month.i18n_label, { months: month.value })}</span>
                        </CustomSelect.Option>
                      ))}

                      <button
                        type="button"
                        className="flex w-full items-center rounded-sm px-1 py-1.5 text-13 text-secondary select-none hover:bg-layer-1"
                        onClick={() => setmonthModal(true)}
                      >
                        {t("common.customize_time_range")}
                      </button>
                    </>
                  </CustomSelect>
                </div>
              </div>
            </div>
          )
        ) : (
          <Loader className="ml-13">
            <Loader.Item height="50px" />
          </Loader>
        )}
      </div>
    </>
  );
});
