/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { E_PASSWORD_STRENGTH } from "@nerve/constants";

// The server's password lengths, in UTF-16 code units like password.length
// (server/internal/modules/identity/domain/password.go).
const MIN_PASSWORD_LENGTH = 8;
const MAX_PASSWORD_LENGTH = 128;

const isLengthValid = (password: string) =>
  password.length >= MIN_PASSWORD_LENGTH && password.length <= MAX_PASSWORD_LENGTH;

/** The password rules, in the order they are shown; the caller gives each its text. */
export type TPasswordCriterionKey = "length" | "uppercase" | "lowercase" | "number" | "special";

/** The one set of the password rules, each with its test: the strength and the rules shown both come from it. */
const PASSWORD_RULES: { key: TPasswordCriterionKey; test: (password: string) => boolean }[] = [
  { key: "length", test: isLengthValid },
  { key: "uppercase", test: (password) => /[A-Z]/.test(password) },
  { key: "lowercase", test: (password) => /[a-z]/.test(password) },
  { key: "number", test: (password) => /[0-9]/.test(password) },
  { key: "special", test: (password) => /[!@#$%^&*()\-_+=[\]{}|;:'",.<>?/]/.test(password) },
];

type PasswordCriteria = {
  key: TPasswordCriterionKey;
  isValid: boolean;
};

/**
 * Get password criteria for validation display
 */
export const getPasswordCriteria = (password: string): PasswordCriteria[] =>
  PASSWORD_RULES.map(({ key, test }) => ({ key, isValid: test(password) }));

/**
 * Calculate password strength based on various criteria
 */
export const getPasswordStrength = (password: string): E_PASSWORD_STRENGTH => {
  if (!password) {
    return E_PASSWORD_STRENGTH.EMPTY;
  }

  if (!isLengthValid(password)) {
    return E_PASSWORD_STRENGTH.LENGTH_NOT_VALID;
  }

  return getPasswordCriteria(password).every((criterion) => criterion.isValid)
    ? E_PASSWORD_STRENGTH.STRENGTH_VALID
    : E_PASSWORD_STRENGTH.STRENGTH_NOT_VALID;
};
