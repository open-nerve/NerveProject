/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api-error";
import { SessionChangedError, SessionUnavailableError } from "@/lib/auth/token-manager";
import {
  FIELD_ERROR_MESSAGES,
  PROBLEM_MESSAGES,
  errorMessageKey,
  fieldErrorKeys,
  needsErrorBanner,
} from "./authentication.helper";

// The message tables against the API's contract (M2 design 3.11, 7.3, 9.4): api/dist/openapi.yaml is the
// bundled description the server is checked against, so a code added there fails here until it has a message.

const spec = readFileSync(new URL("../../../../api/dist/openapi.yaml", import.meta.url), "utf8").split("\n");
const en = JSON.parse(
  readFileSync(new URL("../../../packages/i18n/src/locales/en/auth.json", import.meta.url), "utf8")
) as Record<string, unknown>;

/** The items of every list under a line that matches `key` (block lists: "- item", deeper than the key). */
function listsUnder(key: RegExp): string[] {
  const items: string[] = [];
  spec.forEach((line, i) => {
    const head = key.exec(line);
    if (!head) return;
    const indent = line.length - line.trimStart().length;
    for (const next of spec.slice(i + 1)) {
      const item = /^(\s*)- (\S+)$/.exec(next);
      if (!item || item[1].length < indent) break;
      items.push(item[2]);
    }
  });
  return items;
}

const problemCodes = new Set(listsUnder(/^\s*x-problem-codes:\s*$/));
// FieldError.code's enum: the only "enum:" list under the FieldError schema.
const fieldErrorStart = spec.findIndex((line) => /^ {4}FieldError:$/.test(line));
const fieldCodes = (() => {
  const enumLine = spec.findIndex((line, i) => i > fieldErrorStart && /^\s+enum:$/.test(line));
  const codes: string[] = [];
  for (const line of spec.slice(enumLine + 1)) {
    const item = /^\s+- (\S+)$/.exec(line);
    if (!item) break;
    codes.push(item[1]);
  }
  return new Set(codes);
})();

/** The value at a dotted i18n key of en's auth namespace, or undefined. */
function english(key: string): unknown {
  return key.split(".").reduce<unknown>((node, part) => (node as Record<string, unknown> | undefined)?.[part], en);
}

describe("the message tables", () => {
  it("read the contract: both lists are found in openapi.yaml", () => {
    expect(problemCodes).toContain("identity.email_taken");
    expect(problemCodes).toContain("bad_request");
    expect(fieldCodes).toContain("common_password");
  });

  it("have a message for every problem code of the contract, and no other", () => {
    expect(new Set(Object.keys(PROBLEM_MESSAGES))).toEqual(problemCodes);
  });

  it("have a message for every FieldError.code of the contract, and no other", () => {
    expect(new Set(Object.keys(FIELD_ERROR_MESSAGES))).toEqual(fieldCodes);
  });

  it("point at messages that exist in English (sync-check keeps zh-CN the same)", () => {
    const keys = [...Object.values(PROBLEM_MESSAGES), ...Object.values(FIELD_ERROR_MESSAGES)];
    for (const key of [...keys, "auth.errors.unknown", "auth.errors.unreachable"]) {
      expect(typeof english(key), key).toBe("string");
    }
  });
});

const problem = (code: string) => ({ status: 400, code, title: "" });

describe("errorMessageKey", () => {
  it("gives the message of the problem's code", () => {
    expect(errorMessageKey(new ApiError(409, problem("identity.email_taken")))).toBe("auth.errors.email_taken");
    expect(errorMessageKey(new ApiError(429, problem("rate_limited")))).toBe("auth.errors.rate_limited");
  });

  it("gives the general message for a code it does not know, or an answer without a problem", () => {
    expect(errorMessageKey(new ApiError(418, problem("teapot")))).toBe("auth.errors.unknown");
    expect(errorMessageKey(new ApiError(502, undefined))).toBe("auth.errors.unknown");
    expect(errorMessageKey(new TypeError("Failed to fetch"))).toBe("auth.errors.unknown");
    expect(errorMessageKey(new SessionChangedError())).toBe("auth.errors.unknown");
  });

  it("says the server cannot be reached when the session is unavailable", () => {
    expect(errorMessageKey(new SessionUnavailableError(0))).toBe("auth.errors.unreachable");
  });
});

describe("fieldErrorKeys", () => {
  it("gives each named field the message of its code", () => {
    const error = new ApiError(422, {
      status: 422,
      code: "validation_failed",
      title: "",
      errors: [
        { field: "password", code: "common_password", message: "is too common" },
        { field: "email", code: "required", message: "is required" },
      ],
    });
    expect(fieldErrorKeys(error)).toEqual({
      password: "auth.errors.field.common_password",
      email: "auth.errors.field.required",
    });
  });

  it("gives nothing for a problem without fields, or for another error", () => {
    expect(fieldErrorKeys(new ApiError(409, { status: 409, code: "identity.email_taken", title: "" }))).toEqual({});
    expect(fieldErrorKeys(new TypeError("Failed to fetch"))).toEqual({});
  });
});

describe("needsErrorBanner", () => {
  /** A problem that names these fields. */
  const naming = (...fields: string[]) =>
    new ApiError(422, {
      status: 422,
      code: "validation_failed",
      title: "",
      errors: fields.map((field) => ({ field, code: "required" as const, message: "is required" })),
    });
  const form = ["email", "password"];

  it("leaves the messages under the fields when the form has every field the error names", () => {
    expect(needsErrorBanner(naming("email"), form)).toBe(false);
    expect(needsErrorBanner(naming("email", "password"), form)).toBe(false);
  });

  it("shows the error above the form when it names a field the form does not have", () => {
    expect(needsErrorBanner(naming("first_name"), form)).toBe(true);
    expect(needsErrorBanner(naming("email", "first_name"), form)).toBe(true);
  });

  it("shows an error without fields above the form", () => {
    expect(needsErrorBanner(new ApiError(409, { status: 409, code: "identity.email_taken", title: "" }), form)).toBe(
      true
    );
    expect(needsErrorBanner(new TypeError("Failed to fetch"), form)).toBe(true);
  });
});
