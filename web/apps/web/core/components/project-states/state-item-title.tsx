/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { SetStateAction } from "react";
import { observer } from "mobx-react";
import { DragDropOutline, EditOutline } from "@makeplane/propel/icons";
import { EIconSize } from "@nerve/constants";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { StateGroupIcon } from "@nerve/propel/icons";
import type { State } from "@nerve/api-client";
// local imports
import { useProjectState } from "@/hooks/store/use-project-state";
import { StateDelete, StateMarksAsDefault } from "./options";

type TStateItemTitleProps = {
  stateCount: number;
  state: State;
  setUpdateStateModal: (value: SetStateAction<boolean>) => void;
  /** Whether the state is shown alone, with none of the controls that change it. */
  disabled: boolean;
};

export const StateItemTitle = observer(function StateItemTitle(props: TStateItemTitleProps) {
  const { stateCount, setUpdateStateModal, disabled, state } = props;
  const { t } = useTranslation();
  // store hooks
  const { getStatePercentageInGroup } = useProjectState();
  // derived values
  const statePercentage = getStatePercentageInGroup(state.id);
  const percentage = statePercentage ? statePercentage / 100 : undefined;

  return (
    <div className="flex w-full items-center justify-between gap-2">
      <div className="flex items-center gap-1 px-1">
        {/* draggable indicator */}
        {!disabled && stateCount != 1 && (
          <div className="absolute -left-1.5 hidden h-3 w-3 flex-shrink-0 cursor-pointer items-center justify-center rounded-xs bg-surface-2 text-secondary transition-colors group-hover:flex hover:text-primary">
            <DragDropOutline className="h-3 w-3" />
          </div>
        )}
        {/* state icon */}
        <div className="flex-shrink-0">
          <StateGroupIcon stateGroup={state.group} color={state.color} size={EIconSize.XL} percentage={percentage} />
        </div>
        {/* state title and description */}
        <div className="min-h-5 px-2 text-13">
          <h6 className="text-13 font-medium">{state.name}</h6>
          <p className="text-11 text-secondary">{state.description}</p>
        </div>
      </div>
      {!disabled && (
        <div className="flex items-center gap-2 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100">
          {/* state mark as default option */}
          <div className="flex-shrink-0 text-11 transition-all">
            <StateMarksAsDefault stateId={state.id} stateName={state.name} isDefault={state.default} />
          </div>
          {/* state edit options */}
          <div className="flex items-center gap-1 transition-all">
            <button
              type="button"
              aria-label={t("common.edit_item", { name: state.name })}
              className="flex h-5 w-5 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm text-secondary transition-colors hover:bg-layer-1 hover:text-primary"
              onClick={() => setUpdateStateModal(true)}
            >
              <EditOutline className="h-3 w-3" />
            </button>
            <StateDelete totalStates={stateCount} state={state} />
          </div>
        </div>
      )}
    </div>
  );
});
