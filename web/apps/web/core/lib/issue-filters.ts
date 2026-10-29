/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueListGroupField, IssueListOrder, ProjectIssueFilters } from "@nerve/api-client";
import type { TWorkItemFilterExpression, TWorkItemFilterConditionData } from "@nerve/types";
import type { MemberIssueQuery, ProjectIssueQuery } from "@/services/issue/issue-list.service";

const fields = [
  "state_id",
  "state_group",
  "priority",
  "label_id",
  "assignee_id",
  "created_by_id",
  "mention_id",
  "subscriber_id",
  "start_date",
  "target_date",
] as const;
type Display = {
  layout?: string;
  group_by?: string | null;
  sub_group_by?: string | null;
  order_by?: string;
  sub_issue?: boolean;
};
const aliases: Record<string, keyof ProjectIssueFilters> = {
  state: "state_id",
  labels: "label_id",
  assignees: "assignee_id",
  created_by: "created_by_id",
  mentions: "mention_id",
  subscriber: "subscriber_id",
};
const groups: Record<string, IssueListGroupField> = {
  state: "state_id",
  "state_detail.group": "state_group",
  priority: "priority",
  labels: "label_id",
  assignees: "assignee_id",
  created_by: "created_by_id",
  target_date: "target_date",
  project: "project_id",
};
const orders: IssueListOrder[] = [
  "created_at",
  "-created_at",
  "updated_at",
  "-updated_at",
  "start_date",
  "-start_date",
  "target_date",
  "-target_date",
  "priority",
  "-priority",
  "state_name",
  "-state_name",
  "label_name",
  "-label_name",
  "assignee_name",
  "-assignee_name",
  "sub_issues_count",
  "-sub_issues_count",
  "link_count",
  "-link_count",
  "sort_order",
];
const names: Record<string, string> = {
  state__name: "state_name",
  labels__name: "label_name",
  assignees__first_name: "assignee_name",
};

type FilterField = (typeof fields)[number];
const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);
const isDateField = (field: string) => field === "start_date" || field === "target_date";
const isId = (value: string) => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
function isDay(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const [year, month, day] = value.split("-").map(Number);
  const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const days = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  return year >= 1 && month >= 1 && month <= 12 && day >= 1 && day <= days[month - 1];
}
function isDateCondition(value: string): boolean {
  if (value === "none" || isDay(value)) return true;
  const bounds = value.split("..");
  if (bounds.length !== 2) return false;
  const [from, to] = bounds;
  return Boolean(from || to) && (!from || isDay(from)) && (!to || isDay(to)) && (!from || !to || from <= to);
}
function validValue(field: FilterField | "project_id", value: string): boolean {
  switch (field) {
    case "state_group":
      return ["backlog", "unstarted", "started", "completed", "cancelled"].includes(value);
    case "priority":
      return ["urgent", "high", "medium", "low", "none"].includes(value);
    case "start_date":
    case "target_date":
      return isDateCondition(value);
    case "label_id":
      return value === "none" || isId(value);
    case "assignee_id":
      return value === "none" || value === "me" || isId(value);
    case "created_by_id":
    case "mention_id":
    case "subscriber_id":
      return value === "me" || isId(value);
    default:
      return isId(value);
  }
}
function cleanValues(field: FilterField | "project_id", input: unknown): string[] | undefined {
  if (!Array.isArray(input) || !input.length) return undefined;
  const values: string[] = [];
  for (const value of input) {
    if (typeof value !== "string" || !validValue(field, value)) return undefined;
    values.push(isId(value) ? value.toLowerCase() : value);
  }
  if (isDateField(field) && values.includes("none") && values.length > 1) return undefined;
  return values;
}
function cleanFilters(input: unknown): ProjectIssueFilters {
  const result: ProjectIssueFilters = {};
  if (!isRecord(input)) return result;
  for (const field of fields) {
    const values = cleanValues(field, input[field]);
    if (values) result[field] = values;
  }
  return result;
}

const memberFields = ["state_group", "priority", "label_id", "start_date", "target_date"] as const;
function cleanMemberFields(input: unknown): ProjectIssueFilters {
  const cleaned = cleanFilters(input);
  return Object.fromEntries(memberFields.flatMap((key) => (cleaned[key] ? [[key, cleaned[key]]] : [])));
}

