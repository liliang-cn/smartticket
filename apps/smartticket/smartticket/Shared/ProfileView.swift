import SwiftUI

/// Account tab: identity, server, and sign-out. Shared across all portals.
struct ProfileView: View {
    @Environment(AuthStore.self) private var auth
    @Environment(LanguageStore.self) private var lang
    @State private var showServer = false
    @State private var confirmLogout = false

    var body: some View {
        NavigationStack {
            List {
                if let u = auth.user {
                    Section {
                        HStack(spacing: 14) {
                            InitialsAvatar(initials: u.initials, size: 52)
                            VStack(alignment: .leading, spacing: 3) {
                                Text(u.displayName).font(.headline)
                                Text(u.email).font(.caption).foregroundStyle(.secondary)
                                Text(u.role.displayName)
                                    .font(.caption2.weight(.semibold)).textCase(.uppercase)
                                    .foregroundStyle(Brand.primary)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }

                Section("Language") {
                    Picker("Language", selection: Bindable(lang).code) {
                        ForEach(AppLanguages.all) { l in
                            Text(l.label).tag(l.code)
                        }
                    }
                }

                Section("Deployment") {
                    Button { showServer = true } label: {
                        HStack {
                            Label("Server", systemImage: "server.rack")
                            Spacer()
                            Text(hostLabel).foregroundStyle(.secondary).lineLimit(1)
                        }
                    }
                    .tint(.primary)
                }

                Section {
                    Button(role: .destructive) { confirmLogout = true } label: {
                        Label("Sign out", systemImage: "rectangle.portrait.and.arrow.right")
                    }
                }
            }
            .navigationTitle("Account")
            .sheet(isPresented: $showServer) { ServerSettingsView() }
            .confirmationDialog("Sign out of SmartTicket?", isPresented: $confirmLogout, titleVisibility: .visible) {
                Button("Sign out", role: .destructive) { auth.logout() }
                Button("Cancel", role: .cancel) {}
            }
        }
    }

    private var hostLabel: String {
        URL(string: AppConfig.shared.baseURL)?.host ?? AppConfig.shared.baseURL
    }
}
