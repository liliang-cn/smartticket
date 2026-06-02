import SwiftUI

struct LoginView: View {
    @Environment(AuthStore.self) private var auth

    @State private var email = ""
    @State private var password = ""
    @State private var busy = false
    @State private var error: String?
    @State private var showServerSheet = false

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    header
                    fields
                    if let error {
                        Text(error)
                            .font(.callout)
                            .foregroundStyle(Brand.danger)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                    Button(action: submit) {
                        HStack {
                            if busy { ProgressView().tint(Brand.primaryInk) }
                            Text(busy ? "Signing in…" : "Sign in")
                                .fontWeight(.semibold)
                        }
                        .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.large)
                    .disabled(busy || email.isEmpty || password.isEmpty)
                }
                .padding(24)
            }
            .scrollDismissesKeyboard(.interactively)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        showServerSheet = true
                    } label: {
                        Image(systemName: "server.rack")
                    }
                }
            }
            .sheet(isPresented: $showServerSheet) { ServerSettingsView() }
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Image(systemName: "ticket.fill")
                .font(.system(size: 40))
                .foregroundStyle(Brand.primary)
                .padding(.bottom, 4)
            Text("Sign in")
                .font(.largeTitle.bold())
            Text("Access your support workspace.")
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.top, 24)
    }

    private var fields: some View {
        VStack(spacing: 14) {
            TextField("Email", text: $email)
                .textContentType(.username)
                .keyboardType(.emailAddress)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            SecureField("Password", text: $password)
                .textContentType(.password)
        }
        .textFieldStyle(.roundedBorder)
        .onSubmit(submit)
    }

    private func submit() {
        guard !busy, !email.isEmpty, !password.isEmpty else { return }
        busy = true
        error = nil
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
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        AppConfig.shared.baseURL = url.trimmingCharacters(in: .whitespaces)
                        dismiss()
                    }
                }
            }
        }
    }
}
