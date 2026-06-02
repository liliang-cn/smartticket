import SwiftUI

@MainActor
@Observable
final class TicketListModel {
    var tickets: [Ticket] = []
    var loading = false
    var error: String?
    var search = ""
    var statusFilter: TicketStatus?

    func load() async {
        loading = tickets.isEmpty
        error = nil
        do {
            let (items, _): ([Ticket], PageMeta?) = try await APIClient.shared.getList(
                "/tickets/",
                query: [
                    "page": "1",
                    "page_size": "50",
                    "status": statusFilter?.rawValue,
                    "search": search.isEmpty ? nil : search,
                ]
            )
            tickets = items
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        loading = false
    }
}

/// Shared ticket list used by every portal. The backend scopes results to the
/// signed-in actor, so customers see only their own tickets here.
struct TicketListView: View {
    let title: LocalizedStringKey
    /// Whether the operator controls (status change, assign) are available.
    var canManage: Bool = false
    var canCreate: Bool = true

    @State private var model = TicketListModel()
    @State private var showCreate = false

    var body: some View {
        NavigationStack {
            Group {
                if let error = model.error, model.tickets.isEmpty {
                    InlineError(message: error) { Task { await model.load() } }
                } else if model.tickets.isEmpty && !model.loading {
                    EmptyStateView(
                        systemImage: "ticket",
                        title: "No tickets",
                        message: model.search.isEmpty ? "Tickets will appear here." : "No tickets match your search."
                    )
                } else {
                    list
                }
            }
            .navigationTitle(title)
            .searchable(text: $model.search, prompt: "Search tickets")
            .onSubmit(of: .search) { Task { await model.load() } }
            .toolbar {
                ToolbarItem(placement: .topBarLeading) { statusMenu }
                if canCreate {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button { showCreate = true } label: { Image(systemName: "square.and.pencil") }
                    }
                }
            }
            .refreshable { await model.load() }
            .task { if model.tickets.isEmpty { await model.load() } }
            .sheet(isPresented: $showCreate) {
                CreateTicketView { Task { await model.load() } }
            }
            .navigationDestination(for: Ticket.self) { ticket in
                TicketDetailView(ticketID: ticket.id, canManage: canManage)
            }
            .overlay { if model.loading && model.tickets.isEmpty { ProgressView() } }
        }
    }

    private var list: some View {
        List(model.tickets) { ticket in
            NavigationLink(value: ticket) { TicketRow(ticket: ticket) }
        }
        .listStyle(.plain)
    }

    private var statusMenu: some View {
        Menu {
            Button("All statuses") { model.statusFilter = nil; Task { await model.load() } }
            Divider()
            ForEach(TicketStatus.allCases, id: \.self) { s in
                Button { model.statusFilter = s; Task { await model.load() } } label: {
                    Label(s.label, systemImage: model.statusFilter == s ? "checkmark" : "")
                }
            }
        } label: {
            Image(systemName: model.statusFilter == nil ? "line.3.horizontal.decrease.circle" : "line.3.horizontal.decrease.circle.fill")
        }
    }
}

struct TicketRow: View {
    let ticket: Ticket
    var body: some View {
        VStack(alignment: .leading, spacing: 7) {
            HStack(spacing: 8) {
                if let num = ticket.ticketNumber {
                    Text(num).font(.caption.monospaced()).foregroundStyle(.secondary)
                }
                Spacer()
                Text(DateText.relative(ticket.createdAt))
                    .font(.caption2).foregroundStyle(.tertiary)
            }
            Text(ticket.title).font(.body.weight(.medium)).lineLimit(2)
            HStack(spacing: 6) {
                StatusBadge(status: ticket.status)
                PriorityBadge(priority: ticket.priority)
                if let c = ticket.customerName, !c.isEmpty {
                    Text(c).font(.caption2).foregroundStyle(.secondary).lineLimit(1)
                }
            }
        }
        .padding(.vertical, 4)
    }
}
