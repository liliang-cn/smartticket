import SwiftUI

// MARK: - Model

struct Customer: Codable, Identifiable, Hashable {
    let id: Int
    let name: String
    let code: String?
    let domain: String?
    let isActive: Bool
    let description: String?

    enum CodingKeys: String, CodingKey {
        case id, name, code, domain, description
        case isActive = "is_active"
    }

    var initials: String {
        let parts = name.split(separator: " ")
        let first = parts.first?.first.map(String.init) ?? "?"
        let second = parts.dropFirst().first?.first.map(String.init) ?? ""
        return (first + second).uppercased()
    }
}

@MainActor
@Observable
final class CustomersModel {
    var customers: [Customer] = []
    var loading = false
    var error: String?
    var search = ""

    var filtered: [Customer] {
        guard !search.isEmpty else { return customers }
        let q = search.lowercased()
        return customers.filter {
            $0.name.lowercased().contains(q)
            || ($0.code ?? "").lowercased().contains(q)
            || ($0.domain ?? "").lowercased().contains(q)
        }
    }

    func load() async {
        loading = true; error = nil
        defer { loading = false }
        do {
            let (items, _): ([Customer], PageMeta?) = try await APIClient.shared.getList("/customers", query: ["page_size": "100"])
            customers = items
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }
}

// MARK: - List

struct CustomersView: View {
    @State private var model = CustomersModel()

    var body: some View {
        NavigationStack {
            Group {
                if let error = model.error, model.customers.isEmpty {
                    InlineError(message: error) { Task { await model.load() } }
                } else if model.customers.isEmpty && model.loading {
                    ProgressView().frame(maxWidth: .infinity).padding(.top, 60)
                } else if model.filtered.isEmpty {
                    EmptyStateView(systemImage: "building.2", title: "No customers",
                                   message: "Customer organizations appear here.")
                } else {
                    List(model.filtered) { c in
                        NavigationLink(value: c) { CustomerRow(customer: c) }
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Customers")
            .searchable(text: $model.search, prompt: Text("Search name, code, domain"))
            .navigationDestination(for: Customer.self) { CustomerDetailView(customer: $0) }
            .background(Color.appBG)
            .task { if model.customers.isEmpty { await model.load() } }
            .refreshable { await model.load() }
        }
    }
}

private struct CustomerRow: View {
    let customer: Customer
    var body: some View {
        HStack(spacing: 12) {
            InitialsAvatar(initials: customer.initials, size: 40)
            VStack(alignment: .leading, spacing: 2) {
                Text(customer.name).font(.body.weight(.medium))
                HStack(spacing: 6) {
                    if let code = customer.code, !code.isEmpty {
                        Text(code).font(.caption2).foregroundStyle(.secondary)
                    }
                    if let d = customer.domain, !d.isEmpty {
                        Text(d).font(.caption2).foregroundStyle(.tertiary)
                    }
                }
            }
            Spacer()
            if !customer.isActive {
                Text("Inactive").font(.caption2).foregroundStyle(.secondary)
                    .padding(.horizontal, 7).padding(.vertical, 2)
                    .background(Color.secondary.opacity(0.12), in: Capsule())
            }
        }
        .padding(.vertical, 4)
    }
}

// MARK: - Detail (customer + their tickets)

@MainActor
@Observable
final class CustomerTicketsModel {
    let customerID: Int
    var tickets: [Ticket] = []
    var loading = false

    init(customerID: Int) { self.customerID = customerID }

    func load() async {
        loading = true; defer { loading = false }
        let (items, _): ([Ticket], PageMeta?) = (try? await APIClient.shared.getList(
            "/tickets", query: ["customer_id": String(customerID), "page_size": "50"])) ?? ([], nil)
        tickets = items
    }
}

struct CustomerDetailView: View {
    let customer: Customer
    @State private var model: CustomerTicketsModel

    init(customer: Customer) {
        self.customer = customer
        _model = State(initialValue: CustomerTicketsModel(customerID: customer.id))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // Header
                HStack(spacing: 14) {
                    InitialsAvatar(initials: customer.initials, size: 52)
                    VStack(alignment: .leading, spacing: 3) {
                        Text(customer.name).font(.title3.bold())
                        if let d = customer.domain, !d.isEmpty {
                            Text(d).font(.caption).foregroundStyle(.secondary)
                        }
                    }
                    Spacer()
                }
                if let desc = customer.description, !desc.isEmpty {
                    Text(desc).font(.callout).foregroundStyle(.secondary)
                }

                Text("Tickets").eyebrow().padding(.top, 4)
                if model.loading && model.tickets.isEmpty {
                    ProgressView().frame(maxWidth: .infinity).padding()
                } else if model.tickets.isEmpty {
                    Text("No tickets for this customer.").font(.callout).foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity).padding(.vertical, 12)
                } else {
                    ForEach(model.tickets) { t in
                        NavigationLink(value: t.id) {
                            ticketRow(t)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .padding(20)
        }
        .background(Color.appBG)
        .navigationTitle(customer.code ?? customer.name)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: Int.self) { TicketDetailView(ticketID: $0, canManage: true) }
        .task { await model.load() }
    }

    private func ticketRow(_ t: Ticket) -> some View {
        HStack(spacing: 10) {
            VStack(alignment: .leading, spacing: 4) {
                Text(t.title).font(.subheadline.weight(.medium)).lineLimit(1).foregroundStyle(.primary)
                HStack(spacing: 6) {
                    StatusBadge(status: t.status)
                    PriorityBadge(priority: t.priority)
                }
            }
            Spacer()
            Image(systemName: "chevron.right").font(.caption).foregroundStyle(.tertiary)
        }
        .card(padding: 12)
    }
}
