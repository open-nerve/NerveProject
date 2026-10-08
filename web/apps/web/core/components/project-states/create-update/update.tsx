/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { State } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
// components
import { StateForm } from "@/components/project-states";
import type { TStateFormData } from "./form";

type TStateUpdate = {
  state: State;
  updateStateCallback: TStateOperationsCallbacks["updateState"];
  handleClose: () => void;
};

export const StateUpdate = observer(function StateUpdate(props: TStateUpdate) {
  const { state, updateStateCallback, handleClose } = props;
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
      const errorStatus = error as { status: number };
      if (errorStatus?.status === 400) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Another state exists with the same name. Please try again with another name.",
        });
      } else {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "State could not be updated. Please try again.",
        });
      }
    }
  };

  return (
    <StateForm
      data={state}
      onSubmit={onSubmit}
      onCancel={onCancel}
      buttonDisabled={loader}
      buttonTitle={loader ? `Updating` : `Update`}
    />
  );
});
