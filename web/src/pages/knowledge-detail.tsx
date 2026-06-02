import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Eye } from "lucide-react";
import { useTranslation } from "react-i18next";
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
  const { t } = useTranslation("knowledge");
  const { id } = useParams();
  const articleId = id ? Number(id) : undefined;
  const { data: article, isLoading } = useArticle(articleId);
  const ref = useReveal(article?.id);

  if (isLoading) {
    return (
      <div className="w-full space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (!article) {
    return (
      <div className="w-full py-20 text-center text-muted-foreground">
        {t("detail.not_found")}
        <div className="mt-4">
          <Button variant="secondary" asChild>
            <Link to="/knowledge">
              <ArrowLeft /> {t("detail.back_to_knowledge")}
            </Link>
          </Button>
        </div>
      </div>
    );
  }

  const tags = parseTags(article.tags);

  return (
    <div ref={ref} className="w-full">
      <Link
        to="/knowledge"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="size-4" /> {t("detail.breadcrumb")}
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
              <Label>{t("detail.summary_label")}</Label>
              <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
                {article.summary}
              </p>
            </Card>
          )}

          <Card data-reveal className="p-5">
            <Label>{t("detail.content_label")}</Label>
            <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
              {article.content || t("detail.no_content")}
            </p>
          </Card>
        </div>

        {/* Meta sidebar */}
        <aside data-reveal className="space-y-4">
          <Card className="p-5">
            <MetaRow
              label={t("detail.meta_status")}
              value={<ArticleStatusBadge status={article.status} />}
            />
            <Separator />
            <MetaRow
              label={t("detail.meta_category")}
              value={
                article.category ? (
                  <span className="font-mono text-xs">{article.category}</span>
                ) : (
                  "—"
                )
              }
            />
            <Separator />
            <MetaRow label={t("detail.meta_views")} value={article.view_count} />
            <MetaRow label={t("detail.meta_version")} value={t("detail.version_value", { version: article.version })} />
            <Separator />
            <MetaRow
              label={t("detail.meta_author")}
              value={
                <span className="font-mono text-xs">
                  {article.created_by || "—"}
                </span>
              }
            />
            <MetaRow label={t("detail.meta_created")} value={relativeTime(article.created_at)} />
            <MetaRow label={t("detail.meta_updated")} value={relativeTime(article.updated_at)} />
          </Card>

          {tags.length > 0 && (
            <Card className="p-5">
              <Label>{t("detail.tags_label")}</Label>
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
