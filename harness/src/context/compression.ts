export interface CompressionOptions {
  maxLines?: number;
  keepErrorLines?: boolean;
}

export interface CompressionResult {
  compressed: string;
  originalLines: number;
  compressedLines: number;
  ratio: number;
}

const ERROR_PATTERNS = [/error/i, /fail/i, /FAIL/, /warning/i, /panic/i, /fatal/i];

function isErrorLine(line: string): boolean {
  return ERROR_PATTERNS.some((p) => p.test(line));
}

export function compressShellOutput(
  output: string,
  options?: CompressionOptions,
): CompressionResult {
  if (!output) {
    return { compressed: '', originalLines: 0, compressedLines: 0, ratio: 0 };
  }

  const lines = output.split('\n');
  const maxLines = options?.maxLines ?? 50;
  const keepErrors = options?.keepErrorLines ?? true;

  if (lines.length <= maxLines) {
    return {
      compressed: output,
      originalLines: lines.length,
      compressedLines: lines.length,
      ratio: 1,
    };
  }

  const headCount = Math.floor(maxLines * 0.3);
  const tailCount = Math.floor(maxLines * 0.3);

  const head = lines.slice(0, headCount);
  const tail = lines.slice(-tailCount);

  const middle = lines.slice(headCount, lines.length - tailCount);
  const errorLines = keepErrors ? middle.filter(isErrorLine) : [];

  const omitted = middle.length - errorLines.length;
  const compressed = [
    ...head,
    '',
    `... (${omitted} lines omitted) ...`,
    ...errorLines,
    '',
    ...tail,
  ];

  const result = compressed.join('\n');
  return {
    compressed: result,
    originalLines: lines.length,
    compressedLines: compressed.length,
    ratio: compressed.length / lines.length,
  };
}
