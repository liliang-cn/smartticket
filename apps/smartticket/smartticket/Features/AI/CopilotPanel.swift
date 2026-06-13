import SwiftUI

// MARK: - Model

@MainActor
@Observable
final class CopilotModel {
    let ticketID: Int
    var suggestions: [AISuggestion] = []
    var running: Set<String> = []
    var loaded = false

    init(ticketID: Int) { self.ticketID = ticketID }

    private struct Ack: Decodable { let adopted: Bool?; let dismissed: Bool? }
    private struct EmptyBody: Encodable {}

    /// Suggestions sorted by the agent's display order.
    var ordered: [AISuggestion] {
        suggestions
            .filter { !$0.isDismissed }
            .sorted { $0.agent.order < $1.agent.order }
    }

    func suggestion(for agent: AIAgent) -> AISuggestion? {
        suggestions.first { $0.agent == agent && !$0.isDismissed }
    }

    func load() async {
        if let resp: SuggestionsResponse = try? await APIClient.shared.getBare("/tickets/\(ticketID)/ai/suggestions") {
            suggestions = resp.suggestions
        }
        loaded = true
    }

    /// Run an on-demand agent. Returns immediately (server is async); we then poll.
    func run(_ agent: AIAgent, draft: String = "") async {
        running.insert(agent.rawValue)
        defer { running.remove(agent.rawValue) }
        let path = "/tickets/\(ticketID)/ai/\(endpoint(for: agent))"
        struct ReviewBody: Encodable { let draft: String }
        _ = try? await APIClient.shared.postBare(path, body: ReviewBody(draft: draft)) as AISuggestion?
        await pollUntilSettled()
    }

    private func endpoint(for agent: AIAgent) -> String {
        switch agent {
        case .researcher: return "research"
        case .reviewer: return "review"
        case .drafter: return "draft"
        default: return "research"
        }
    }

    /// Poll a few times while any suggestion is still pending.
    func pollUntilSettled() async {
        for _ in 0..<12 {
            try? await Task.sleep(for: .seconds(3))
            await load()
            if !suggestions.contains(where: { $0.isPending }) { return }
        }
    }

    func adopt(_ s: AISuggestion) async {
        _ = try? await APIClient.shared.postBare("/tickets/\(ticketID)/ai/suggestions/\(s.id)/adopt", body: EmptyBody()) as Ack?
        await load()
    }

    func dismiss(_ s: AISuggestion) async {
        _ = try? await APIClient.shared.postBare("/tickets/\(ticketID)/ai/suggestions/\(s.id)/dismiss", body: EmptyBody()) as Ack?
        await load()
    }
}

// MARK: - Panel

struct CopilotPanel: View {
    let ticketID: Int
    /// Inserts text into the parent's reply composer (Draft / Researcher / Reviewer).
    var onInsert: (String) -> Void

    @State private var model: CopilotModel
    @State private var currentDraft: String

    init(ticketID: Int, currentDraft: String = "", onInsert: @escaping (String) -> Void) {
        self.ticketID = ticketID
        self.onInsert = onInsert
        _model = State(initialValue: CopilotModel(ticketID: ticketID))
        _currentDraft = State(initialValue: currentDraft)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Label("AI Copilot", systemImage: "sparkles")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(Brand.primary)
                Spacer()
            }

            // On-demand run chips for agents without a suggestion yet.
            let missing = [AIAgent.researcher, .drafter, .reviewer].filter { model.suggestion(for: $0) == nil }
            if !missing.isEmpty {
                HStack(spacing: 8) {
                    ForEach(missing, id: \.self) { agent in
                        runChip(agent)
                    }
                }
            }

            ForEach(model.ordered) { s in
                SuggestionCard(suggestion: s,
                               running: model.running.contains(s.agentName),
                               onInsert: onInsert,
                               onRerun: { Task { await model.run(s.agent, draft: currentDraft) } },
                               onAdopt: { Task { await model.adopt(s) } },
                               onDismiss: { Task { await model.dismiss(s) } })
            }

