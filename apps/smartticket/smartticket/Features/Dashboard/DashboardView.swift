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

    private let columns = [GridItem(.flexible(), spacing: 12), GridItem(.flexible(), spacing: 12)]

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 22) {
                    greeting

                    if let error = model.error {
                        InlineError(message: error) { Task { await model.load() } }
                    } else if let s = model.stats {
                        LazyVGrid(columns: columns, spacing: 12) {
                            StatCard(value: s.openTickets, label: "Open", systemImage: "tray", tint: Brand.primary)
                            StatCard(value: s.inProgressTickets, label: "In progress", systemImage: "arrow.triangle.2.circlepath", tint: Brand.info)
                            StatCard(value: s.resolvedTickets, label: "Resolved", systemImage: "checkmark.circle", tint: Brand.success)
                            StatCard(value: s.overdueTickets, label: "Overdue", systemImage: "exclamationmark.triangle", tint: Brand.danger)
                        }
                        totalCard(s)
                    } else if model.loading {
                        ProgressView().frame(maxWidth: .infinity).padding(.top, 60)
                    }
                }
                .padding(20)
            }
            .background(Color.appBG)
            .navigationTitle("Dashboard")
            .refreshable { await model.load() }
            .task { await model.load() }
        }
    }

    private var greeting: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Overview").eyebrow()
            Text("Hello, \(auth.user?.displayName ?? "there").")
                .font(.system(.largeTitle, design: .rounded).weight(.bold))
        }
    }

    private func totalCard(_ s: TicketStats) -> some View {
        HStack(spacing: 16) {
            Image(systemName: "tray.full.fill")
                .font(.title2)
                .foregroundStyle(Brand.ink)
                .frame(width: 48, height: 48)
                .background(Brand.gradient, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            VStack(alignment: .leading, spacing: 2) {
                Text("\(s.totalTickets) total tickets")
                    .font(.headline)
                Text("Across all statuses")
                    .font(.subheadline).foregroundStyle(.secondary)
            }
            Spacer()
            Image(systemName: "chevron.right").font(.footnote.weight(.semibold)).foregroundStyle(.tertiary)
        }
        .card()
    }
}
