import SwiftUI

/// The brand palette — amber accent over adaptive surfaces. The app reads well
/// in both light and dark; surfaces use the system grouped colors so it feels
/// native, while the amber + gradients give it a distinct, premium identity.
enum Brand {
    static let primary = Color(hex: 0xF59E0B)   // amber
    static let primaryHi = Color(hex: 0xFFB01F) // brighter amber for gradients
    static let ink = Color(hex: 0x19130A)       // legible foreground on amber
    static let success = Color(hex: 0x10B981)
    static let info = Color(hex: 0x3B82F6)
    static let violet = Color(hex: 0x8B5CF6)
    static let danger = Color(hex: 0xEF4444)

    /// Signature accent gradient used on heroes, key numbers and primary CTAs.
    static let gradient = LinearGradient(
        colors: [Color(hex: 0xFFB01F), Color(hex: 0xF59E0B)],
        startPoint: .topLeading, endPoint: .bottomTrailing
    )

    /// A soft tinted gradient for card washes.
    static func wash(_ c: Color) -> LinearGradient {
        LinearGradient(
            colors: [c.opacity(0.16), c.opacity(0.04)],
            startPoint: .topLeading, endPoint: .bottomTrailing
        )
    }
}

extension Color {
    init(hex: UInt32, alpha: Double = 1) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: alpha
        )
    }

    /// Screen background (subtle grouped gray / near-black).
    static let appBG = Color(.systemGroupedBackground)
    /// Raised card surface.
    static let card = Color(.secondarySystemGroupedBackground)
    /// Hairline separators / borders.
    static let hairline = Color(.separator).opacity(0.5)
}

// MARK: - Card surface

private struct CardSurface: ViewModifier {
    var padding: CGFloat = 16
    var radius: CGFloat = 20
    func body(content: Content) -> some View {
        content
            .padding(padding)
            .background(Color.card, in: RoundedRectangle(cornerRadius: radius, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: radius, style: .continuous)
                    .strokeBorder(Color.hairline, lineWidth: 0.5)
            )
            .shadow(color: .black.opacity(0.06), radius: 12, y: 5)
    }
}

extension View {
    /// Wrap content in the app's standard elevated card.
    func card(padding: CGFloat = 16, radius: CGFloat = 20) -> some View {
        modifier(CardSurface(padding: padding, radius: radius))
    }

    /// A small uppercase mono-ish caption used as a section/eyebrow label.
    func eyebrow() -> some View {
        self.font(.caption2.weight(.semibold))
            .textCase(.uppercase)
            .tracking(1.2)
            .foregroundStyle(.secondary)
    }
}

// MARK: - Semantic tints

extension TicketStatus {
    var tint: Color {
        switch self {
        case .open: return Brand.primary
        case .inProgress: return Brand.info
        case .resolved, .closed: return Brand.success
        case .cancelled: return .secondary
        }
    }
    var icon: String {
        switch self {
        case .open: return "tray"
        case .inProgress: return "arrow.triangle.2.circlepath"
        case .resolved: return "checkmark.circle.fill"
        case .closed: return "lock.fill"
        case .cancelled: return "slash.circle"
        }
    }
}

extension TicketPriority {
    var tint: Color {
        switch self {
        case .low: return .secondary
        case .medium: return Brand.info
        case .high: return Brand.primary
        case .critical: return Brand.danger
        }
    }
}

// MARK: - Relative / absolute date helpers

enum DateText {
    private static let iso = ISO8601DateFormatter()
    private static let isoFractional: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()

    static func parse(_ s: String?) -> Date? {
        guard let s, !s.isEmpty else { return nil }
        return iso.date(from: s) ?? isoFractional.date(from: s)
    }

    static func relative(_ s: String?) -> String {
        guard let date = parse(s) else { return "—" }
        let f = RelativeDateTimeFormatter()
        f.unitsStyle = .abbreviated
        return f.localizedString(for: date, relativeTo: Date())
    }

    static func medium(_ s: String?) -> String {
        guard let date = parse(s) else { return "—" }
        return date.formatted(date: .abbreviated, time: .shortened)
    }
}
