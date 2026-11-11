/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// nerve imports
import { CyclesOutline } from "@makeplane/propel/icons";
import type { Cycle } from "@nerve/api-client";
// local imports
import { PowerKMenuBuilder } from "./builder";

type Props = {
  cycles: Cycle[];
  onSelect: (cycle: Cycle) => void;
};

export const PowerKCyclesMenu = observer(function PowerKCyclesMenu({ cycles, onSelect }: Props) {
  return (
    <PowerKMenuBuilder
      items={cycles}
      getIcon={() => CyclesOutline}
      getKey={(cycle) => cycle.id}
      getValue={(cycle) => cycle.name}
      getLabel={(cycle) => cycle.name}
      onSelect={onSelect}
      emptyText="No cycles found"
    />
  );
});