            if model.loaded && model.ordered.isEmpty && missing.isEmpty {
                Text("No AI suggestions yet.")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .padding(14)
        .background(Brand.wash(Brand.primary).opacity(0.5), in: RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).strokeBorder(Brand.primary.opacity(0.12), lineWidth: 0.5))
        .task { if !model.loaded { await model.load() } }
    }

    private func runChip(_ agent: AIAgent) -> some View {
        Button {
            Task { await model.run(agent, draft: currentDraft) }
        } label: {
            Label(agent.title, systemImage: model.running.contains(agent.rawValue) ? "hourglass" : agent.icon)
                .font(.caption.weight(.semibold))
                .padding(.horizontal, 10).padding(.vertical, 6)
                .background(Color.card, in: Capsule())
                .overlay(Capsule().strokeBorder(Color.hairline, lineWidth: 0.5))
        }
        .disabled(model.running.contains(agent.rawValue))
        .tint(Brand.primary)
    }
}

// MARK: - Card

private struct SuggestionCard: View {
    let suggestion: AISuggestion
    let running: Bool
    var onInsert: (String) -> Void
    var onRerun: () -> Void
    var onAdopt: () -> Void
    var onDismiss: () -> Void

    @State private var showReasoning = false

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Header
            HStack(spacing: 8) {
                Image(systemName: suggestion.agent.icon).foregroundStyle(Brand.primary)
                Text(suggestion.agent.title).font(.subheadline.weight(.semibold))
                Spacer()
                if suggestion.isPending || running {
                    Label("Analyzing…", systemImage: "hourglass")
                        .font(.caption2).foregroundStyle(.secondary)
                        .symbolEffect(.pulse)
                } else if suggestion.isDone {
                    confidenceBadge
                } else if suggestion.isFailed {
                    Text("Failed").font(.caption2).foregroundStyle(Brand.danger)
                }
            }

