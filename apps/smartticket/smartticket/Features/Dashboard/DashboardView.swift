import SwiftUI

@MainActor
@Observable
final class DashboardModel {
    var stats: TicketStats?
    var loading = true
    var error: String?

    func load() async {
        error = nil
        do {
            stats = try await APIClient.shared.get("/tickets/stats")
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        loading = false
    }
}

struct DashboardView: View {
    @Environment(AuthStore.self) private var auth
    @State private var model = DashboardModel()

    private let columns = [GridItem(.flexible()), GridItem(.flexible())]

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    if let name = auth.user?.displayName {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Overview").font(.caption2.weight(.semibold))
                                .textCase(.uppercase).foregroundStyle(.secondary)
                            Text("Hello, \(name).").font(.title.bold())
                        }
                    }
                    if let error = model.error {
                        InlineError(message: error) { Task { await model.load() } }
                    } else if let s = model.stats {
                        LazyVGrid(columns: columns, spacing: 12) {
                            StatCard(value: s.openTickets, label: "Open", tint: Brand.primary)
                            StatCard(value: s.inProgressTickets, label: "In progress", tint: Brand.info)
                            StatCard(value: s.resolvedTickets, label: "Resolved", tint: Brand.success)
                            StatCard(value: s.overdueTickets, label: "Overdue", tint: Brand.danger)
                        }
                        StatCard(value: s.totalTickets, label: "Total tickets")
                    } else if model.loading {
                        ProgressView().frame(maxWidth: .infinity).padding(.top, 40)
                    }
                }
                .padding()
            }
            .navigationTitle("Dashboard")
            .refreshable { await model.load() }
            .task { await model.load() }
        }
    }
}
