import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Eye } from "lucide-react";
import { useArticle } from "@/features/knowledge/api";
import {
  ArticleStatusBadge,
  parseTags,
} from "@/features/knowledge/status-badge";
import { relativeTime } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton, Separator } from "@/components/ui/misc";
import { useReveal } from "@/lib/use-reveal";

function MetaRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 py-2 text-sm">
      <span className="font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      <span className="text-right">{value}</span>
    </div>
  );
}

export function KnowledgeDetailPage() {
  const { id } = useParams();
  const articleId = id ? Number(id) : undefined;
  const { data: article, isLoading } = useArticle(articleId);
  const ref = useReveal(article?.id);

  if (isLoading) {
    return (
      <div className="mx-auto max-w-5xl space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (!article) {
    return (
      <div className="mx-auto max-w-5xl py-20 text-center text-muted-foreground">
        Article not found.
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/knowledge">
              <ArrowLeft /> Back to knowledge
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  const tags = parseTags(article.tags);

  return (
    <div ref={ref} className="mx-auto max-w-5xl">
      <Link
        to="/knowledge"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> Knowledge
      </Link>

      <div data-reveal className="mb-6">
        <div className="flex flex-wrap items-center gap-2">
          <ArticleStatusBadge status={article.status} />
          {article.category && (
            <span className="font-mono text-xs text-primary/80">
              {article.category}
            </span>
          )}
          <span className="inline-flex items-center gap-1 font-mono text-xs text-muted-foreground">
            <Eye className="size-3.5" /> {article.view_count}
          </span>
        </div>
        <h1 className="mt-2 text-2xl">{article.title}</h1>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_18rem]">
        {/* Main */}
        <div className="space-y-5">
          {article.summary && (
            <Card data-reveal className="p-5">
              <Label>Summary</Label>
              <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
                {article.summary}
              </p>
            </Card>
          )}

          <Card data-reveal className="p-5">
            <Label>Content</Label>
            <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
              {article.content || "No content."}
            </p>
          </Card>
        </div>

        {/* Meta sidebar */}
        <aside data-reveal className="space-y-4">
          <Card className="p-5">
            <MetaRow
              label="Status"
              value={<ArticleStatusBadge status={article.status} />}
            />
            <Separator />
            <MetaRow
              label="Category"
              value={
                article.category ? (
                  <span className="font-mono text-xs">{article.category}</span>
                ) : (
                  "—"
                )
              }
            />
            <Separator />
            <MetaRow label="Views" value={article.view_count} />
            <MetaRow label="Version" value={`v${article.version}`} />
            <Separator />
            <MetaRow
              label="Author"
              value={
                <span className="font-mono text-xs">
                  {article.created_by || "—"}
                </span>
              }
            />
            <MetaRow label="Created" value={relativeTime(article.created_at)} />
            <MetaRow label="Updated" value={relativeTime(article.updated_at)} />
          </Card>

          {tags.length > 0 && (
            <Card className="p-5">
              <Label>Tags</Label>
              <div className="mt-3 flex flex-wrap gap-1.5">
                {tags.map((t) => (
                  <Badge key={t} tone="neutral">
                    {t}
                  </Badge>
                ))}
              </div>
            </Card>
          )}
        </aside>
      </div>
    </div>
  );
}
