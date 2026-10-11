/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { StartOfTheWeek } from "@nerve/api-client";
import { START_OF_THE_WEEK_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { CustomSelect } from "@nerve/ui";
// components
import { SettingsControlItem } from "@/components/settings/control-item";
// hooks
import { useUserProfile } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

// The i18n key of the day's name.
const startOfWeekLabelKey = (startOfWeek: StartOfTheWeek | undefined) =>
  START_OF_THE_WEEK_OPTIONS.find((option) => option.value === startOfWeek)?.i18n_label;

export const StartOfWeekPreference = observer(function StartOfWeekPreference(props: {
  option: { title: string; description: string };
}) {
  // hooks
  const { data: userProfile, updateUserProfile } = useUserProfile();
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();
  // derived values
  const labelKey = startOfWeekLabelKey(userProfile?.start_of_the_week);

  // It says how nerve settled the change only in the session it was sent in (M3 design 7.1).
  const handleStartOfWeekChange = (val: StartOfTheWeek) =>
    followInSession(() => updateUserProfile({ start_of_the_week: val }), {
      done: () =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("power_k.preferences_actions.toast.generic.success"),
        }),
      failed: toastRefusal,
    });

  return (
    <SettingsControlItem
      title={props.option.title}
      description={props.option.description}
      control={
        <CustomSelect
          value={userProfile?.start_of_the_week}
          label={labelKey && t(labelKey)}
          onChange={handleStartOfWeekChange}
          buttonClassName="border border-subtle-1"
          input
          maxHeight="lg"
          placement="bottom-end"
        >
          <>
            {START_OF_THE_WEEK_OPTIONS.map((day) => (
              <CustomSelect.Option key={day.value} value={day.value}>
                {t(day.i18n_label)}
              </CustomSelect.Option>
            ))}
          </>
        </CustomSelect>
      }
    />
  );
});
