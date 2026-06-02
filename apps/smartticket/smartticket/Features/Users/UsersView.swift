import SwiftUI

@MainActor
@Observable
final class UsersModel {
    var users: [UserInfo] = []
    var loading = false
    var error: String?
    var search = ""

    func load() async {
        loading = users.isEmpty
        error = nil
        do {
            let (items, _): ([UserInfo], PageMeta?) = try await APIClient.shared.getList(
                "/users",
                query: ["page": "1", "page_size": "100", "search": search.isEmpty ? nil : search]
            )
            users = items
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        loading = false
    }
}

/// Admin-only directory of team members and customer users.
struct UsersView: View {
    @State private var model = UsersModel()

    var body: some View {
        NavigationStack {
            Group {
                if let error = model.error, model.users.isEmpty {
                    InlineError(message: error) { Task { await model.load() } }
                } else if model.users.isEmpty && !model.loading {
                    EmptyStateView(systemImage: "person.2", title: "No users")
                } else {
                    List(model.users) { user in
                        HStack(spacing: 12) {
                            InitialsAvatar(initials: user.initials)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(user.displayName).font(.body.weight(.medium))
                                Text(user.email).font(.caption).foregroundStyle(.secondary).lineLimit(1)
                            }
                            Spacer()
                            VStack(alignment: .trailing, spacing: 3) {
                                Text(user.role.displayName)
                                    .font(.caption2.weight(.semibold)).textCase(.uppercase)
                                    .foregroundStyle(Brand.primary)
                                if !user.isActive {
                                    Text("Inactive").font(.caption2).textCase(.uppercase)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                        .padding(.vertical, 2)
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Users")
            .searchable(text: $model.search, prompt: "Search users")
            .onSubmit(of: .search) { Task { await model.load() } }
            .refreshable { await model.load() }
            .task { if model.users.isEmpty { await model.load() } }
            .overlay { if model.loading && model.users.isEmpty { ProgressView() } }
        }
    }
}
