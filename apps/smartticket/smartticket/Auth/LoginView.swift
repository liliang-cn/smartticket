import SwiftUI

struct LoginView: View {
    @Environment(AuthStore.self) private var auth

    @State private var email = ""
    @State private var password = ""
    @State private var busy = false
    @State private var error: String?
    @State private var showServerSheet = false
    @FocusState private var focus: Field?

    private enum Field { case email, password }

    var body: some View {
        NavigationStack {
            ZStack {
                backdrop
                ScrollView {
                    VStack(spacing: 28) {
                        header
                        formCard
                    }
                    .padding(24)
                    .frame(maxWidth: 460)
                    .frame(maxWidth: .infinity)
                }
                .scrollDismissesKeyboard(.interactively)
            }
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { showServerSheet = true } label: {
                        Image(systemName: "server.rack")
                    }
                    .tint(.secondary)
                }
            }
            .sheet(isPresented: $showServerSheet) { ServerSettingsView() }
        }
    }

    /// Soft amber glow bleeding from the top — the brand's signature backdrop.
    private var backdrop: some View {
        ZStack(alignment: .top) {
            Color.appBG.ignoresSafeArea()
            RadialGradient(
                colors: [Brand.primary.opacity(0.22), .clear],
                center: .top, startRadius: 0, endRadius: 420
            )
            .ignoresSafeArea()
        }
    }

    private var header: some View {
        VStack(spacing: 16) {
            Image(systemName: "ticket.fill")
                .font(.system(size: 34, weight: .semibold))
                .foregroundStyle(Brand.ink)
                .frame(width: 76, height: 76)
                .background(Brand.gradient, in: RoundedRectangle(cornerRadius: 22, style: .continuous))
                .shadow(color: Brand.primary.opacity(0.5), radius: 20, y: 8)
            VStack(spacing: 6) {
                Text("Sign in").font(.largeTitle.bold())
                Text("Access your support workspace.")
                    .font(.callout).foregroundStyle(.secondary)
            }
        }
        .padding(.top, 32)
    }

    private var formCard: some View {
        VStack(spacing: 16) {
            HStack(spacing: 12) {
                Image(systemName: "envelope").font(.system(size: 16)).foregroundStyle(.secondary).frame(width: 22)
                TextField("Email", text: $email)
                    .focused($focus, equals: .email)
                    .keyboardType(.emailAddress)
                    .textContentType(.username)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .submitLabel(.next)
                    .onSubmit { focus = .password }
            }
            .fieldChrome()

            HStack(spacing: 12) {
                Image(systemName: "lock").font(.system(size: 16)).foregroundStyle(.secondary).frame(width: 22)
                SecureField("Password", text: $password)
                    .focused($focus, equals: .password)
                    .textContentType(.password)
                    .submitLabel(.go)
                    .onSubmit(submit)
            }
            .fieldChrome()

            if let error {
                Label(error, systemImage: "exclamationmark.circle.fill")
                    .font(.footnote)
                    .foregroundStyle(Brand.danger)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .transition(.opacity)
            }

            Button(action: submit) {
                HStack(spacing: 8) {
                    if busy { ProgressView().tint(Brand.ink) }
                    Text(busy ? "Signing in…" : "Sign in").fontWeight(.semibold)
                    if !busy { Image(systemName: "arrow.right") }
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 15)
                .foregroundStyle(Brand.ink)
                .background(Brand.gradient, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                .opacity(canSubmit ? 1 : 0.5)
            }
            .disabled(!canSubmit)
            .padding(.top, 4)
        }
        .card(padding: 18, radius: 24)
        .animation(.default, value: error)
    }

    private var canSubmit: Bool { !busy && !email.isEmpty && !password.isEmpty }

    private func submit() {
        guard canSubmit else { return }
        busy = true; error = nil; focus = nil
        Task {
            do {
                try await auth.login(email: email.trimmingCharacters(in: .whitespaces), password: password)
            } catch {
                self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
            }
            busy = false
        }
    }
}

private extension View {
    /// Capsule-ish chrome for a login field row.
    func fieldChrome() -> some View {
        self.padding(.horizontal, 14).padding(.vertical, 14)
            .background(Color(.tertiarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: 13, style: .continuous))
    }
}

/// Lets the user point the app at their self-hosted deployment.
struct ServerSettingsView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var url = AppConfig.shared.baseURL

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("https://support.example.com", text: $url)
                        .keyboardType(.URL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                } header: {
                    Text("Server URL")
                } footer: {
                    Text("The address of your self-hosted SmartTicket deployment.")
                }
            }
            .navigationTitle("Server")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        AppConfig.shared.baseURL = url.trimmingCharacters(in: .whitespaces)
                        dismiss()
                    }.fontWeight(.semibold)
                }
            }
        }
    }
}
