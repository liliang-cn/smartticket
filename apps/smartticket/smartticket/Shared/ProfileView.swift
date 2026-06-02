import SwiftUI

/// Account tab: identity, language, server, and sign-out. Shared across portals.
struct ProfileView: View {
    @Environment(AuthStore.self) private var auth
    @Environment(LanguageStore.self) private var lang
    @State private var showServer = false
    @State private var confirmLogout = false

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    if let u = auth.user { identityCard(u) }
                    settingsCard
                    signOutButton
                }
                .padding(20)
            }
            .background(Color.appBG)
            .navigationTitle("Account")
            .sheet(isPresented: $showServer) { ServerSettingsView() }
            .confirmationDialog("Sign out of SmartTicket?", isPresented: $confirmLogout, titleVisibility: .visible) {
                Button("Sign out", role: .destructive) { auth.logout() }
                Button("Cancel", role: .cancel) {}
            }
        }
    }

    private func identityCard(_ u: UserInfo) -> some View {
        VStack(spacing: 12) {
            InitialsAvatar(initials: u.initials, size: 72)
            VStack(spacing: 4) {
                Text(u.displayName).font(.title3.bold())
                Text(u.email).font(.subheadline).foregroundStyle(.secondary)
            }
            Text(u.role.displayName)
                .font(.caption.weight(.semibold)).textCase(.uppercase)
                .foregroundStyle(Brand.primary)
                .padding(.horizontal, 12).padding(.vertical, 5)
                .background(Brand.wash(Brand.primary), in: Capsule())
        }
        .frame(maxWidth: .infinity)
        .card(padding: 24)
    }

    private var settingsCard: some View {
        VStack(spacing: 0) {
            HStack {
                Label("Language", systemImage: "globe").font(.callout)
                Spacer()
                Picker("Language", selection: Bindable(lang).code) {
                    ForEach(AppLanguages.all) { l in Text(l.label).tag(l.code) }
                }
                .labelsHidden()
                .tint(.secondary)
            }
            .padding(.vertical, 14)

            Divider()

            Button { showServer = true } label: {
                HStack {
                    Label("Server", systemImage: "server.rack").font(.callout)
                    Spacer()
                    Text(hostLabel).foregroundStyle(.secondary).lineLimit(1).font(.callout)
                    Image(systemName: "chevron.right").font(.caption.weight(.semibold)).foregroundStyle(.tertiary)
                }
                .padding(.vertical, 14)
            }
            .tint(.primary)
        }
        .padding(.horizontal, 16)
        .background(Color.card, in: RoundedRectangle(cornerRadius: 20, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 20, style: .continuous).strokeBorder(Color.hairline, lineWidth: 0.5))
    }

    private var signOutButton: some View {
        Button(role: .destructive) { confirmLogout = true } label: {
            Label("Sign out", systemImage: "rectangle.portrait.and.arrow.right")
                .font(.body.weight(.medium))
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .foregroundStyle(Brand.danger)
                .background(Brand.wash(Brand.danger), in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        }
    }

    private var hostLabel: String {
        URL(string: AppConfig.shared.baseURL)?.host ?? AppConfig.shared.baseURL
    }
}
