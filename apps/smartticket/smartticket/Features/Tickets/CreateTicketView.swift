import SwiftUI

struct CreateTicketView: View {
    var onCreated: () -> Void

    @Environment(\.dismiss) private var dismiss
    @Environment(AuthStore.self) private var auth

    @State private var title = ""
    @State private var description = ""
    @State private var priority: TicketPriority = .medium
    @State private var requesterName = ""
    @State private var requesterEmail = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                Section("Subject") {
                    TextField("Brief summary", text: $title)
                }
                Section("Details") {
                    TextField("Describe the issue…", text: $description, axis: .vertical)
                        .lineLimit(4...10)
                }
                Section("Priority") {
                    Picker("Priority", selection: $priority) {
                        ForEach(TicketPriority.allCases, id: \.self) { Text($0.label).tag($0) }
                    }
                    .pickerStyle(.segmented)
                }
                Section("Requester") {
                    TextField("Name", text: $requesterName)
                    TextField("Email", text: $requesterEmail)
                        .keyboardType(.emailAddress)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }
                if let error {
                    Text(error).foregroundStyle(Brand.danger).font(.callout)
                }
            }
            .navigationTitle("New ticket")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit", action: submit)
                        .disabled(busy || title.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .onAppear {
                if let u = auth.user {
                    if requesterName.isEmpty { requesterName = u.displayName }
                    if requesterEmail.isEmpty { requesterEmail = u.email }
                }
            }
        }
    }

    private func submit() {
        busy = true
        error = nil
        struct Body: Encodable {
            let title: String
            let description: String
            let priority: String
            let severity: String
            let requester_name: String
            let requester_email: String
        }
        let body = Body(
            title: title.trimmingCharacters(in: .whitespaces),
            description: description,
            priority: priority.rawValue,
            severity: "minor",
            requester_name: requesterName,
            requester_email: requesterEmail
        )
        Task {
            do {
                let _: Ticket = try await APIClient.shared.post("/tickets/", body: body)
                onCreated()
                dismiss()
            } catch {
                self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
            }
            busy = false
        }
    }
}
