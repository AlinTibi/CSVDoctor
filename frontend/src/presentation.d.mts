import type { Row } from "./types";
import type { Issue } from "./types";
export function escapeHTML(value: unknown): string;
export function changedCells(before: Row[], after: Row[]): boolean[][];
export function findingGroups(
  counts: Record<string, number>,
  issues: Issue[],
): Map<string, { message: string; severity: string; locations: string[] }>;
