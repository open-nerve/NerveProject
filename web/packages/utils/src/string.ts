/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Content, JSONContent } from "@nerve/types";

/**
 * @description Replaces underscores with spaces in snake_case strings
 * @param {string} str - String to replace underscores in
 * @returns {string} String with underscores replaced by spaces
 * @example
 * replaceUnderscoreIfSnakeCase("snake_case") // returns "snake case"
 */
export const replaceUnderscoreIfSnakeCase = (str: string) => str.replace(/_/g, " ");

/**
 * @description Truncates text to specified length and adds ellipsis
 * @param {string} str - String to truncate
 * @param {number} length - Maximum length before truncation
 * @returns {string} Truncated string with ellipsis if needed
 * @example
 * truncateText("This is a long text", 7) // returns "This is..."
 */
export const truncateText = (str: string, length: number) => {
  if (!str || str === "") return "";

  return str.length > length ? `${str.substring(0, length)}...` : str;
};

/**
 * @description Copies full URL (origin + path) to clipboard
 * @param {string} path - URL path to copy
 * @returns {Promise<void>} Promise that resolves when copying is complete
 * @example
 * await copyUrlToClipboard("issues/123") // copies "https://example.com/issues/123"
 */
export const copyUrlToClipboard = async (path: string) => {
  // get origin or default to empty string if not in browser
  const originUrl = typeof window !== "undefined" ? window.location.origin : "";
  // create URL object and ensure proper path formatting
  const url = new URL(path, originUrl);
  await copyTextToClipboard(url.toString());
};

/**
 * @description Formats number count, showing "99+" for numbers over 99
 * @param {number} number - Number to format
 * @returns {string} Formatted number string
 * @example
 * getNumberCount(50) // returns "50"
 * getNumberCount(100) // returns "99+"
 */
export const getNumberCount = (number: number): string => {
  if (number > 99) {
    return "99+";
  }
  return number.toString();
};

/**
 * @description: This function will capitalize the first letter of a string
 * @param str String
 * @returns String
 */
export const capitalizeFirstLetter = (str: string) => str.charAt(0).toUpperCase() + str.slice(1);

// the body of an HTML fragment, as the browser parses it
const parseHTML = (html: string) => new DOMParser().parseFromString(html, "text/html").body;

/**
 * @description: This function will remove all the HTML tags from the string and truncate the string to the specified length
 * @param {string} html
 * @param {number} length
 * @return {string}
 * @example:
 * const html = "<p>Some text</p>";
 * const text = stripAndTruncateHTML(html);
 * console.log(text); // Some text
 */
export const stripAndTruncateHTML = (html: string, length: number = 55) =>
  truncateText((parseHTML(html).textContent ?? "").trim(), length);

/**
 * @returns {boolean} true if email is valid, false otherwise
 * @description Returns true if email is valid, false otherwise
 * @param {string} email string to check if it is a valid email
 * @example checkEmailValidity("hello world") => false
 * @example checkEmailValidity("user@example.com") => true
 */
