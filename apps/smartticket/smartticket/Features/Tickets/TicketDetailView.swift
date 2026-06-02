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
            VStack(alignment: .leading, spacing: 16) {
                if let t = model.ticket {
                    header(t)
                    if canManage { manageBar(t) }
                    conversation
                } else if let error = model.error {
                    InlineError(message: error) { Task { await model.load() } }
                } else if model.loading {
                    ProgressView().frame(maxWidth: .infinity).padding(.top, 60)
                }
            }
            .padding(20)
        }
        .background(Color.appBG)
        .navigationTitle(model.ticket?.ticketNumber ?? "Ticket")
        .navigationBarTitleDisplayMode(.inline)
        .safeAreaInset(edge: .bottom) { if model.ticket != nil { composer } }
        .task { await model.load() }
    }

    private func header(_ t: Ticket) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(t.title).font(.title2.bold())
            HStack(spacing: 6) {
                StatusBadge(status: t.status)
                PriorityBadge(priority: t.priority)
            }
            if let d = t.description, !d.isEmpty {
                Text(d).font(.callout).foregroundStyle(.secondary)
            }
            Divider()
            VStack(spacing: 10) {
                metaRow("person", "Requester", t.requesterName ?? t.requesterEmail ?? "—")
                if let c = t.customerName, !c.isEmpty { metaRow("building.2", "Customer", c) }
                metaRow("person.badge.shield.checkmark", "Assignee", t.assignedUser?.displayName ?? String(localized: "Unassigned"))
                metaRow("calendar", "Created", DateText.medium(t.createdAt))
                if let sla = t.slaStatus, !sla.isEmpty { metaRow("timer", "SLA", sla.capitalized) }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .card(padding: 18)
    }

    private func metaRow(_ icon: String, _ k: LocalizedStringKey, _ v: String) -> some View {
        HStack(spacing: 10) {
            Image(systemName: icon).font(.caption).foregroundStyle(Brand.primary).frame(width: 18)
            Text(k).font(.caption).foregroundStyle(.secondary)
            Spacer()
            Text(v).font(.caption.weight(.medium)).multilineTextAlignment(.trailing)
        }
    }

    private func manageBar(_ t: Ticket) -> some View {
        HStack(spacing: 10) {
            Menu {
                ForEach(TicketStatus.allCases, id: \.self) { s in
                    Button { Task { await model.setStatus(s) } } label: { Label(s.label, systemImage: s.icon) }
                }
            } label: {
                Label("Set status", systemImage: "arrow.triangle.2.circlepath")
                    .font(.subheadline.weight(.medium))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(Color.card, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .overlay(RoundedRectangle(cornerRadius: 12, style: .continuous).strokeBorder(Color.hairline, lineWidth: 0.5))
            }
            if t.assignedTo == nil, let me = auth.user {
                Button { Task { await model.assignToMe(userID: me.id) } } label: {
                    Label("Assign to me", systemImage: "person.fill.badge.plus")
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .foregroundStyle(Brand.ink)
                        .background(Brand.gradient, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
                }
            }
        }
        .tint(.primary)
    }

    private var conversation: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Conversation").eyebrow().padding(.top, 4)
            if model.messages.isEmpty {
                Text("No replies yet.").font(.callout).foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity).padding(.vertical, 12)
            } else {
                ForEach(model.messages) { MessageBubble(message: $0) }
            }
        }
    }

    private var composer: some View {
        VStack(spacing: 10) {
            if canManage {
                Toggle(isOn: $internalNote) {
                    Label("Internal note (hidden from customer)", systemImage: "lock")
                        .font(.caption)
                }
                .tint(Brand.primary)
            }
            HStack(alignment: .bottom, spacing: 10) {
                TextField("Write a reply…", text: $reply, axis: .vertical)
                    .lineLimit(1...5)
                    .padding(.horizontal, 14).padding(.vertical, 10)
                    .background(Color(.tertiarySystemBackground), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
                    .overlay(RoundedRectangle(cornerRadius: 20, style: .continuous).strokeBorder(Color.hairline, lineWidth: 0.5))
                Button {
                    let text = reply.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard !text.isEmpty else { return }
                    reply = ""
                    Task { await model.reply(text, internalNote: internalNote) }
                } label: {
                    Image(systemName: "arrow.up")
                        .font(.system(size: 18, weight: .bold))
                        .foregroundStyle(Brand.ink)
                        .frame(width: 40, height: 40)
                        .background(canSend ? AnyShapeStyle(Brand.gradient) : AnyShapeStyle(Color.secondary.opacity(0.3)), in: Circle())
                }
                .disabled(!canSend)
            }
        }
        .padding(14)
        .background(.bar)
    }

    private var canSend: Bool {
        !model.sending && !reply.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }
}

struct MessageBubble: View {
    let message: TicketMessage

    private var isCustomer: Bool { (message.authorRole ?? "").lowercased() == "customer" }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Text(message.authorName ?? "User").font(.caption.weight(.semibold))
                if message.isFromAI { tag("AI", Brand.info) }
                if message.isInternal { tag("Internal", Brand.primary) }
                Spacer()
                Text(DateText.relative(message.createdAt)).font(.caption2).foregroundStyle(.tertiary)
            }
            Text(message.content).font(.callout).foregroundStyle(.primary)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(14)
        .background(bubbleFill, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .strokeBorder(borderColor, lineWidth: 0.5)
        )
    }

    private var bubbleFill: Color {
        if message.isInternal { return Brand.primary.opacity(0.08) }
        return isCustomer ? Color.card : Brand.info.opacity(0.06)
    }
    private var borderColor: Color {
        message.isInternal ? Brand.primary.opacity(0.2) : Color.hairline
    }

    private func tag(_ text: String, _ color: Color) -> some View {
        Text(text)
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 6).padding(.vertical, 1)
            .background(color.opacity(0.18), in: Capsule())
            .foregroundStyle(color)
    }
}