/** Invalid values or unrepresentable ANDs clear the whole field, never widen it into an OR. */
export function toProjectFilters(expression: TWorkItemFilterExpression): ProjectIssueFilters {
  const result: ProjectIssueFilters = {};
  if (!isRecord(expression)) return result;
  const conditions = "and" in expression ? expression.and : [expression];
  if (!Array.isArray(conditions)) return result;
  const seen = new Set<FilterField>();
  const invalid = new Set<FilterField>();
  for (const condition of conditions) {
    if (!isRecord(condition) || "and" in condition) continue;
    for (const [key, input] of Object.entries(condition)) {
      const at = key.lastIndexOf("__");
      const property = key.slice(0, at);
      const field = fields.find((candidate) => candidate === (aliases[property] ?? property));
      if (!field) continue;
      const operator = key.slice(at + 2);
      const date = isDateField(field);
      if (seen.has(field) && !date) invalid.add(field);
      seen.add(field);
      if (typeof input !== "string" || !(date ? ["exact", "range"] : ["exact", "in"]).includes(operator)) {
        invalid.add(field);
        continue;
      }
      const value = input === "None" ? "none" : input;
      const values =
        operator === "range"
          ? [value.replace(",", "..")]
          : operator === "in"
            ? value.split(",").map((v) => (v === "None" ? "none" : v))
            : [value];
      // A date exact condition is a single day/none; range must carry both slots, including an empty bound.
      if (date && (operator === "range" ? value.split(",").length !== 2 : value !== "none" && !isDay(value))) {
        invalid.add(field);
        continue;
      }
      const cleaned = cleanValues(field, values);
      if (!cleaned) invalid.add(field);
      else result[field] = [...(result[field] ?? []), ...cleaned];
    }
  }
  for (const field of invalid) delete result[field];
  return cleanFilters(result);
}

/** Member views additionally accept one validated project condition. */
export function toMemberFilters(
  expression: TWorkItemFilterExpression
): ProjectIssueFilters & Pick<MemberIssueQuery, "project_id"> {
  const result = cleanMemberFields(toProjectFilters(expression));
  const nodes = isRecord(expression) ? ("and" in expression ? expression.and : [expression]) : [];
  const projects = Array.isArray(nodes)
    ? nodes.flatMap((node) =>
        isRecord(node) && !("and" in node) ? Object.entries(node).filter(([key]) => key.startsWith("project_id__")) : []
      )
    : [];
  const [key, value] = projects.length === 1 ? projects[0] : [];
  const ids =
    typeof value === "string"
      ? key === "project_id__in"
        ? value.split(",")
        : key === "project_id__exact"
          ? [value]
          : undefined
      : undefined;
  const projectIds = cleanValues("project_id", ids);
  return { ...result, ...(projectIds ? { project_id: projectIds } : {}) };
}

export function toFilterExpression(filters: ProjectIssueFilters): TWorkItemFilterExpression {
  const and: TWorkItemFilterConditionData[] = [];
  const cleaned = cleanFilters(filters);
  for (const field of fields) {
    const values = cleaned[field];
    if (!values) continue;
    if (isDateField(field))
      for (const value of values) {
        and.push(
          value.includes("..") ? { [`${field}__range`]: value.replace("..", ",") } : { [`${field}__exact`]: value }
        );
      }
    else and.push({ [`${field}__in`]: values.join(",") });
  }
  return and.length ? { and } : {};
}

function group(field: string | null | undefined): IssueListGroupField | undefined {
  return Object.entries(groups).find(([alias, value]) => field === alias || field === value)?.[1];
}
function order(value = "-created_at"): IssueListOrder {
  if (typeof value !== "string") return "-created_at";
  const negative = value.startsWith("-");
  const bare = negative ? value.slice(1) : value;
  const mapped = `${negative ? "-" : ""}${names[bare] ?? bare}`;
  return orders.find((entry) => entry === mapped) ?? "-created_at";
}
export function projectListQuery(filters: ProjectIssueFilters, display: Display = {}): ProjectIssueQuery {
  display ??= {};
  const groupBy =
    display.layout === "calendar"
      ? "target_date"
      : display.layout === "spreadsheet"
        ? undefined
        : group(display.group_by);
  const sub = display.layout === "kanban" ? group(display.sub_group_by) : undefined;
  return {
    ...cleanFilters(filters),
    order_by: order(display.order_by),
    sub_issue: display.sub_issue === false ? "false" : "true",
    ...(groupBy && groupBy !== "project_id" ? { group_by: groupBy } : {}),
    ...(groupBy && groupBy !== "project_id" && sub && sub !== groupBy && sub !== "project_id"
      ? { sub_group_by: sub }
      : {}),
  };
}
export function memberListQuery(
  filters: ProjectIssueFilters & Pick<MemberIssueQuery, "project_id">,
  display: Display = {},
  relation: MemberIssueQuery["relation"]
): MemberIssueQuery {
  display ??= {};
  const groupBy = group(display.group_by);
  const cleaned = cleanMemberFields(filters);
  const projectIds = cleanValues("project_id", filters?.project_id);
  return {
    relation,
    ...(projectIds ? { project_id: projectIds } : {}),
    ...cleaned,
    order_by: order(display.order_by),
    sub_issue: display.sub_issue === false ? "false" : "true",
    ...(groupBy && !["state_id", "assignee_id", "created_by_id"].includes(groupBy) ? { group_by: groupBy } : {}),
  };
}

/** Existing display controls name groups differently from the generated wire vocabulary. */
export function issueFieldForGroup(field: string): keyof Issue {
  const grouped = group(field);
  switch (grouped) {
    case "label_id":
      return "label_ids";
    case "assignee_id":
      return "assignee_ids";
    case "state_group":
      return "state_id";
    default:
      return grouped ?? "state_id";
  }
}

export type IssueListOptions = { limit: number; query?: ProjectIssueQuery };
