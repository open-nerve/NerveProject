/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { Command } from "cmdk";
// nerve imports
import { START_OF_THE_WEEK_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { EStartOfTheWeek } from "@nerve/types";
// local imports
import { PowerKModalCommandItem } from "../../modal/command-item";

type Props = {
  onSelect: (day: EStartOfTheWeek) => void;
};

export function PowerKPreferencesStartOfWeekMenu(props: Props) {
  const { onSelect } = props;
  // hooks
  const { t } = useTranslation();

  return (
    <Command.Group>
      {START_OF_THE_WEEK_OPTIONS.map((day) => (
        <PowerKModalCommandItem key={day.value} onSelect={() => onSelect(day.value)} label={t(day.i18n_label)} />
      ))}
    </Command.Group>
  );
}
