/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { State } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
// components
import { StateForm } from "@/components/project-states";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import type { TStateFormData } from "./form";

type TStateUpdate = {
  state: State;
  updateStateCallback: TStateOperationsCallbacks["updateState"];
  handleClose: () => void;
};

export const StateUpdate = observer(function StateUpdate(props: TStateUpdate) {
  const { state, updateStateCallback, handleClose } = props;
  const { t } = useTranslation();
  // states
  const [loader, setLoader] = useState(false);

  const onCancel = () => {
    setLoader(false);
    handleClose();
  };

  const onSubmit = async (formData: TStateFormData) => {
    if (!state.id) return;

    try {
      await updateStateCallback(state.id, formData);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "State updated successfully.",
      });
      handleClose();
    } catch (error) {
      setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
    }
  };

  return (
    <StateForm
      // the fields the form edits, all a change sends: nerve's StateUpdate takes no others
      data={{ name: state.name, color: state.color, description: state.description }}
      onSubmit={onSubmit}
      onCancel={onCancel}
      buttonDisabled={loader}
      buttonTitle={loader ? `Updating` : `Update`}
    />
  );
});
