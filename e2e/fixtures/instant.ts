// API timestamps are UTC RFC3339Nano; retain nanoseconds when sorting trimmed fractions.
export const instantKey = (value: string) =>
  value.replace(/(?:\.(\d+))?Z$/, (_, fraction = "") => `.${fraction.padEnd(9, "0")}Z`);
