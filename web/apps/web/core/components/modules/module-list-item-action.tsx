/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type React from "react";
import { observer } from "mobx-react";
import { InfoOutline } from "@makeplane/propel/icons";
// nerve imports
import type { ModuleListItem } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { cn } from "@nerve/utils";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { ModuleLeadAvatar } from "./lead-avatar";
import { ModuleDates } from "./module-dates";
import { ModuleStatusDropdown } from "./module-status-dropdown";
import { ModuleQuickActions } from "./quick-actions";

type Props = {
  workspaceSlug: string;
  projectId: string;
  row: ModuleListItem;
  parentRef: React.RefObject<HTMLDivElement | null>;
  /** Previews the module, or closes its preview. */
  onPeek: () => void;
};

/**
 * The actions and the facts of a module's row (P6a spec 2.6, R5, R11, R13): "more details", which previews it; its
 * days, calendar days, and its status, which the caller changes there while he writes the project's modules
 * (canWriteModules) and it is not archived, each nerve's change to make (ModuleDates, ModuleStatusDropdown:
 * useModuleChange); its lead; its quick actions, one menu for the row whatever the screen (its right-click menu is the
 * row's). No star to favorite it: favorites are M7's.
 */
export const ModuleListItemAction = observer(function ModuleListItemAction(props: Props) {
  const { workspaceSlug, projectId, row, parentRef, onPeek } = props;
  // hooks
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const { canWriteModules } = useUserPermissions();

  const editable = canWriteModules(workspaceSlug, projectId) && row.archived_at === null;
  const dated = row.start_date !== null || row.target_date !== null;

  const handlePeek = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    e.stopPropagation();
    onPeek();
  };

  return (
    <>
      <button
        type="button"
        onClick={handlePeek}
        aria-label={t("project_module.more_details")}
        className={cn("z-[1] flex-shrink-0", isMobile ? "flex" : "hidden group-hover:flex")}
      >
        <InfoOutline className="h-4 w-4 text-placeholder" aria-hidden="true" />
      </button>
      <ModuleDates
        module={row}
        editable={editable}
        buttonContainerClassName="h-6 w-full flex items-center gap-1.5 text-tertiary border-[0.5px] border-strong rounded-sm text-11"
        buttonVariant="transparent-with-text"
        className="h-7"
        mergeDates
        hideIcon={{ from: dated, to: dated }}
      />
      <ModuleStatusDropdown moduleId={row.id} editable={editable} />
      <ModuleLeadAvatar leadId={row.lead_id} />
      <ModuleQuickActions parentRef={parentRef} moduleId={row.id} projectId={projectId} workspaceSlug={workspaceSlug} />
    </>
  );
});
