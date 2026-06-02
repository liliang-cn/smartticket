import SwiftUI

/// The brand palette, mirroring the web console's "mission-control" aesthetic
/// (amber accent on a near-black surface).
enum Brand {
    static let primary = Color(hex: 0xFFB01F)
    static let primaryInk = Color(hex: 0x19130A)
    static let success = Color(hex: 0x34D399)
    static let info = Color(hex: 0x4CC4FF)
    static let danger = Color(hex: 0xF0494F)
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
}

extension TicketStatus {
    var tint: Color {
        switch self {
        case .open: return Brand.primary
        case .inProgress: return Brand.info
        case .resolved, .closed: return Brand.success
        case .cancelled: return .secondary
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

    /// "2h ago", "3d ago", localized to the device.
    static func relative(_ s: String?) -> String {
        guard let date = parse(s) else { return "—" }
        let f = RelativeDateTimeFormatter()
        f.unitsStyle = .abbreviated
        return f.localizedString(for: date, relativeTo: Date())
    }

    /// "Jun 2, 2026 at 3:04 PM", localized to the device.
    static func medium(_ s: String?) -> String {
        guard let date = parse(s) else { return "—" }
        return date.formatted(date: .abbreviated, time: .shortened)
    }
}
