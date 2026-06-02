import SwiftUI

@MainActor
@Observable
final class KnowledgeModel {
    var articles: [KnowledgeArticle] = []
    var loading = false
    var error: String?
    var search = ""

    func load() async {
        loading = articles.isEmpty
        error = nil
        do {
            let (items, _): ([KnowledgeArticle], PageMeta?) = try await APIClient.shared.getList(
                "/knowledge/articles",
                query: ["page": "1", "page_size": "50", "search": search.isEmpty ? nil : search]
            )
            articles = items
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        loading = false
    }
}

struct KnowledgeView: View {
    @State private var model = KnowledgeModel()

    var body: some View {
        NavigationStack {
            Group {
                if let error = model.error, model.articles.isEmpty {
                    InlineError(message: error) { Task { await model.load() } }
                } else if model.articles.isEmpty && !model.loading {
                    EmptyStateView(systemImage: "book", title: "No articles",
                                   message: "Knowledge base articles will appear here.")
                } else {
                    List(model.articles) { article in
                        NavigationLink(value: article) {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(article.title).font(.body.weight(.medium)).lineLimit(2)
                                if let s = article.summary, !s.isEmpty {
                                    Text(s).font(.caption).foregroundStyle(.secondary).lineLimit(2)
                                }
                                if let cat = article.category, !cat.isEmpty {
                                    Text(cat.uppercased()).font(.caption2).foregroundStyle(Brand.primary)
                                }
                            }
                            .padding(.vertical, 2)
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Knowledge")
            .searchable(text: $model.search, prompt: "Search articles")
            .onSubmit(of: .search) { Task { await model.load() } }
            .refreshable { await model.load() }
            .task { if model.articles.isEmpty { await model.load() } }
            .navigationDestination(for: KnowledgeArticle.self) { ArticleDetailView(article: $0) }
            .overlay { if model.loading && model.articles.isEmpty { ProgressView() } }
        }
    }
}

struct ArticleDetailView: View {
    let article: KnowledgeArticle
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                Text(article.title).font(.title2.bold())
                HStack(spacing: 10) {
                    if let cat = article.category, !cat.isEmpty {
                        Text(cat.uppercased()).font(.caption2).foregroundStyle(Brand.primary)
                    }
                    if let v = article.version { Text("v\(v)").font(.caption2).foregroundStyle(.secondary) }
                    Text(DateText.relative(article.updatedAt)).font(.caption2).foregroundStyle(.tertiary)
                }
                Divider()
                Text(article.content ?? article.summary ?? "")
                    .font(.body)
            }
            .padding()
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .navigationTitle("Article")
        .navigationBarTitleDisplayMode(.inline)
    }
}
