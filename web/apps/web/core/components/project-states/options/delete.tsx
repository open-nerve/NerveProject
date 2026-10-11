/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { CloseOutline, LoadingOutline } from "@makeplane/propel/icons";
// nerve imports
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { State } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { AlertModalCore } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { useStateChanges } from "../use-state-changes";

type TStateDelete = {
  totalStates: number;
  state: State;
};

export const StateDelete = observer(function StateDelete(props: TStateDelete) {
  const { totalStates, state } = props;
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const { remove } = useStateChanges();
  // states
  const [isDeleteModal, setIsDeleteModal] = useState(false);
  const [isDelete, setIsDelete] = useState(false);
  // derived values
  const isDeleteDisabled = state.default || totalStates === 1;

  // Busy, and the dialog not to be dismissed, until the page has followed nerve's answer: it closes once nerve has
  // deleted the state, and stays, its buttons enabled again, when nerve refuses, the reason in a toast.
  const handleDeleteState = async () => {
    if (isDeleteDisabled) return;
    setIsDelete(true);
    await remove(state.id, () => setIsDeleteModal(false));
    setIsDelete(false);
  };

  return (
    <>
      <AlertModalCore
        handleClose={() => setIsDeleteModal(false)}
        handleSubmit={handleDeleteState}
        isSubmitting={isDelete}
        isOpen={isDeleteModal}
        title="Delete State"
        content={
          <>
            Are you sure you want to delete state- <span className="font-medium text-primary">{state?.name}</span>? All
            of the data related to the state will be permanently removed. This action cannot be undone.
          </>
        }
      />

      <button
        type="button"
        aria-label={t("common.actions.delete")}
        className={cn(
          "flex h-5 w-5 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm transition-colors focus:outline-none",
          isDeleteDisabled ? "bg-surface-2 text-secondary" : "text-danger-primary hover:bg-layer-1"
        )}
        disabled={isDeleteDisabled}
        onClick={() => setIsDeleteModal(true)}
      >
        <Tooltip
          label={
            state.default ? "Cannot delete the default state." : totalStates === 1 ? `Cannot have an empty group.` : ``
          }
          layout="stacked"
          disabled={!isDeleteDisabled || isMobile}
        >
          {isDelete ? (
            <LoadingOutline className="h-3.5 w-3.5 text-secondary" />
          ) : (
            <CloseOutline className="h-3.5 w-3.5" />
          )}
        </Tooltip>
      </button>
    </>
  );
});
