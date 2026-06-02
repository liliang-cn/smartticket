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
            ScrollView {
                Group {
                    if let error = model.error, model.articles.isEmpty {
                        InlineError(message: error) { Task { await model.load() } }
                    } else if model.articles.isEmpty && !model.loading {
                        EmptyStateView(systemImage: "book", title: "No articles",
                                       message: "Knowledge base articles will appear here.")
                    } else {
                        LazyVStack(spacing: 12) {
                            ForEach(model.articles) { article in
                                NavigationLink(value: article) { ArticleCard(article: article) }
                                    .buttonStyle(.plain)
                            }
                        }
                        .padding(20)
                    }
                }
            }
            .background(Color.appBG)
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

struct ArticleCard: View {
    let article: KnowledgeArticle
    var body: some View {
        HStack(spacing: 14) {
            Image(systemName: "doc.text.fill")
                .font(.system(size: 17))
                .foregroundStyle(Brand.primary)
                .frame(width: 42, height: 42)
                .background(Brand.wash(Brand.primary), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            VStack(alignment: .leading, spacing: 4) {
                Text(article.title).font(.body.weight(.semibold)).lineLimit(2)
                    .foregroundStyle(.primary)
                if let s = article.summary, !s.isEmpty {
                    Text(s).font(.caption).foregroundStyle(.secondary).lineLimit(2)
                }
                if let cat = article.category, !cat.isEmpty {
                    Text(cat).font(.caption2.weight(.semibold)).foregroundStyle(Brand.primary)
                        .padding(.horizontal, 7).padding(.vertical, 2)
                        .background(Brand.wash(Brand.primary), in: Capsule())
                }
            }
            Spacer(minLength: 0)
            Image(systemName: "chevron.right").font(.caption.weight(.semibold)).foregroundStyle(.tertiary)
        }
        .card()
    }
}

struct ArticleDetailView: View {
    let article: KnowledgeArticle
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text(article.title).font(.title.bold())
                HStack(spacing: 10) {
                    if let cat = article.category, !cat.isEmpty {
                        Text(cat).font(.caption2.weight(.semibold)).foregroundStyle(Brand.primary)
                            .padding(.horizontal, 8).padding(.vertical, 3)
                            .background(Brand.wash(Brand.primary), in: Capsule())
                    }
                    if let v = article.version { Text("v\(v)").font(.caption2).foregroundStyle(.secondary) }
                    Text(DateText.relative(article.updatedAt)).font(.caption2).foregroundStyle(.tertiary)
                }
                Divider()
                Text(article.content ?? article.summary ?? "")
                    .font(.body)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(20)
        }
        .background(Color.appBG)
        .navigationTitle("Article")
        .navigationBarTitleDisplayMode(.inline)
    }
}
