// Runs oxlint in the current directory (a workspace package) and requires the number of warnings
// to equal the cap given as the only argument. Each package's check:lint script is the one place
// that holds its cap; the cap may only go down (docs/v0/M0-foundation/M0-design.md 5.2,
// docs/v0/M1-frontend-trim/M1-design.md 7.1).
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";

const args = process.argv.slice(2);
if (args.length !== 1 || !/^\d+$/.test(args[0])) {
  console.error("usage: node tools/lint-cap.mjs <cap>");
  process.exit(2);
}
const cap = Number(args[0]);
const { name } = JSON.parse(readFileSync("package.json", "utf8"));

const oxlint = spawnSync("oxlint", ["--format=json", "."], { encoding: "utf8", maxBuffer: 256 * 1024 * 1024 });
if (oxlint.error) throw oxlint.error;

let diagnostics;
try {
  ({ diagnostics } = JSON.parse(oxlint.stdout));
} catch {
  process.stderr.write(oxlint.stdout + oxlint.stderr);
  console.error(`${name}: oxlint did not print a JSON report (exit code ${oxlint.status}).`);
  process.exit(1);
}

const position = (d) => d.labels[0]?.span ?? { line: 0, column: 0 };
const byLocation = (a, b) =>
  a.filename.localeCompare(b.filename) ||
  position(a).line - position(b).line ||
  position(a).column - position(b).column;
const print = (list) => {
  for (const d of list.toSorted(byLocation)) {
    const { line, column } = position(d);
    // a file that does not parse gives a diagnostic without a rule code
    console.error(`${d.filename}:${line}:${column}  ${d.code ?? "syntax"}  ${d.message}`);
  }
};

const count = (n, noun) => `${n} ${noun}${n === 1 ? "" : "s"}`;
const warnings = diagnostics.filter((d) => d.severity === "warning");
const errors = diagnostics.filter((d) => d.severity !== "warning");

if (errors.length > 0) {
  print(errors);
  console.error(`${name}: oxlint reports ${count(errors.length, "error")} (listed above); errors are never allowed.`);
  process.exit(1);
}
if (warnings.length > cap) {
  print(warnings);
  console.error(
    `${name}: more oxlint warnings than the cap of ${cap}. Fix the new warnings (every warning is listed above); the cap may only go down.`
  );
  process.exit(1);
}
if (warnings.length < cap) {
  console.error(
    `${name}: ${count(warnings.length, "oxlint warning")}, fewer than the cap of ${cap}. Lower the cap in the check:lint script of package.json to ${warnings.length} in the same commit.`
  );
  process.exit(1);
}
console.log(`${name}: ${count(warnings.length, "oxlint warning")}, equal to the cap.`);
