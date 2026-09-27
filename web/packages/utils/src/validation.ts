/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/**
 * Input Validation Utilities
 * Following OWASP Input Validation best practices using allowlist approach
 *
 * Security: Blocks injection-risk characters: < > ' " % # { } [ ] * ^ !
 * These patterns are designed to prevent XSS, SQL injection, template injection,
 * and other security vulnerabilities while maintaining good UX
 */

// =============================================================================
// VALIDATION REGEX PATTERNS
// =============================================================================

/**
 * Company/Organization Name Pattern (for company_name, workspace names)
 * Allows: Unicode letters (\p{L}), numbers (\p{N}), spaces, underscores, hyphens
 * Use case: International business names like "Société Générale", "株式会社", "Müller GmbH"
 * Blocks: Special punctuation and injection-risk chars
 */
const COMPANY_NAME_REGEX = /^[\p{L}\p{N}\s_-]+$/u;

/**
 * Requires at least one Unicode letter or digit in a string.
 * Used to reject symbol-only inputs like "-_________-" in workspace/company names.
 */
const HAS_ALPHANUMERIC_REGEX = /[\p{L}\p{N}]/u;

/**
 * URL Slug Pattern (for workspace slugs, URL-safe identifiers)
 * Allows: Unicode letters (\p{L}), numbers (\p{N}), underscores, hyphens
 * Use case: International URL-safe identifiers like "josé-workspace", "李明-project"
 * Blocks: Spaces and special characters (URL encoding will handle Unicode in actual URLs)
 */
const SLUG_REGEX = /^[\p{L}\p{N}_-]+$/u;

// =============================================================================
// VALIDATION FUNCTIONS
// =============================================================================

/**
 * @description Validates company and organization names
 * @param {string} workspaceName - Workspace name to validate
 * @param {boolean} required - Whether the field is required
 * @returns {boolean | string} true if valid, error message if invalid
 * @example
 * validateWorkspaceName("Acme Corp") // returns true
 * validateWorkspaceName("Acme_Corp-123") // returns true
 * validateWorkspaceName("Acme{Corp}") // returns error message
 */
export const validateWorkspaceName = (workspaceName: string, required: boolean = false): boolean | string => {
  if (!workspaceName || workspaceName.trim() === "") {
    return required ? "Workspace name is required" : true;
  }

  if (workspaceName.length > 80) {
    return "Workspace name must be 80 characters or less";
  }

  if (hasInjectionRiskChars(workspaceName)) {
    return "Workspace name cannot contain special characters like < > ' \" { } [ ] * ^ ! # %";
  }

  if (!COMPANY_NAME_REGEX.test(workspaceName)) {
    return "Workspace name can only contain letters, numbers, spaces, hyphens, and underscores";
  }

  if (!HAS_ALPHANUMERIC_REGEX.test(workspaceName)) {
    return "Workspace name must contain at least one letter or number";
  }

  return true;
};

/**
 * @description Validates URL slugs and identifiers
 * @param {string} slug - Slug to validate
 * @returns {boolean | string} true if valid, error message if invalid
 * @example
 * validateSlug("my-workspace") // returns true
 * validateSlug("my_workspace_123") // returns true
 * validateSlug("my workspace") // returns error message (spaces not allowed)
 */
export const validateSlug = (slug: string): boolean | string => {
  if (!slug || slug.trim() === "") {
    return "Slug is required";
  }

  if (slug.length > 48) {
    return "Slug must be 48 characters or less";
  }

  if (hasInjectionRiskChars(slug)) {
    return "Slug cannot contain special characters like < > ' \" { } [ ] * ^ ! # %";
  }

  if (!SLUG_REGEX.test(slug)) {
    return "Slug can only contain letters, numbers, hyphens, and underscores";
  }

  return true;
};

/**
 * @description Checks if a string contains any injection-risk characters
 * @param {string} input - String to check
 * @returns {boolean} true if injection-risk characters found
 * @example
 * hasInjectionRiskChars("Hello World") // returns false
 * hasInjectionRiskChars("Hello<script>") // returns true
 */
const hasInjectionRiskChars = (input: string): boolean => {
  const injectionRiskPattern = /[<>'"{}[\]*^!#%]/;
  return injectionRiskPattern.test(input);
};
