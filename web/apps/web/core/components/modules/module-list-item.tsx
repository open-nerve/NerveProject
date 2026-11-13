/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";
import { CircularProgress } from "@makeplane/propel/components/circular-progress";
import { TickOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { progressPercent } from "@nerve/utils";
// components
import { ListItem } from "@/components/core/list";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { ModuleListItemAction } from "./module-list-item-action";
import { usePeekModule } from "./module-peek-overview";

type Props = {
  moduleId: string;
  workspaceSlug: string;
  projectId: string;
};

/**
 * A module's row in its project's list or archive (P6a spec 2.6, R8), its list's row as the store gives it: its name,
 * a link to its page (an archived module's opens its preview instead, as the old row did); its progress, the completed of
 * its work items not cancelled (progressPercent, the one reading of it), a tick once all are done, and a "!" for a
 * module completed short of that; then its actions and its facts (ModuleListItemAction).
 */
export const ModuleListItem = observer(function ModuleListItem(props: Props) {
  const { moduleId, workspaceSlug, projectId } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // hooks
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const [peekModule, peek] = usePeekModule();
  // store hooks
  const { getModuleRow } = useModules();
  const row = getModuleRow(moduleId);

  if (!row) return null;

  const progress = progressPercent(row);
  const togglePeek = () => peek(peekModule === row.id ? null : row.id);

  return (
    <ListItem
      title={row.name}
      itemLink={`/${workspaceSlug}/projects/${projectId}/modules/${row.id}`}
      onItemClick={row.archived_at === null ? undefined : togglePeek}
      prependTitleElement={
        <div className="relative size-[30px]">
          <div className="absolute inset-0 flex scale-150 items-center justify-center">
            <CircularProgress
              value={progress}
              size="md"
              variant={progress === 100 ? "success" : "brand"}
              aria-label={t("common.progress")}
            />
          </div>
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center" aria-hidden="true">
            {progress === 100 ? (
              <TickOutline className="h-3 w-3 stroke-[2] text-accent-primary" />
            ) : row.status === "completed" ? (
              <span className="text-13 text-accent-primary">!</span>
            ) : (
              <span className="text-9 text-tertiary">{`${progress}%`}</span>
            )}
          </div>
        </div>
      }
      actionableItems={
        <ModuleListItemAction
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          row={row}
          parentRef={parentRef}
          onPeek={togglePeek}
        />
      }
      isMobile={isMobile}
      parentRef={parentRef}
      isSidebarOpen={peekModule !== null}
    />
  );
});
