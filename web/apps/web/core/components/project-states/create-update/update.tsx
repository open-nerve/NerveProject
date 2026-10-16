/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { State } from "@nerve/api-client";
// components
import { StateForm } from "@/components/project-states";
// local imports
import { useStateChanges, type StateUnderField } from "../use-state-changes";
import type { TStateFormData } from "./form";

type TStateUpdate = {
  state: State;
  handleClose: () => void;
};

export const StateUpdate = observer(function StateUpdate(props: TStateUpdate) {
  const { state, handleClose } = props;
  const { update } = useStateChanges();
  const { t } = useTranslation();

  // the form closes once nerve has made the change, and stays with what it holds when nerve refuses it
  const onSubmit = (formData: TStateFormData, underField: StateUnderField) =>
    update(state.id, formData, underField, () => {
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("toast.success"),
        message: t("project_settings.states.updated"),
      });
      handleClose();
    });

  return (
    <StateForm
      // the fields the form edits, all a change sends: nerve's StateUpdate takes no others
      data={{ name: state.name, color: state.color, description: state.description }}
      onSubmit={onSubmit}
      onCancel={handleClose}
      buttonTitle={t("common.update")}
      busyButtonTitle={t("common.updating")}
    />
  );
});
