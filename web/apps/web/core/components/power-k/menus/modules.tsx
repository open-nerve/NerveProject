/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// nerve imports
import type { Module } from "@nerve/api-client";
import { ModuleStatusIcon } from "@nerve/propel/icons";
// local imports
import { PowerKMenuBuilder } from "./builder";

type Props = {
  modules: Module[];
  onSelect: (module: Module) => void;
};

export const PowerKModulesMenu = observer(function PowerKModulesMenu({ modules, onSelect }: Props) {
  return (
    <PowerKMenuBuilder
      items={modules}
      getKey={(module) => module.id}
      getIconNode={(module) => <ModuleStatusIcon status={module.status} className="size-3.5 shrink-0" />}
      getValue={(module) => module.name}
      getLabel={(module) => module.name}
      onSelect={onSelect}
      emptyText="No modules found"
    />
  );
});
