/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, useRef } from "react";
import { observer } from "mobx-react";

// nerve imports
import { EIconSize } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { StateGroupIcon } from "@nerve/propel/icons";
import { AddOutline, ChevronDownOutline } from "@makeplane/propel/icons";
import type { State, StateGroup } from "@nerve/api-client";
import { cn } from "@nerve/utils";
// components
import { StateList, StateCreate } from "@/components/project-states";

type TGroupItem = {
  projectId: string;
  groupKey: StateGroup;
  groupsExpanded: Partial<StateGroup>[];
  groupedStates: Record<string, State[]>;
  states: State[];
  isEditable: boolean;
  handleGroupCollapse: (groupKey: StateGroup) => void;
  handleExpand: (groupKey: StateGroup) => void;
};

export const GroupItem = observer(function GroupItem(props: TGroupItem) {
  const { projectId, groupKey, groupedStates, states, groupsExpanded, isEditable, handleExpand, handleGroupCollapse } =
    props;
  // refs
  const dropElementRef = useRef<HTMLDivElement | null>(null);
  // nerve hooks
  const { t } = useTranslation();
  // state
  const [createState, setCreateState] = useState(false);
  // derived values
  const currentStateExpanded = groupsExpanded.includes(groupKey);
  const shouldShowEmptyState = states.length === 0 && currentStateExpanded && !createState;

  return (
    <div className="space-y-1 rounded-sm border border-subtle bg-surface-2 p-2 transition-all" ref={dropElementRef}>
      <div className="flex items-center justify-between gap-2">
        <button
          type="button"
          className="flex w-full cursor-pointer items-center py-1"
          aria-expanded={currentStateExpanded}
          onClick={() => (!currentStateExpanded ? handleExpand(groupKey) : handleGroupCollapse(groupKey))}
        >
          <span
            className={cn(
              "flex h-5 w-5 flex-shrink-0 items-center justify-center overflow-hidden rounded-sm transition-all",
              {
                "rotate-0": currentStateExpanded,
                "-rotate-90": !currentStateExpanded,
              }
            )}
          >
            <ChevronDownOutline className="h-4 w-4" />
          </span>
          <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center overflow-hidden rounded-sm">
            <StateGroupIcon stateGroup={groupKey} size={EIconSize.XL} />
          </span>
          <span className="px-1 text-14 font-medium text-secondary capitalize">{groupKey}</span>
        </button>
        <button
          type="button"
          aria-label={t("common.add")}
          className={cn(
            "flex h-6 w-6 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm text-accent-primary/80 transition-colors hover:bg-layer-1 hover:text-accent-primary",
            (!isEditable || createState) && "cursor-not-allowed text-placeholder hover:text-placeholder"
          )}
          onClick={() => {
            if (!createState) {
              handleExpand(groupKey);
              setCreateState(true);
            }
          }}
          disabled={!isEditable || createState}
        >
          <AddOutline className="h-4 w-4" />
        </button>
      </div>

      {shouldShowEmptyState && (
        <div className="flex h-full flex-col items-center justify-center py-4 text-13 text-tertiary">
          <div>{t("project_settings.states.empty_state.title", { groupKey })}</div>
          {isEditable && <div>{t("project_settings.states.empty_state.description")}</div>}
        </div>
      )}

      {currentStateExpanded && (
        <div id="group-droppable-container">
          <StateList groupKey={groupKey} groupedStates={groupedStates} states={states} disabled={!isEditable} />
        </div>
      )}

      {isEditable && createState && (
        <div className="">
          <StateCreate projectId={projectId} groupKey={groupKey} handleClose={() => setCreateState(false)} />
        </div>
      )}
    </div>
  );
});
