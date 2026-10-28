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

/** UI expressions are a local representation; unsupported combinations are cleared. */
export function toProjectFilters(expression: TWorkItemFilterExpression): ProjectIssueFilters {
  const result: ProjectIssueFilters = {};
  const conditions = "and" in expression && Array.isArray(expression.and) ? expression.and : [expression];
  for (const condition of conditions)
    for (const [key, input] of Object.entries(condition)) {
      if (typeof input !== "string") continue;
      const at = key.lastIndexOf("__");
      const field = aliases[key.slice(0, at)] ?? key.slice(0, at);
      const allowed = fields.find((candidate) => candidate === field);
      const operator = key.slice(at + 2);
      if (!allowed || !["exact", "in", "range"].includes(operator)) continue;
      const date = allowed === "start_date" || allowed === "target_date";
      const values = date ? [operator === "range" ? input.replace(",", "..") : input] : input.split(",");
      if (values.some((value) => value.includes("fromnow") || value.includes(";"))) continue;
      result[allowed] = [
        ...(result[allowed] ?? []),
        ...values.filter(Boolean).map((value) => (value === "None" ? "none" : value)),
      ];
    }
  return result;
}

export function toFilterExpression(filters: ProjectIssueFilters): TWorkItemFilterExpression {
  const and: TWorkItemFilterConditionData[] = [];
  for (const field of fields) {
    const values = filters[field];
    if (!values?.length) continue;
    if (field === "start_date" || field === "target_date")
      for (const value of values) {
        and.push(
          value.includes("..") ? { [`${field}__range`]: value.replace("..", ",") } : { [`${field}__exact`]: value }
        );
      }
    else if (field !== "subscriber_id") and.push({ [`${field}__in`]: values.join(",") });
  }
  return and.length ? { and } : {};
}

function group(field: string | null | undefined): IssueListGroupField | undefined {
  return field ? (groups[field] ?? Object.values(groups).find((value) => value === field)) : undefined;
}
function order(value = "-created_at"): IssueListOrder {
  const negative = value.startsWith("-");
  const bare = negative ? value.slice(1) : value;
  const mapped = `${negative ? "-" : ""}${names[bare] ?? bare}`;
  return orders.find((entry) => entry === mapped) ?? "-created_at";
}
export function projectListQuery(filters: ProjectIssueFilters, display: Display = {}): ProjectIssueQuery {
  const groupBy =
    display.layout === "calendar"
      ? "target_date"
      : display.layout === "spreadsheet"
        ? undefined
        : group(display.group_by);
  const sub = display.layout === "kanban" ? group(display.sub_group_by) : undefined;
  return {
    ...filters,
    order_by: order(display.order_by),
    sub_issue: display.sub_issue === false ? "false" : "true",
    ...(groupBy && groupBy !== "project_id" ? { group_by: groupBy } : {}),
    ...(sub && sub !== groupBy && sub !== "project_id" ? { sub_group_by: sub } : {}),
  };
}
export function memberListQuery(
  filters: ProjectIssueFilters & Pick<MemberIssueQuery, "project_id">,
  display: Display = {},
  relation: MemberIssueQuery["relation"]
): MemberIssueQuery {
  const groupBy = group(display.group_by);
  return {
    relation,
    ...(filters.project_id?.length ? { project_id: filters.project_id } : {}),
    ...Object.fromEntries(
      ["state_group", "priority", "label_id", "start_date", "target_date"].flatMap((field) => {
        const key = fields.find((candidate) => candidate === field)!;
        return filters[key] ? [[key, filters[key]]] : [];
      })
    ),
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
