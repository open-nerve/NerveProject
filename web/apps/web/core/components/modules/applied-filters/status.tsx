/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { CloseOutline } from "@makeplane/propel/icons";
// nerve imports
import type { ModuleStatus } from "@nerve/api-client";
import { MODULE_STATUS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { ModuleStatusIcon } from "@nerve/propel/icons";

type Props = {
  handleRemove: (status: ModuleStatus) => void;
  values: readonly ModuleStatus[];
};

/** The statuses a modules list is filtered by, each by its name in the page, each removable. */
export const AppliedStatusFilters = observer(function AppliedStatusFilters(props: Props) {
  const { handleRemove, values } = props;
  const { t } = useTranslation();

  return (
    <>
      {values.map((status) => {
        const statusDetails = MODULE_STATUS.find((s) => s.value === status);
        const name = statusDetails ? t(statusDetails.i18n_label) : status;
        return (
          <div key={status} className="flex items-center gap-1 rounded-sm bg-layer-3 px-1.5 py-1 text-11">
            <ModuleStatusIcon status={status} height="12px" width="12px" />
            {name}
            <button
              type="button"
              className="grid place-items-center text-tertiary hover:text-secondary"
              aria-label={t("common.remove_item", { name })}
              onClick={() => handleRemove(status)}
            >
              <CloseOutline height={10} width={10} aria-hidden="true" />
            </button>
          </div>
        );
      })}
    </>
  );
});
