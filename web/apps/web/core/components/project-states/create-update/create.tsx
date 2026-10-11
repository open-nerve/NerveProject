/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { STATE_GROUPS } from "@nerve/constants";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { StateGroup } from "@nerve/api-client";
// components
import { StateForm } from "@/components/project-states";
// local imports
import { useStateChanges, type StateUnderField } from "../use-state-changes";
import type { TStateFormData } from "./form";

type TStateCreate = {
  projectId: string;
  groupKey: StateGroup;
  handleClose: () => void;
};

export const StateCreate = observer(function StateCreate(props: TStateCreate) {
  const { projectId, groupKey, handleClose } = props;
  const { create } = useStateChanges();

  // the form closes once nerve has made the state, and stays with what it holds when nerve refuses it
  const onSubmit = (formData: TStateFormData, underField: StateUnderField) =>
    create(projectId, { ...formData, group: groupKey }, underField, () => {
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "State created successfully.",
      });
      handleClose();
    });

  return (
    <StateForm
      data={{ name: "", description: "", color: STATE_GROUPS[groupKey].color }}
      onSubmit={onSubmit}
      onCancel={handleClose}
      buttonTitle="Create"
      busyButtonTitle="Creating"
    />
  );
});
