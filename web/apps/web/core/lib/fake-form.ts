/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for react-hook-form, for the tests of a form's component rendered on the server: a test mocks the module
// with it over what it keeps (vi.hoisted), as
//   vi.mock("react-hook-form", async () => (await import("@/lib/fake-form")).fakeForm(form));
// The component's useForm keeps there the submit the component gives handleSubmit, which the test calls with the
// values as the form would, and gives the component the test's setError and reset, which record what it says under
// its fields (underFields, fake-refusal.ts) and its emptying. The fields are empty and valid, nothing is being
// submitted, and Controller renders nothing.

import type { Mock } from "vitest";

/** What the stand-in keeps of a form whose values are V: the submit the component gave it, and the form's calls. */
type KeptForm<V> = { submit: ((values: V) => Promise<void>) | undefined; setError: Mock; reset?: Mock };

/** react-hook-form, as the component reads it, keeping in form what the component gives it. */
export function fakeForm<V>(form: KeptForm<V>) {
  return {
    useForm: () => ({
      handleSubmit: (submit: (values: V) => Promise<void>) => {
        form.submit = submit;
        return () => undefined;
      },
      control: {},
      watch: () => "",
      setError: form.setError,
      reset: form.reset,
      formState: { errors: {}, isSubmitting: false, isValid: true },
    }),
    Controller: () => null,
  };
}
