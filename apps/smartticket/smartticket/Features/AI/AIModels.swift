import SwiftUI

// MARK: - AI advisory team suggestions

/// One advisory suggestion produced by an AI team agent for a ticket.
/// `payload` is a JSON string whose shape depends on `agentName`.
struct AISuggestion: Codable, Identifiable, Hashable {
    let id: Int
    let agentName: String
    let status: String // pending | done | adopted | dismissed | failed
    let confidence: Double
    let payload: String

    enum CodingKeys: String, CodingKey {
        case id, status, confidence, payload
        case agentName = "agent_name"
    }

    var agent: AIAgent { AIAgent(rawValue: agentName) ?? .other }
    var isDone: Bool { status == "done" || status == "adopted" }
    var isPending: Bool { status == "pending" }
    var isDismissed: Bool { status == "dismissed" }
    var isFailed: Bool { status == "failed" }

    /// Decode the agent-specific payload struct from the JSON string.
    func decoded<T: Decodable>(_ type: T.Type) -> T? {
        guard let data = payload.data(using: .utf8), !payload.isEmpty else { return nil }
        return try? JSONDecoder().decode(T.self, from: data)
    }
}

enum AIAgent: String, CaseIterable {
    case triage = "Triage"
    case sentinel = "Sentinel"
    case researcher = "Researcher"
    case reviewer = "Reviewer"
    case drafter = "Drafter"
    case other = "?"

    var title: LocalizedStringKey {
        switch self {
        case .triage: return "Triage"
        case .sentinel: return "Escalation risk"
        case .researcher: return "Researcher"
        case .reviewer: return "Reviewer"
        case .drafter: return "Draft reply"
        case .other: return "Agent"
        }
    }

    var icon: String {
        switch self {
        case .triage: return "scope"
        case .sentinel: return "exclamationmark.triangle"
        case .researcher: return "magnifyingglass"
        case .reviewer: return "checkmark.seal"
        case .drafter: return "pencil.and.outline"
        case .other: return "sparkles"
        }
    }

    /// Display order in the panel.
    var order: Int {
        switch self {
        case .triage: return 0
        case .sentinel: return 1
        case .researcher: return 2
        case .reviewer: return 3
        case .drafter: return 4
        case .other: return 9
        }
    }

    /// On-demand agents have a Run button; auto agents appear when present.
    var isOnDemand: Bool { self == .researcher || self == .reviewer || self == .drafter }
}

// MARK: - Per-agent payloads

struct TriagePayload: Decodable {
    let priority: String?
    let severity: String?
    let category: String?
    let reasoning: String?
    let confidence: Double?
}

struct SentinelPayload: Decodable {
    let sentiment: String?
    let churnRisk: String?
    let slaBreachRisk: Bool?
    let escalate: Bool?
    let reasoning: String?
    let confidence: Double?

    enum CodingKeys: String, CodingKey {
        case sentiment, escalate, reasoning, confidence
        case churnRisk = "churn_risk"
        case slaBreachRisk = "sla_breach_risk"
    }
}

struct KBSnippet: Decodable, Hashable {
    let title: String?
    let snippet: String?
}

struct SimilarTicket: Decodable, Hashable {
    let id: Int?
    let title: String?
    let resolution: String?
    let mergeCandidate: Bool?
    let score: Double?

    enum CodingKeys: String, CodingKey {
        case id, title, resolution, score
        case mergeCandidate = "merge_candidate"
    }
}

struct ResearcherPayload: Decodable {
    let kbCitations: [KBSnippet]?
    let similarTickets: [SimilarTicket]?
    let suggestedResolution: String?
    let confidence: Double?

    enum CodingKeys: String, CodingKey {
        case confidence
        case kbCitations = "kb_citations"
        case similarTickets = "similar_tickets"
        case suggestedResolution = "suggested_resolution"
    }
}

struct ReviewIssue: Decodable, Hashable {
    let type: String?
    let severity: String?
    let note: String?
}

struct ReviewerPayload: Decodable {
    let issues: [ReviewIssue]?
    let revisedDraft: String?
    let approve: Bool?
    let confidence: Double?

    enum CodingKeys: String, CodingKey {
        case issues, approve, confidence
        case revisedDraft = "revised_draft"
    }
}

struct DrafterPayload: Decodable {
    let reply: String?
    let confidence: Double?
}

/// Wrapper for `GET /tickets/:id/ai/suggestions` which returns `{ suggestions: [...] }`.
struct SuggestionsResponse: Decodable {
    let suggestions: [AISuggestion]
}
