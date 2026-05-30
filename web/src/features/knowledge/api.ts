import { useQuery } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { KnowledgeArticle } from "@/lib/types";

export interface ArticleFilters {
  page: number;
  page_size: number;
  search?: string;
  category?: string;
  status?: string;
}

export interface ArticlePage {
  items: KnowledgeArticle[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export function useArticles(filters: ArticleFilters) {
  return useQuery({
    queryKey: ["articles", filters],
    queryFn: async (): Promise<ArticlePage> => {
      const res = await api.get("/knowledge/articles", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          search: filters.search || undefined,
          category: filters.category || undefined,
          status: filters.status || undefined,
        },
      });
      // The list endpoint returns { success, data: KnowledgeArticle[], meta: {...} }.
      const body = res.data as {
        data?: KnowledgeArticle[];
        meta?: {
          total?: number;
          page?: number;
          page_size?: number;
          total_pages?: number;
        };
      };
      const meta = body.meta ?? {};
      return {
        items: body.data ?? [],
        total: meta.total ?? 0,
        page: meta.page ?? filters.page,
        page_size: meta.page_size ?? filters.page_size,
        total_pages: meta.total_pages ?? 1,
      };
    },
    placeholderData: (prev) => prev,
  });
}

export function useArticle(id: number | undefined) {
  return useQuery({
    queryKey: ["article", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/knowledge/articles/${id}`);
      return unwrap<KnowledgeArticle>(res.data);
    },
  });
}
