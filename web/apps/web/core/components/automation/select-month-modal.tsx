/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "react-router";
// react-hook-form
import { Controller, useForm } from "react-hook-form";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { Button } from "@nerve/propel/button";
import type { ProjectUpdate } from "@nerve/api-client";
// ui
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// types
/** The auto-archiving's range, as ProjectUpdate takes it: a number of months. */
type TArchiveIn = Pick<ProjectUpdate, "archive_in">;
type Props = {
  isOpen: boolean;
  initialValues: TArchiveIn;
  handleClose: () => void;
  /** Sends the range; calls done once nerve has made it, and settles once the page has followed it. */
  handleChange: (formData: TArchiveIn, done: () => void) => Promise<void>;
};

export function SelectMonthModal({ initialValues, isOpen, handleClose, handleChange }: Props) {
  const { workspaceSlug, projectId } = useParams();

  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    control,
    reset,
  } = useForm<TArchiveIn>({
    defaultValues: initialValues,
  });

  const onClose = () => {
    handleClose();
    reset(initialValues);
  };

  // the modal waits for nerve, and cannot be closed meanwhile: it closes once nerve has made the range
  const onSubmit = async (formData: TArchiveIn) => {
    if (!workspaceSlug && !projectId) return;
    await handleChange(formData, onClose);
  };

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isSubmitting ? undefined : onClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
      <form onSubmit={handleSubmit(onSubmit)}>
        <div>
          <h3 className="text-16 leading-6 font-medium text-primary">Customize time range</h3>
          <div className="mt-8 flex items-center gap-2">
            <div className="flex w-full flex-col justify-center gap-1">
              <Controller
                control={control}
                name="archive_in"
                rules={{
                  required: "Select a month between 1 and 12.",
                  min: 1,
                  max: 12,
                }}
                render={({ field: { value, onChange, ref } }) => (
                  <div className="relative flex w-full flex-col justify-center gap-1">
                    <Field name="archive_in" invalid={Boolean(errors.archive_in)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="archive_in"
                          name="archive_in"
                          type="number"
                          value={value ?? ""}
                          // the field holds a number, which nerve's ProjectUpdate takes; none while it is empty
                          onChange={(event) =>
                            onChange(event.target.value === "" ? undefined : Number(event.target.value))
                          }
                          ref={ref}
                          placeholder="Enter Months"
                          min={1}
                          max={12}
                        />
                      </InputGroup>
                    </Field>
                    <span className="absolute top-2.5 right-8 text-13 text-secondary">Months</span>
                  </div>
                )}
              />
              {errors.archive_in && (
                <span className="px-1 text-13 text-danger-primary">Select a month between 1 and 12.</span>
              )}
            </div>
          </div>
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
            {isSubmitting ? "Submitting..." : "Submit"}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
