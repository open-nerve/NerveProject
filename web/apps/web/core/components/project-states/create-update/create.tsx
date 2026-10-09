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
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import type { TStateFormData } from "./form";

type TStateCreate = {
  groupKey: StateGroup;
  createStateCallback: TStateOperationsCallbacks["createState"];
  handleClose: () => void;
};

export const StateCreate = observer(function StateCreate(props: TStateCreate) {
  const { groupKey, createStateCallback, handleClose } = props;
  const toastRefusal = useRefusalToast();

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
      toastRefusal(error);
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
