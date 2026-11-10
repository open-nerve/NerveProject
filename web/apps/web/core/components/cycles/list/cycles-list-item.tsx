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
import { cycleCounts, progressPercent } from "@nerve/utils";
// components
import { ListItem } from "@/components/core/list";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { usePeekCycle } from "../cycle-peek-overview";
import { CycleListItemAction } from "./cycle-list-item-action";

type TCyclesListItem = {
  cycleId: string;
  workspaceSlug: string;
  projectId: string;
  className?: string;
};

/**
 * A cycle's row in its project's list or archive (P5a spec 2.5), its list's row as the store gives it: its name, a link
 * to its page (an archived cycle's opens its preview instead), and its progress, the completed of its work items not
 * cancelled (progressPercent), of the counts cycleCounts chooses (those it was frozen with once it has ended, R7), with a
 * tick once all are done; then its actions (CycleListItemAction).
 */
export const CyclesListItem = observer(function CyclesListItem(props: TCyclesListItem) {
  const { cycleId, workspaceSlug, projectId, className = "" } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // hooks
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const [peekCycle, peek] = usePeekCycle();
  // store hooks
  const { getCycleRow } = useCycles();
  const row = getCycleRow(cycleId);

  if (!row) return null;

  const progress = progressPercent(cycleCounts(row.status, row, row.progress_snapshot));
  const togglePeek = () => peek(peekCycle === row.id ? null : row.id);

  return (
    <ListItem
      title={row.name}
      itemLink={`/${workspaceSlug}/projects/${projectId}/cycles/${row.id}`}
      onItemClick={row.archived_at === null ? undefined : togglePeek}
      className={className}
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
              <TickOutline className="h-3 w-3" />
            ) : (
              <span className="text-9 text-primary">{`${progress}%`}</span>
            )}
          </div>
        </div>
      }
      actionableItems={
        <CycleListItemAction
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          row={row}
          parentRef={parentRef}
          isPeekOpen={peekCycle !== null}
          onPeek={togglePeek}
        />
      }
      isMobile={isMobile}
      parentRef={parentRef}
      isSidebarOpen={peekCycle !== null}
    />
  );
});
