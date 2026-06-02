import SwiftUI

/// A small pill rendering a ticket status with its tint.
struct StatusBadge: View {
    let status: TicketStatus
    var body: some View {
        Text(status.label)
            .font(.caption2.weight(.semibold))
            .textCase(.uppercase)
            .padding(.horizontal, 8).padding(.vertical, 3)
            .background(status.tint.opacity(0.15), in: Capsule())
            .foregroundStyle(status.tint)
    }
}

struct PriorityBadge: View {
    let priority: TicketPriority
    var body: some View {
        Text(priority.label)
            .font(.caption2.weight(.semibold))
            .textCase(.uppercase)
            .padding(.horizontal, 8).padding(.vertical, 3)
            .overlay(Capsule().stroke(priority.tint.opacity(0.4), lineWidth: 1))
            .foregroundStyle(priority.tint)
    }
}

/// Circular initials avatar.
struct InitialsAvatar: View {
    let initials: String
    var size: CGFloat = 36
    var body: some View {
        Text(initials)
            .font(.system(size: size * 0.38, weight: .semibold))
            .frame(width: size, height: size)
            .background(Brand.primary.opacity(0.18), in: Circle())
            .foregroundStyle(Brand.primary)
    }
}

/// Centered placeholder for empty collections.
struct EmptyStateView: View {
    let systemImage: String
    let title: LocalizedStringKey
    var message: LocalizedStringKey? = nil
    var body: some View {
        ContentUnavailableView {
            Label(title, systemImage: systemImage)
        } description: {
            if let message { Text(message) }
        }
    }
}

/// Inline error row with an optional retry action.
struct InlineError: View {
    let message: String
    var retry: (() -> Void)? = nil
    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.title2).foregroundStyle(Brand.danger)
            Text(message).font(.callout).multilineTextAlignment(.center)
                .foregroundStyle(.secondary)
            if let retry {
                Button("Try again", action: retry).buttonStyle(.bordered)
            }
        }
        .padding()
        .frame(maxWidth: .infinity)
    }
}

/// A labeled metric card used on the dashboard.
struct StatCard: View {
    let value: Int
    let label: LocalizedStringKey
    var tint: Color = .primary
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("\(value)")
                .font(.system(size: 30, weight: .bold, design: .rounded))
                .foregroundStyle(tint)
                .contentTransition(.numericText())
            Text(label)
                .font(.caption2.weight(.medium))
                .textCase(.uppercase)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(.background.secondary, in: RoundedRectangle(cornerRadius: 14))
    }
}