            if suggestion.isDone {
                body(for: suggestion)
            }
        }
        .opacity(suggestion.isDone && suggestion.confidence < 0.4 ? 0.6 : 1)
        .padding(12)
        .background(Color.card, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 14, style: .continuous).strokeBorder(Color.hairline, lineWidth: 0.5))
    }

    private var confidenceBadge: some View {
        Text("\(Int(suggestion.confidence * 100))%")
            .font(.caption2.weight(.bold))
            .padding(.horizontal, 7).padding(.vertical, 2)
            .background(Brand.primary.opacity(0.14), in: Capsule())
            .foregroundStyle(Brand.primary)
    }

    @ViewBuilder
    private func body(for s: AISuggestion) -> some View {
        switch s.agent {
        case .triage:
            if let p = s.decoded(TriagePayload.self) {
                chips([("Priority", p.priority), ("Severity", p.severity), ("Category", p.category)])
                reasoning(p.reasoning)
                actionRow(adoptTitle: "Acknowledge")
            }
        case .sentinel:
            if let p = s.decoded(SentinelPayload.self) {
                HStack(spacing: 6) {
                    if p.escalate == true { tag("Escalate", Brand.danger) }
                    if let s = p.sentiment { tag(s.capitalized, Brand.info) }
                    if let c = p.churnRisk { tag("Churn: \(c)", Brand.primary) }
                    if p.slaBreachRisk == true { tag("SLA risk", Brand.danger) }
                }
                reasoning(p.reasoning)
                actionRow(adoptTitle: "Acknowledge")
            }
        case .researcher:
            if let p = s.decoded(ResearcherPayload.self) {
                if let res = p.suggestedResolution, !res.isEmpty {
                    Text(res).font(.callout).foregroundStyle(.primary).fixedSize(horizontal: false, vertical: true)
                }
                if let cites = p.kbCitations, !cites.isEmpty {
                    listLabel("Knowledge", cites.compactMap { $0.title })
                }
                if let sims = p.similarTickets, !sims.isEmpty {
                    listLabel("Similar tickets", sims.map { "#\($0.id ?? 0) \($0.title ?? "")" })
                }
                actionRow(insert: p.suggestedResolution, insertTitle: "Insert resolution")
            }
        case .reviewer:
            if let p = s.decoded(ReviewerPayload.self) {
                if let issues = p.issues, !issues.isEmpty {
                    ForEach(issues, id: \.self) { i in
                        Text("• \(i.type ?? ""): \(i.note ?? "")").font(.caption).foregroundStyle(.secondary)
                    }
                } else {
                    Text(p.approve == true ? "No issues — looks good." : "Reviewed.").font(.caption).foregroundStyle(.secondary)
                }
                actionRow(insert: p.revisedDraft, insertTitle: "Use revised draft")
            }
        case .drafter:
            if let p = s.decoded(DrafterPayload.self), let reply = p.reply {
                Text(reply).font(.callout).foregroundStyle(.primary).fixedSize(horizontal: false, vertical: true)
                actionRow(insert: reply, insertTitle: "Use draft")
            }
        case .other:
            EmptyView()
        }
    }

    // MARK: bits

    private func chips(_ pairs: [(String, String?)]) -> some View {
        HStack(spacing: 6) {
            ForEach(pairs.indices, id: \.self) { i in
                if let v = pairs[i].1, !v.isEmpty {
                    HStack(spacing: 3) {
                        Text(pairs[i].0).foregroundStyle(.secondary)
                        Text(v.capitalized).fontWeight(.semibold)
                    }
                    .font(.caption2)
                    .padding(.horizontal, 7).padding(.vertical, 3)
                    .background(Color(.tertiarySystemBackground), in: Capsule())
                }
            }
        }
    }

    @ViewBuilder
    private func reasoning(_ text: String?) -> some View {
        if let text, !text.isEmpty {
            DisclosureGroup(isExpanded: $showReasoning) {
                Text(text).font(.caption).foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            } label: {
                Text("Reasoning").font(.caption.weight(.medium)).foregroundStyle(Brand.primary)
            }
            .tint(Brand.primary)
        }
    }

    private func listLabel(_ title: LocalizedStringKey, _ items: [String]) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title).font(.caption2.weight(.semibold)).foregroundStyle(.secondary)
            ForEach(items.prefix(3), id: \.self) { Text("• \($0)").font(.caption).foregroundStyle(.secondary).lineLimit(1) }
        }
    }

    private func actionRow(insert: String? = nil, insertTitle: LocalizedStringKey = "Insert", adoptTitle: LocalizedStringKey? = nil) -> some View {
        HStack(spacing: 10) {
            if let insert, !insert.isEmpty {
                Button { onInsert(insert); onAdopt() } label: {
                    Label(insertTitle, systemImage: "text.insert").font(.caption.weight(.semibold))
                }
                .buttonStyle(.borderedProminent).tint(Brand.primary).controlSize(.small)
            }
            if let adoptTitle {
                Button { onAdopt() } label: { Text(adoptTitle).font(.caption.weight(.semibold)) }
                    .buttonStyle(.bordered).controlSize(.small).tint(Brand.primary)
            }
            Spacer()
            Button { onDismiss() } label: { Image(systemName: "xmark").font(.caption2) }
                .buttonStyle(.bordered).controlSize(.small).tint(.secondary)
            Button(action: onRerun) { Image(systemName: "arrow.clockwise").font(.caption2) }
                .buttonStyle(.bordered).controlSize(.small).tint(.secondary)
        }
    }

    private func tag(_ text: String, _ color: Color) -> some View {
        Text(text).font(.caption2.weight(.semibold))
            .padding(.horizontal, 7).padding(.vertical, 2)
            .background(color.opacity(0.16), in: Capsule()).foregroundStyle(color)
    }
}
