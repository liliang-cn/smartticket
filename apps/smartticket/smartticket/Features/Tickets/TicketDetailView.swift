import SwiftUI

@MainActor
@Observable
final class TicketDetailModel {
    let ticketID: Int
    var ticket: Ticket?
    var messages: [TicketMessage] = []
    var loading = true
    var error: String?
    var sending = false

    init(ticketID: Int) { self.ticketID = ticketID }

    func load() async {
        error = nil
        do {
            async let t: Ticket = APIClient.shared.get("/tickets/\(ticketID)")
            async let m: [TicketMessage] = fetchMessages()
            ticket = try await t
            messages = try await m
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        loading = false
    }

    private func fetchMessages() async throws -> [TicketMessage] {
        let (items, _): ([TicketMessage], PageMeta?) = try await APIClient.shared.getList("/tickets/\(ticketID)/messages")
        return items
    }

    func reply(_ text: String, internalNote: Bool) async {
        sending = true
        defer { sending = false }
        struct Body: Encodable { let content: String; let is_internal: Bool }
        do {
            let _: TicketMessage = try await APIClient.shared.post("/tickets/\(ticketID)/messages", body: Body(content: text, is_internal: internalNote))
            messages = try await fetchMessages()
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }

    func setStatus(_ status: TicketStatus) async {
        struct Body: Encodable { let status: String }
        do {
            let updated: Ticket = try await APIClient.shared.put("/tickets/\(ticketID)", body: Body(status: status.rawValue))
            ticket = updated
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }

    func assignToMe(userID: Int) async {
        struct Body: Encodable { let assigned_to: Int }
        do {
            let _: Ticket = try await APIClient.shared.post("/tickets/\(ticketID)/assign", body: Body(assigned_to: userID))
            await load()
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }
}

struct TicketDetailView: View {
    let ticketID: Int
    var canManage: Bool

    @Environment(AuthStore.self) private var auth
    @State private var model: TicketDetailModel
    @State private var reply = ""
    @State private var internalNote = false

    init(ticketID: Int, canManage: Bool) {
        self.ticketID = ticketID
        self.canManage = canManage
        _model = State(initialValue: TicketDetailModel(ticketID: ticketID))
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                if let t = model.ticket {
                    header(t)
                    if canManage { manageBar(t) }
                    Divider()
                    conversation
                } else if let error = model.error {
                    InlineError(message: error) { Task { await model.load() } }
                } else if model.loading {
                    ProgressView().frame(maxWidth: .infinity).padding(.top, 40)
                }
            }
            .padding()
        }
        .navigationTitle(model.ticket?.ticketNumber ?? "Ticket")
        .navigationBarTitleDisplayMode(.inline)
        .safeAreaInset(edge: .bottom) { if model.ticket != nil { composer } }
        .task { await model.load() }
    }

    private func header(_ t: Ticket) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(t.title).font(.title2.bold())
            HStack(spacing: 6) {
                StatusBadge(status: t.status)
                PriorityBadge(priority: t.priority)
            }
            if let d = t.description, !d.isEmpty {
                Text(d).font(.callout).foregroundStyle(.secondary)
            }
            metaGrid(t)
        }
    }

    private func metaGrid(_ t: Ticket) -> some View {
        VStack(spacing: 6) {
            metaRow("Requester", t.requesterName ?? t.requesterEmail ?? "—")
            if let c = t.customerName, !c.isEmpty { metaRow("Customer", c) }
            metaRow("Assignee", t.assignedUser?.displayName ?? String(localized: "Unassigned"))
            metaRow("Created", DateText.medium(t.createdAt))
            if let sla = t.slaStatus, !sla.isEmpty { metaRow("SLA", sla.capitalized) }
        }
        .padding(12)
        .background(.background.secondary, in: RoundedRectangle(cornerRadius: 12))
    }

    private func metaRow(_ k: LocalizedStringKey, _ v: String) -> some View {
        HStack {
            Text(k).font(.caption).foregroundStyle(.secondary)
            Spacer()
            Text(v).font(.caption.weight(.medium)).multilineTextAlignment(.trailing)
        }
    }

    private func manageBar(_ t: Ticket) -> some View {
        HStack {
            Menu {
                ForEach(TicketStatus.allCases, id: \.self) { s in
                    Button(s.label) { Task { await model.setStatus(s) } }
                }
            } label: {
                Label("Set status", systemImage: "arrow.triangle.2.circlepath")
            }
            Spacer()
            if t.assignedTo == nil, let me = auth.user {
                Button {
                    Task { await model.assignToMe(userID: me.id) }
                } label: {
                    Label("Assign to me", systemImage: "person.fill.badge.plus")
                }
            }
        }
        .font(.callout)
        .buttonStyle(.bordered)
    }

    private var conversation: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Conversation").font(.headline)
            if model.messages.isEmpty {
                Text("No replies yet.").font(.callout).foregroundStyle(.secondary)
            } else {
                ForEach(model.messages) { MessageBubble(message: $0) }
            }
        }
    }

    private var composer: some View {
        VStack(spacing: 8) {
            if canManage {
                Toggle("Internal note (hidden from customer)", isOn: $internalNote)
                    .font(.caption)
            }
            HStack(alignment: .bottom, spacing: 8) {
                TextField("Write a reply…", text: $reply, axis: .vertical)
                    .textFieldStyle(.roundedBorder)
                    .lineLimit(1...4)
                Button {
                    let text = reply.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard !text.isEmpty else { return }
                    reply = ""
                    Task { await model.reply(text, internalNote: internalNote) }
                } label: {
                    Image(systemName: "arrow.up.circle.fill").font(.title)
                }
                .disabled(model.sending || reply.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(12)
        .background(.bar)
    }
}

struct MessageBubble: View {
    let message: TicketMessage
    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack(spacing: 6) {
                Text(message.authorName ?? "User").font(.caption.weight(.semibold))
                if message.isFromAI {
                    Text("AI").font(.caption2).padding(.horizontal, 5).padding(.vertical, 1)
                        .background(Brand.info.opacity(0.2), in: Capsule()).foregroundStyle(Brand.info)
                }
                if message.isInternal {
                    Text("Internal").font(.caption2).padding(.horizontal, 5).padding(.vertical, 1)
                        .background(Brand.primary.opacity(0.2), in: Capsule()).foregroundStyle(Brand.primary)
                }
                Spacer()
                Text(DateText.relative(message.createdAt)).font(.caption2).foregroundStyle(.tertiary)
            }
            Text(message.content).font(.callout)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(
            message.isInternal ? Brand.primary.opacity(0.06) : Color(.secondarySystemBackground),
            in: RoundedRectangle(cornerRadius: 12)
        )
    }
}
