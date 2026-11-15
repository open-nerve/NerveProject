/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// nerve imports
import type { Module } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { ModuleStatusIcon } from "@nerve/propel/icons";
// local imports
import { PowerKMenuBuilder } from "./builder";

type Props = {
  modules: Module[];
  onSelect: (module: Module) => void;
  /** The modules ticked: those a work item is in. */
  value?: readonly string[];
};

/** Modules by their names, an archived one's captioned (as the module dropdown lists it), those of value ticked. */
export const PowerKModulesMenu = observer(function PowerKModulesMenu({ modules, onSelect, value }: Props) {
  const { t } = useTranslation();
  return (
    <PowerKMenuBuilder
      items={modules}
      getKey={(module) => module.id}
      getIconNode={(module) => <ModuleStatusIcon status={module.status} className="size-3.5 shrink-0" />}
      getValue={(module) => module.name}
      getLabel={(module) =>
        module.archived_at ? (
          <>
            <span className="truncate">{module.name}</span>
            <span className="shrink-0 text-caption-sm-regular text-placeholder">
              <span className="sr-only">, </span>
              {t("common.archived")}
            </span>
          </>
        ) : (
          module.name
        )
      }
      isSelected={(module) => !!value?.includes(module.id)}
      onSelect={onSelect}
      emptyTextKey="power_k.menus.no_modules"
    />
  );
});
