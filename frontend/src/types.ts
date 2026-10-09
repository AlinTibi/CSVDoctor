export interface Row {
  cells: string[];
  line: number;
  blank: boolean;
}
export interface Issue {
  code: string;
  severity: string;
  message: string;
  row: number;
  column: number;
}
export interface ParseOptions {
  encoding: string;
  delimiter: string;
  header: boolean;
  quoteMode: string;
}
export interface ExportOptions {
  encoding: string;
  delimiter: string;
  lineEnding: string;
}
export interface Repairs {
  headers: boolean;
  padRows: boolean;
  keepUneven: boolean;
  formulaProtection: boolean;
}
export interface Document {
  name: string;
  bytes: number;
  sha256: string;
  encoding: string;
  encodingConfirmed: boolean;
  delimiter: string;
  delimiterConfirmed: boolean;
  candidates: { delimiter: string; columns: number; consistency: number }[];
  lineEnding: string;
  parsed: boolean;
  header: boolean;
  quoteMode: string;
  rowCount: number;
  columns: number;
  maxColumns: number;
  counts: Record<string, number>;
  issues: Issue[];
  preview: Row[];
  rawPreview: string;
}
export interface Snapshot {
  revision: string;
  document: Document | null;
}
export interface Plan {
  canExport: boolean;
  blockers: string[];
  changes: { kind: string; count: number; description: string }[];
  before: Row[];
  after: Row[];
  report: {
    remaining: Record<string, number>;
    inputRows: number;
    outputRows: number;
  };
}
export interface ReviewResult {
  token: string;
  plan: Plan;
}
declare global {
  interface Window {
    go?: {
      main: {
        App: {
          InitialDocument(): Promise<Snapshot>;
          OpenFile(o: ParseOptions): Promise<Snapshot | null>;
          LoadFile(path: string, o: ParseOptions): Promise<Snapshot>;
          Diagnose(o: ParseOptions): Promise<Snapshot>;
          Review(
            revision: string,
            r: Repairs,
            o: ExportOptions,
          ): Promise<ReviewResult>;
          Export(
            token: string,
          ): Promise<{
            cancelled: boolean;
            filename: string;
            reportFilename: string;
            remaining: Record<string, number>;
          }>;
        };
      };
    };
    runtime?: {
      OnFileDrop(
        callback: (x: number, y: number, paths: string[]) => void,
        useDropTarget: boolean,
      ): void;
    };
  }
}