export const checkEmailValidity = (email: string): boolean => {
  if (!email) return false;

  const isEmailValid =
    /^(([^<>()[\]\\.,;:\s@\"]+(\.[^<>()[\]\\.,;:\s@\"]+)*)|(\".+\"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/.test(
      email
    );

  return isEmailValid;
};

// true when the HTML has no text and none of the allowed tags
export const isEmptyHtmlString = (htmlString: string, allowedHTMLTags: string[] = []) => {
  const body = parseHTML(htmlString);
  return (
    (body.textContent ?? "").trim() === "" && !(allowedHTMLTags.length && body.querySelector(allowedHTMLTags.join(",")))
  );
};

/**
 * @description
 * Check if a JSONContent object is empty
 * @param {JSONContent} content
 * @returns {boolean}
 */
const isJSONContentEmpty = (content: JSONContent | undefined): boolean => {
  // If it has text, check if text is meaningful
  if (!content) {
    return true;
  }
  if (content.text !== undefined) {
    return !content.text || content.text.trim() === "";
  }

  // If it has no content array, consider it empty
  if (!content.content || content.content.length === 0) {
    // Special case: empty paragraph nodes should be considered empty
    if (content.type === "paragraph" || content.type === "doc") {
      return true;
    }
    // For other node types without content (like hard breaks), check if they're meaningful
    return (
      content.type !== "hardBreak" &&
      content.type !== "image" &&
      content.type !== "mention-component" &&
      content.type !== "image-component"
    );
  }

  // Check if all nested content is empty
  return content.content.every(isJSONContentEmpty);
};

/**
 * @description
 * This function will check if the comment is empty or not.
 * It returns true if comment is empty.
 * Now supports TipTap Content types (HTMLContent, JSONContent, JSONContent[], null)
 *
 * For HTML content:
 * 1. If comment is undefined/null
 * 2. If comment is an empty string
 * 3. If comment is "<p></p>"
 * 4. If comment contains only empty HTML tags
 *
 * For JSON content:
 * 1. If content is null/undefined
 * 2. If content has no meaningful text or nested content
 * 3. If all nested content is empty
 *
 * @param {Content} comment - TipTap Content type
 * @returns {boolean}
 */
export const isCommentEmpty = (comment: Content | undefined): boolean => {
  // Handle null/undefined
  if (!comment) return true;

  // Handle HTMLContent (string)
  if (typeof comment === "string") {
    return (
      comment.trim() === "" ||
      comment === "<p></p>" ||
      isEmptyHtmlString(comment, ["img", "mention-component", "image-component"])
    );
  }

  // Handle JSONContent[] (array)
  if (Array.isArray(comment)) {
    return comment.length === 0 || comment.every(isJSONContentEmpty);
  }

  // Handle JSONContent (object)
  return isJSONContentEmpty(comment);
};

/**
 * @description
 * This function test whether a URL is valid or not.
 *
 * It accepts URLs with or without the protocol.
 * @param {string} url
 * @returns {boolean}
 * @example
 * checkURLValidity("https://example.com") => true
 * checkURLValidity("example.com") => true
 * checkURLValidity("example") => false
 */
export const checkURLValidity = (url: string): boolean => {
  if (!url) return false;

  // regex to support complex query parameters and fragments
  const urlPattern =
    /^(https?:\/\/)?((([a-z\d-]+\.)*[a-z\d-]+\.[a-z]{2,6})|(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}))(:\d+)?(\/[\w.-]*)*(\?[^#\s]*)?(#[\w-]*)?$/i;

  return urlPattern.test(url);
};

/**
 * @returns {boolean} true if searchQuery is substring of text in the same order, false otherwise
 * @description Returns true if searchQuery is substring of text in the same order, false otherwise
 * @param {string} text string to compare from
 * @param {string} searchQuery
 * @example substringMatch("hello world", "hlo") => true
 * @example substringMatch("hello world", "hoe") => false
 */
export const substringMatch = (text: string, searchQuery: string): boolean => {
  try {
    let searchIndex = 0;

    for (let i = 0; i < text.length; i++) {
      if (text[i].toLowerCase() === searchQuery[searchIndex]?.toLowerCase()) searchIndex++;

      // All characters of searchQuery found in order
      if (searchIndex === searchQuery.length) return true;
    }

    // Not all characters of searchQuery found in order
    return false;
  } catch (_err) {
    return false;
  }
};

/**
 * @description Copies text to clipboard
 * @param {string} text - Text to copy
 * @returns {Promise<void>} Promise that resolves when copying is complete
 * @example
 * await copyTextToClipboard("Hello, World!") // copies "Hello, World!" to clipboard
 */
const fallbackCopyTextToClipboard = (text: string) => {
  const textArea = document.createElement("textarea");
  textArea.value = text;

  // Avoid scrolling to bottom
  textArea.style.top = "0";
  textArea.style.left = "0";
  textArea.style.position = "fixed";

  document.body.appendChild(textArea);
  textArea.focus();
  textArea.select();

  try {
    // FIXME: Even though we are using this as a fallback, execCommand is deprecated 👎. We should find a better way to do this.
    // https://developer.mozilla.org/en-US/docs/Web/API/Document/execCommand
    document.execCommand("copy");
  } catch (_err) {
    // catch fallback error
  }

  document.body.removeChild(textArea);
};

/**
 * @description Copies text to clipboard
 * @param {string} text - Text to copy
 * @returns {Promise<void>} Promise that resolves when copying is complete
 * @example
 * await copyTextToClipboard("Hello, World!") // copies "Hello, World!" to clipboard
 */
export const copyTextToClipboard = async (text: string): Promise<void> => {
  if (!navigator.clipboard) {
    fallbackCopyTextToClipboard(text);
    return;
  }
  await navigator.clipboard.writeText(text);
};

/**
 * @description Joins URL path segments properly, removing duplicate slashes using URL encoding
 * @param {...string} segments - URL path segments to join
 * @returns {string} Properly joined URL path
 * @example
 * joinUrlPath("/workspace", "/projects") => "/workspace/projects"
 * joinUrlPath("/workspace", "projects") => "/workspace/projects"
 * joinUrlPath("workspace", "projects") => "/workspace/projects"
 * joinUrlPath("/workspace/", "/projects/") => "/workspace/projects/"
 */
export const joinUrlPath = (...segments: string[]): string => {
  if (segments.length === 0) return "";

  // Filter out empty segments
  const validSegments = segments.filter((segment) => segment !== "");
  if (validSegments.length === 0) return "";

  // Process segments to normalize slashes
  const processedSegments = validSegments.map((segment, index) => {
    let processed = segment;

    // Remove leading slashes from all segments except the first
    while (processed.startsWith("/")) {
      processed = processed.substring(1);
    }

    // Remove trailing slashes from all segments except the last
    if (index < validSegments.length - 1) {
      while (processed.endsWith("/")) {
        processed = processed.substring(0, processed.length - 1);
      }
    }

    return processed;
  });

  // Join segments with single slash
  const joined = processedSegments.join("/");

  // Use URL constructor to normalize the path and handle double slashes
  try {
    // Create a dummy URL to leverage browser's URL normalization
    const dummyUrl = new URL(`http://example.com/${joined}`);
    return dummyUrl.pathname;
  } catch {
    // Fallback: manually handle double slashes by splitting and filtering
    const pathParts = joined.split("/").filter((part) => part !== "");
    return pathParts.length > 0 ? `/${pathParts.join("/")}` : "";
  }
};
