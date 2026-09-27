/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
// nerve imports
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IUser } from "@nerve/types";
import { EOnboardingSteps } from "@nerve/types";
import { cn, getFileURL, validatePersonName } from "@nerve/utils";
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

export const ProfileSetupStep = observer(function ProfileSetupStep({ handleStepChange }: Props) {
  // store hooks
  const { data: user, updateCurrentUser } = useUser();
  // form info
  const {
    handleSubmit,
    control,
    watch,
    formState: { errors, isSubmitting, isValid },
  } = useForm<TProfileSetupFormValues>({
    defaultValues: {
      first_name: user?.first_name ?? "",
      last_name: user?.last_name ?? "",
    },
    mode: "onChange",
  });

  const handleSubmitUserDetail = async (formData: TProfileSetupFormValues) => {
    const userDetailsPayload: Partial<IUser> = {
      first_name: formData.first_name,
      last_name: formData.last_name,
    };
    try {
      await updateCurrentUser(userDetailsPayload);
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error",
        message: "User details update failed. Please try again!",
      });
    }
  };

  const onSubmit = async (formData: TProfileSetupFormValues) => {
    if (!user) return;
    await handleSubmitUserDetail(formData);
    handleStepChange(EOnboardingSteps.PROFILE_SETUP);
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
              required: "Name is required",
              validate: validatePersonName,
              maxLength: {
                value: 50,
                message: "Name must be within 50 characters.",
              },
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
