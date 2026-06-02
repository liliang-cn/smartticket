import SwiftUI

/// A soft, filled status pill with an icon.
struct StatusBadge: View {
    let status: TicketStatus
    var body: some View {
        Label {
            Text(status.label)
        } icon: {
            Image(systemName: status.icon)
        }
        .font(.caption2.weight(.semibold))
        .labelStyle(.titleAndIcon)
        .padding(.horizontal, 9).padding(.vertical, 4)
        .background(status.tint.opacity(0.14), in: Capsule())
        .foregroundStyle(status.tint)
    }
}

/// An outlined priority pill with a leading dot.
struct PriorityBadge: View {
    let priority: TicketPriority
    var body: some View {
        HStack(spacing: 5) {
            Circle().fill(priority.tint).frame(width: 6, height: 6)
            Text(priority.label)
        }
        .font(.caption2.weight(.semibold))
        .padding(.horizontal, 9).padding(.vertical, 4)
        .overlay(Capsule().strokeBorder(priority.tint.opacity(0.35), lineWidth: 1))
        .foregroundStyle(priority.tint)
    }
}

/// Circular initials avatar with a gradient ring.
struct InitialsAvatar: View {
    let initials: String
    var size: CGFloat = 40
    var body: some View {
        Text(initials)
            .font(.system(size: size * 0.38, weight: .bold, design: .rounded))
            .foregroundStyle(Brand.primary)
            .frame(width: size, height: size)
            .background(Brand.wash(Brand.primary), in: Circle())
            .overlay(Circle().strokeBorder(Brand.primary.opacity(0.25), lineWidth: 1))
    }
}

/// Centered placeholder for empty collections.
struct EmptyStateView: View {
    let systemImage: String
    let title: LocalizedStringKey
    var message: LocalizedStringKey? = nil
    var body: some View {
        VStack(spacing: 14) {
            Image(systemName: systemImage)
                .font(.system(size: 40, weight: .light))
                .foregroundStyle(Brand.primary.opacity(0.7))
                .frame(width: 84, height: 84)
                .background(Brand.wash(Brand.primary), in: Circle())
            Text(title).font(.headline)
            if let message {
                Text(message).font(.subheadline).foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(40)
    }
}

/// Inline error with an optional retry action.
struct InlineError: View {
    let message: String
    var retry: (() -> Void)? = nil
    var body: some View {
        VStack(spacing: 14) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.title)
                .foregroundStyle(Brand.danger)
                .frame(width: 84, height: 84)
                .background(Brand.wash(Brand.danger), in: Circle())
            Text(message).font(.callout).multilineTextAlignment(.center)
                .foregroundStyle(.secondary)
            if let retry {
                Button(action: retry) {
                    Label("Try again", systemImage: "arrow.clockwise")
                }
                .buttonStyle(.borderedProminent)
                .tint(Brand.primary)
            }
        }
        .padding(40)
        .frame(maxWidth: .infinity)
    }
}

/// A gradient-washed metric card with an icon chip and a large rounded number.
struct StatCard: View {
    let value: Int
    let label: LocalizedStringKey
    var systemImage: String = "number"
    var tint: Color = Brand.primary

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(tint)
                .frame(width: 34, height: 34)
                .background(tint.opacity(0.14), in: RoundedRectangle(cornerRadius: 10, style: .continuous))

            Text("\(value)")
                .font(.system(size: 32, weight: .bold, design: .rounded))
                .foregroundStyle(.primary)
                .contentTransition(.numericText())

            Text(label)
                .font(.caption.weight(.medium))
                .foregroundStyle(.secondary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(Brand.wash(tint), in: RoundedRectangle(cornerRadius: 20, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 20, style: .continuous)
                .strokeBorder(tint.opacity(0.15), lineWidth: 0.5)
        )
    }
}

/// Section eyebrow + optional trailing accessory.
struct SectionLabel: View {
    let text: LocalizedStringKey
    var body: some View {
        Text(text).eyebrow()
    }
}
