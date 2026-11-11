/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { CycleStatus } from "@nerve/api-client";
import { CYCLE_STATUS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { CloseOutline } from "@makeplane/propel/icons";
import { cn } from "@nerve/utils";

type Props = {
  handleRemove: (status: CycleStatus) => void;
  values: readonly CycleStatus[];
};

/** The statuses a cycles list is filtered by, each by its name in the page, each removable. */
export const AppliedStatusFilters = observer(function AppliedStatusFilters(props: Props) {
  const { handleRemove, values } = props;
  const { t } = useTranslation();

  return (
    <>
      {values.map((status) => {
        const statusDetails = CYCLE_STATUS.find((s) => s.value === status);
        const name = statusDetails ? t(statusDetails.i18n_title) : status;
        return (
          <div
            key={status}
            className={cn(
              "flex items-center gap-1 rounded-sm px-1.5 py-1 text-11",
              statusDetails?.bgColor,
              statusDetails?.textColor
            )}
          >
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
