/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { STATE_GROUPS } from "@nerve/constants";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { StateGroup } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
// components
import { StateForm } from "@/components/project-states";
import type { TStateFormData } from "./form";

type TStateCreate = {
  groupKey: StateGroup;
  createStateCallback: TStateOperationsCallbacks["createState"];
  handleClose: () => void;
};

export const StateCreate = observer(function StateCreate(props: TStateCreate) {
  const { groupKey, createStateCallback, handleClose } = props;

  // states
  const [loader, setLoader] = useState(false);

  const onCancel = () => {
    setLoader(false);
    handleClose();
  };

  const onSubmit = async (formData: TStateFormData) => {
    if (!groupKey) return;

    try {
      await createStateCallback({ ...formData, group: groupKey });

      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "State created successfully.",
      });
      handleClose();
    } catch (error) {
      const errorStatus = error as { status: number; data: { error: string } };
      if (errorStatus?.status === 400) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "State with that name already exists. Please try again with another name.",
        });
      } else {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: errorStatus.data.error ?? "State could not be created. Please try again.",
        });
      }
    }
  };

  return (
    <StateForm
      data={{ name: "", description: "", color: STATE_GROUPS[groupKey].color }}
      onSubmit={onSubmit}
      onCancel={onCancel}
      buttonDisabled={loader}
      buttonTitle={loader ? `Creating` : `Create`}
    />
  );
});
