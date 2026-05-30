import { Badge } from "@/components/ui/badge";

const STATUS_TONE: Record<string, Parameters<typeof Badge>[0]["tone"]> = {
  draft: "slate",
  published: "green",
  archived: "neutral",
};

export function ArticleStatusBadge({ status }: { status: string }) {
  return (
    <Badge tone={STATUS_TONE[status] ?? "neutral"} className="uppercase">
      {status || "unknown"}
    </Badge>
  );
}

/**
 * Parse the backend `tags` field, which is a JSON-encoded array string.
 * Falls back to comma-splitting, then to the raw string, so malformed
 * data never breaks the UI.
 */
export function parseTags(raw: string | undefined | null): string[] {
  if (!raw) return [];
  const trimmed = raw.trim();
  if (!trimmed) return [];
  try {
    const parsed = JSON.parse(trimmed);
    if (Array.isArray(parsed)) {
      return parsed.map((t) => String(t).trim()).filter(Boolean);
    }
  } catch {
    // not JSON — fall through
  }
  return trimmed
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);
}
