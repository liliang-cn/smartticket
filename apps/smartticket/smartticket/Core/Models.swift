import Foundation
import SwiftUI

// MARK: - Roles

enum Role: String, Codable, CaseIterable {
    case admin, engineer, support, customer, sales

    var isCustomer: Bool { self == .customer }
    /// Everyone who is not a customer is "team" (operator-facing surface).
    var isTeam: Bool { self != .customer }
    var isAdmin: Bool { self == .admin }

    var displayName: LocalizedStringKey {
        switch self {
        case .admin: return "Admin"
        case .engineer: return "Engineer"
        case .support: return "Support"
        case .customer: return "Customer"
        case .sales: return "Sales"
        }
    }
}

// MARK: - User

struct UserInfo: Codable, Identifiable, Hashable {
    let id: Int
    let email: String
    let username: String
    let firstName: String
    let lastName: String
    let role: Role
    let isActive: Bool
    let lastLoginAt: String?
    let createdAt: String?
    let customerId: Int?

    enum CodingKeys: String, CodingKey {
        case id, email, username, role
        case firstName = "first_name"
        case lastName = "last_name"
        case isActive = "is_active"
        case lastLoginAt = "last_login_at"
        case createdAt = "created_at"
        case customerId = "customer_id"
    }

    var displayName: String {
        let full = "\(firstName) \(lastName)".trimmingCharacters(in: .whitespaces)
        return full.isEmpty ? username : full
    }

    var initials: String {
        let a = firstName.first.map(String.init) ?? username.first.map(String.init) ?? "?"
        let b = lastName.first.map(String.init) ?? ""
        return (a + b).uppercased()
    }
}

// MARK: - Auth

struct TokenPair: Codable {
    let accessToken: String
    let refreshToken: String
    let expiresAt: String?
    let tokenType: String?

    enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case expiresAt = "expires_at"
        case tokenType = "token_type"
    }
}

struct LoginResponse: Codable {
    let user: UserInfo
    let tokens: TokenPair
}

struct RefreshResponse: Codable {
    let tokens: TokenPair
}

// MARK: - Tickets

enum TicketStatus: String, Codable, CaseIterable {
    case open, inProgress = "in_progress", resolved, closed, cancelled

    var label: LocalizedStringKey {
        switch self {
        case .open: return "Open"
        case .inProgress: return "In progress"
        case .resolved: return "Resolved"
        case .closed: return "Closed"
        case .cancelled: return "Cancelled"
        }
    }
}

enum TicketPriority: String, Codable, CaseIterable {
    case low, medium, high, critical

    var label: LocalizedStringKey {
        switch self {
        case .low: return "Low"
        case .medium: return "Medium"
        case .high: return "High"
        case .critical: return "Critical"
        }
    }
}

struct Ticket: Codable, Identifiable, Hashable {
    let id: Int
    let ticketNumber: String?
    let title: String
    let description: String?
    let status: TicketStatus
    let priority: TicketPriority
    let severity: String?
    let category: String?
    let customerId: Int?
    let customerName: String?
    let assignedTo: Int?
    let assignedUser: UserInfo?
    let requesterName: String?
    let requesterEmail: String?
    let createdAt: String?
    let updatedAt: String?
    let resolvedAt: String?
    let dueDate: String?
    let slaStatus: String?

    enum CodingKeys: String, CodingKey {
        case id, title, description, status, priority, severity, category
        case ticketNumber = "ticket_number"
        case customerId = "customer_id"
        case customerName = "customer_name"
        case assignedTo = "assigned_to"
        case assignedUser = "assigned_user"
        case requesterName = "requester_name"
        case requesterEmail = "requester_email"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case resolvedAt = "resolved_at"
        case dueDate = "due_date"
        case slaStatus = "sla_status"
    }
}

struct TicketMessage: Codable, Identifiable, Hashable {
    let id: Int
    let ticketId: Int
    let userId: Int?
    let authorName: String?
    let authorRole: String?
    let content: String
    let isInternal: Bool
    let isFromAI: Bool
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id, content
        case ticketId = "ticket_id"
        case userId = "user_id"
        case authorName = "author_name"
        case authorRole = "author_role"
        case isInternal = "is_internal"
        case isFromAI = "is_from_ai"
        case createdAt = "created_at"
    }
}

struct TicketEvent: Codable, Identifiable, Hashable {
    let id: Int
    let action: String
    let summary: String
    let actorName: String?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id, action, summary
        case actorName = "actor_name"
        case createdAt = "created_at"
    }
}

struct TicketStats: Codable {
    let totalTickets: Int
    let openTickets: Int
    let inProgressTickets: Int
    let resolvedTickets: Int
    let closedTickets: Int
    let overdueTickets: Int

    enum CodingKeys: String, CodingKey {
        case totalTickets = "total_tickets"
        case openTickets = "open_tickets"
        case inProgressTickets = "in_progress_tickets"
        case resolvedTickets = "resolved_tickets"
        case closedTickets = "closed_tickets"
        case overdueTickets = "overdue_tickets"
    }
}

// MARK: - Knowledge

struct KnowledgeArticle: Codable, Identifiable, Hashable {
    let id: Int
    let title: String
    let content: String?
    let summary: String?
    let category: String?
    let status: String?
    let viewCount: Int?
    let version: Int?
    let createdAt: String?
    let updatedAt: String?
    let createdBy: String?

    enum CodingKeys: String, CodingKey {
        case id, title, content, summary, category, status, version
        case viewCount = "view_count"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
        case createdBy = "created_by"
    }
}

// MARK: - Branding

struct Branding: Codable {
    let appName: String
    let appSubtitle: String
    let workspaceName: String
    let primaryColor: String
    let loginTagline: String
    let loginSubtext: String

    enum CodingKeys: String, CodingKey {
        case appName = "app_name"
        case appSubtitle = "app_subtitle"
        case workspaceName = "workspace_name"
        case primaryColor = "primary_color"
        case loginTagline = "login_tagline"
        case loginSubtext = "login_subtext"
    }
}

// MARK: - Pagination

struct PageMeta: Codable {
    let total: Int?
    let page: Int?
    let pageSize: Int?
    let totalPages: Int?

    enum CodingKeys: String, CodingKey {
        case total, page
        case pageSize = "page_size"
        case totalPages = "total_pages"
    }
}
