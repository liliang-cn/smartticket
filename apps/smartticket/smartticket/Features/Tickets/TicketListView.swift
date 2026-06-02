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
    var canManage: Bool = false
    var canCreate: Bool = true

    @State private var model = TicketListModel()
    @State private var showCreate = false

    var body: some View {
        NavigationStack {
            ScrollView {
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
                        LazyVStack(spacing: 12) {
                            ForEach(model.tickets) { ticket in
                                NavigationLink(value: ticket) { TicketCard(ticket: ticket) }
                                    .buttonStyle(.plain)
                            }
                        }
                        .padding(20)
                    }
                }
            }
            .background(Color.appBG)
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
            .sheet(isPresented: $showCreate) { CreateTicketView { Task { await model.load() } } }
            .navigationDestination(for: Ticket.self) { ticket in
                TicketDetailView(ticketID: ticket.id, canManage: canManage)
            }
            .overlay { if model.loading && model.tickets.isEmpty { ProgressView() } }
        }
    }

    private var statusMenu: some View {
        Menu {
            Button { model.statusFilter = nil; Task { await model.load() } } label: {
                Label("All statuses", systemImage: model.statusFilter == nil ? "checkmark" : "")
            }
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

struct TicketCard: View {
    let ticket: Ticket
    var body: some View {
        HStack(spacing: 0) {
            // Status accent rail.
            RoundedRectangle(cornerRadius: 3)
                .fill(ticket.status.tint)
                .frame(width: 4)
                .padding(.vertical, 4)

            VStack(alignment: .leading, spacing: 9) {
                HStack(spacing: 8) {
                    if let num = ticket.ticketNumber {
                        Text(num)
                            .font(.caption.monospaced().weight(.medium))
                            .foregroundStyle(Brand.primary)
                    }
                    Spacer()
                    Text(DateText.relative(ticket.createdAt))
                        .font(.caption2).foregroundStyle(.tertiary)
                }
                Text(ticket.title)
                    .font(.body.weight(.semibold))
                    .foregroundStyle(.primary)
                    .lineLimit(2)
                    .multilineTextAlignment(.leading)
                HStack(spacing: 6) {
                    StatusBadge(status: ticket.status)
                    PriorityBadge(priority: ticket.priority)
                    if let c = ticket.customerName, !c.isEmpty {
                        Text(c).font(.caption2).foregroundStyle(.secondary).lineLimit(1)
                    }
                }
            }
            .padding(.leading, 14)
            .padding(.vertical, 4)
        }
        .padding(12)
        .background(Color.card, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .strokeBorder(Color.hairline, lineWidth: 0.5)
        )
        .shadow(color: .black.opacity(0.05), radius: 8, y: 3)
    }
}
