/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
// nerve imports
import type { UserUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EOnboardingSteps } from "@nerve/types";
import { cn, getFileURL } from "@nerve/utils";
// helpers
import { errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/helpers/authentication.helper";
// hooks
import { useUser } from "@/hooks/store/user";
// local components
import { CommonOnboardingHeader } from "../common";

type Props = {
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => void;
};

type TProfileSetupFormValues = {
  first_name: string;
  last_name: string;
};

/**
 * The field the step shows, whose errors show under it. The step checks only that it is filled: the rules of the
 * names are nerve's (M2 design 4.2), and its refusal shows under the field.
 */
const FIELDS = ["first_name"] as const;

export const ProfileSetupStep = observer(function ProfileSetupStep({ handleStepChange }: Props) {
  const { t } = useTranslation();
  // store hooks
  const { data: user, updateCurrentUser } = useUser();
  // form info
  const {
    handleSubmit,
    control,
    watch,
    setError,
    formState: { errors, isSubmitting, isValid },
  } = useForm<TProfileSetupFormValues>({
    defaultValues: {
      first_name: user?.first_name ?? "",
      last_name: user?.last_name ?? "",
    },
    mode: "onChange",
  });

  /** Saves the names; false when nerve did not, so the step stays (M2 design 7.1: nor for another account). */
  const handleSubmitUserDetail = async (formData: TProfileSetupFormValues): Promise<boolean> => {
    const userDetailsPayload: UserUpdate = {
      first_name: formData.first_name,
      last_name: formData.last_name,
    };
    try {
      await updateCurrentUser(userDetailsPayload);
      return true;
    } catch (error) {
      // A refusal of the name shows under it, anything else in a toast (M2 design 7.3).
      const key = fieldErrorKeys(error).first_name;
      if (key !== undefined) setError("first_name", { type: "manual", message: t(key) });
      if (needsErrorBanner(error, FIELDS))
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
      return false;
    }
  };

  const onSubmit = async (formData: TProfileSetupFormValues) => {
    if (!user) return;
    if (await handleSubmitUserDetail(formData)) handleStepChange(EOnboardingSteps.PROFILE_SETUP);
  };

  const isButtonDisabled = isSubmitting || !isValid;

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-10">
      {/* Header */}
      <CommonOnboardingHeader title="Create your profile." description="This is how you will appear in Nerve." />

      {/* Profile picture: shown, not uploaded, until the file storage (M5) */}
      <div className="flex size-12 items-center justify-center rounded-full bg-accent-primary text-18 font-semibold text-on-color">
        {user?.avatar_url ? (
          <img
            src={getFileURL(user.avatar_url)}
            alt={user.display_name}
            className="h-full w-full rounded-full object-cover"
          />
        ) : (
          <>{watch("first_name")[0] ?? "R"}</>
        )}
      </div>

      <div className="flex w-full flex-col gap-6">
        {/* Name Input */}
        <div className="flex flex-col gap-2">
          <label
            className="block text-13 font-medium text-tertiary after:ml-0.5 after:text-danger-primary after:content-['*']"
            htmlFor="first_name"
          >
            Name
          </label>
          <Controller
            control={control}
            name="first_name"
            rules={{
              required: t("name_is_required"),
            }}
            render={({ field: { value, onChange, ref } }) => (
              <input
                ref={ref}
                id="first_name"
                name="first_name"
                type="text"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                autoFocus
                className={cn(
                  "w-full rounded-md border border-strong bg-surface-1 px-3 py-2 text-secondary transition-all duration-200 placeholder:text-placeholder focus:border-transparent focus:ring-2 focus:ring-accent-strong focus:outline-none",
                  {
                    "border-strong": !errors.first_name,
                    "border-danger-strong": errors.first_name,
                  }
                )}
                placeholder="Enter your full name"
                autoComplete="on"
              />
            )}
          />
          {errors.first_name && <span className="text-13 text-danger-primary">{errors.first_name.message}</span>}
        </div>
      </div>
      {/* Continue Button */}
      <Button variant="primary" type="submit" className="w-full" size="xl" disabled={isButtonDisabled}>
        Continue
      </Button>
    </form>
  );
});
