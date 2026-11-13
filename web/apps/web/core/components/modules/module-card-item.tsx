/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";
import { Link } from "react-router";
import { InfoOutline, WorkItemsOutline } from "@makeplane/propel/icons";
import { LinearProgress } from "@makeplane/propel/components/linear-progress";
import { Tooltip } from "@makeplane/propel/components/tooltip";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Card } from "@nerve/ui";
import { progressPercent } from "@nerve/utils";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { useUserPermissions } from "@/hooks/store/user";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { ModuleLeadAvatar } from "./lead-avatar";
import { ModuleDates } from "./module-dates";
import { usePeekModule } from "./module-peek-overview";
import { ModuleStatusDropdown } from "./module-status-dropdown";
import { ModuleQuickActions } from "./quick-actions";

type Props = {
  moduleId: string;
  workspaceSlug: string;
  projectId: string;
};

/**
 * A module's card in its project's list laid out as cards (P6a spec 2.6, R5, R8, R11, R13), its list's row as the
 * store gives it: its name, the link to its page, which the whole card is (the controls over it are buttons of their
 * own, never a clickable element that is neither); its status, which the caller changes there while he writes the
 * project's modules (canWriteModules) and it is not archived; "more details", which previews it; its progress, the
 * completed of its work items not cancelled, as a count and a bar of the same reading (progressPercent); its lead; its
 * days, changed as its status is; its quick actions. No star to favorite it: favorites are M7's.
 */
export const ModuleCardItem = observer(function ModuleCardItem(props: Props) {
  const { moduleId, workspaceSlug, projectId } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // hooks
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const [peekModule, peek] = usePeekModule();
  // store hooks
  const { getModuleRow } = useModules();
  const { canWriteModules } = useUserPermissions();
  const row = getModuleRow(moduleId);

  if (!row) return null;

  const editable = canWriteModules(workspaceSlug, projectId) && row.archived_at === null;
  const dated = row.start_date !== null || row.target_date !== null;

  return (
    <Card ref={parentRef} className="relative" data-prevent-progress>
      <div className="flex items-center justify-between gap-2">
        <Link
          to={`/${workspaceSlug}/projects/${projectId}/modules/${row.id}`}
          className="truncate after:absolute after:inset-0"
        >
          <Tooltip label={row.name} layout="stacked" disabled={isMobile}>
            <span className="truncate text-14 font-medium">{row.name}</span>
          </Tooltip>
        </Link>
        <div className="relative z-[1] flex items-center gap-2">
          <ModuleStatusDropdown moduleId={row.id} editable={editable} />
          <button
            type="button"
            aria-label={t("project_module.more_details")}
            onClick={() => peek(peekModule === row.id ? null : row.id)}
          >
            <InfoOutline className="h-4 w-4 text-placeholder" aria-hidden="true" />
          </button>
        </div>
      </div>
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5 text-secondary">
            <WorkItemsOutline className="h-4 w-4 text-tertiary" aria-hidden="true" />
            <span className="text-11 text-tertiary">
              {t("project_module.work_items_done", {
                completed: row.completed_issues,
                total: row.total_issues - row.cancelled_issues,
              })}
            </span>
          </div>
          <span className="relative z-[1]">
            <ModuleLeadAvatar leadId={row.lead_id} />
          </span>
        </div>
        <LinearProgress
          value={progressPercent(row)}
          size="md"
          variant="brand"
          showValue={false}
          aria-label={t("common.progress")}
        />
        <div className="relative z-[1] flex items-center justify-between gap-2 py-0.5">
          <ModuleDates
            module={row}
            editable={editable}
            buttonContainerClassName="h-6 w-full flex items-center gap-1.5 text-tertiary border-[0.5px] border-strong rounded-sm text-11"
            buttonVariant="transparent-with-text"
            className="h-7"
            hideIcon={{ from: dated, to: dated }}
          />
          <ModuleQuickActions
            parentRef={parentRef}
            moduleId={row.id}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
          />
        </div>
      </div>
    </Card>
  );
});
