import { useMutation, useQuery } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { KnowledgeArticle } from "@/lib/types";

/** A single ranked hit from semantic search / a RAG citation. */
export interface SearchHit {
  article_id: number;
  title: string;
  snippet: string;
  score: number;
  source_url?: string;
}

export interface AskResult {
  answer: string;
  citations: SearchHit[];
}

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

/** Semantic search over the knowledge base. */
export function useKnowledgeSearch() {
  return useMutation({
    mutationFn: async (vars: {
      query: string;
      topK?: number;
    }): Promise<SearchHit[]> => {
      const res = await api.post("/knowledge/search", {
        query: vars.query,
        top_k: vars.topK,
      });
      return unwrap<{ hits: SearchHit[] }>(res.data).hits ?? [];
    },
  });
}

/** RAG question-answering over the knowledge base. */
export function useKnowledgeAsk() {
  return useMutation({
    mutationFn: async (vars: {
      question: string;
      topK?: number;
    }): Promise<AskResult> => {
      const res = await api.post("/knowledge/ask", {
        question: vars.question,
        top_k: vars.topK,
      });
      return unwrap<AskResult>(res.data);
    },
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
