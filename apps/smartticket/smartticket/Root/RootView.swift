import SwiftUI

/// Top-level switch: loading → login → role-appropriate portal.
struct RootView: View {
    @Environment(AuthStore.self) private var auth

    var body: some View {
        let phase = auth.phase
        return content(phase)
            .animation(.default, value: phase)
            .task { if auth.phase == .loading { await auth.bootstrap() } }
    }

    @ViewBuilder
    private func content(_ phase: AuthStore.Phase) -> some View {
        switch phase {
        case .loading:
            LaunchView()
        case .signedOut:
            LoginView()
        case .signedIn:
            portal(for: auth.user?.role ?? .customer)
        }
    }

    @ViewBuilder
    private func portal(for role: Role) -> some View {
        if role.isAdmin {
            AdminPortal()
        } else if role.isTeam {
            TeamPortal()
        } else {
            CustomerPortal()
        }
    }
}

private struct LaunchView: View {
    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "ticket.fill")
                .font(.system(size: 44))
                .foregroundStyle(Brand.primary)
            ProgressView()
        }
    }
}
