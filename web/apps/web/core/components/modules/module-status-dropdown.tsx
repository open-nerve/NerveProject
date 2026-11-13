/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { ModuleStatus } from "@nerve/api-client";
import { MODULE_STATUS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { ModuleStatusIcon } from "@nerve/propel/icons";
import { CustomSelect } from "@nerve/ui";
// hooks
import { useModules } from "@/hooks/store/use-modules";
// local imports
import { useModuleChange } from "./use-module-change";

type Props = {
  moduleId: string;
  /** Whether the caller changes the module's status here: he writes its project's modules, and it is not archived. */
  editable: boolean;
};

/**
 * A module's status as nerve gives it (P6a spec 2.6), a select of the six statuses while editable: a status picked is
 * nerve's to make (useModuleChange), the select busy until it has, and the status then nerve's. None while the store
 * does not give the module.
 */
export const ModuleStatusDropdown = observer(function ModuleStatusDropdown({ moduleId, editable }: Props) {
  // hooks
  const { t } = useTranslation();
  const { getModuleById, update } = useModules();
  const { busy, send } = useModuleChange();
  const module = getModuleById(moduleId);
  if (!module) return null;
  const status = MODULE_STATUS.find((one) => one.value === module.status);
  const disabled = !editable || busy;

  return (
    <CustomSelect
      customButton={
        <span
          className={`flex h-6 w-20 items-center justify-center rounded-sm text-center text-11 ${
            disabled ? "cursor-not-allowed" : "cursor-pointer"
          }`}
          style={{
            color: status?.color,
            backgroundColor: status && `${status.color}20`,
          }}
        >
          {status && t(status.i18n_label)}
        </span>
      }
      value={module.status}
      onChange={(picked: ModuleStatus) => {
        if (picked !== module.status) void send(() => update(module.id, { status: picked }));
      }}
      disabled={disabled}
    >
      {MODULE_STATUS.map((one) => (
        <CustomSelect.Option key={one.value} value={one.value}>
          <div className="flex items-center gap-2">
            <ModuleStatusIcon status={one.value} />
            {t(one.i18n_label)}
          </div>
        </CustomSelect.Option>
      ))}
    </CustomSelect>
  );
});
