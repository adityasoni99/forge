import type { StaleDoc } from './types.js';

export interface GitLogEntry {
  path: string;
  date: string;
}

export interface DocGardenOptions {
  codeToDocMapping: Record<string, string>;
  staleDaysThreshold: number;
  referenceDate: string;
}

function daysBetween(dateA: string, dateB: string): number {
  const a = new Date(dateA).getTime();
  const b = new Date(dateB).getTime();
  return Math.abs(Math.floor((b - a) / (1000 * 60 * 60 * 24)));
}

export function findStaleDocCandidates(
  codeChanges: GitLogEntry[],
  docFiles: GitLogEntry[],
  options: DocGardenOptions,
): StaleDoc[] {
  const docMap = new Map<string, GitLogEntry>();
  for (const doc of docFiles) {
    docMap.set(doc.path, doc);
  }

  const affectedDocs = new Map<string, { codePaths: string[]; latestCodeDate: string }>();

  for (const change of codeChanges) {
    for (const [codePrefix, docPath] of Object.entries(options.codeToDocMapping)) {
      if (change.path.startsWith(codePrefix)) {
        const existing = affectedDocs.get(docPath);
        if (!existing) {
          affectedDocs.set(docPath, { codePaths: [change.path], latestCodeDate: change.date });
        } else {
          existing.codePaths.push(change.path);
          if (change.date > existing.latestCodeDate) {
            existing.latestCodeDate = change.date;
          }
        }
      }
    }
  }

  const stale: StaleDoc[] = [];
  for (const [docPath, info] of affectedDocs) {
    const doc = docMap.get(docPath);
    const docDate = doc?.date ?? '1970-01-01';

    if (docDate < info.latestCodeDate) {
      const staleDays = daysBetween(docDate, options.referenceDate);
      if (staleDays >= options.staleDaysThreshold) {
        stale.push({
          docPath,
          relatedCodePaths: info.codePaths,
          docLastModified: docDate,
          codeLastModified: info.latestCodeDate,
          staleDays,
        });
      }
    }
  }

  return stale;
}
