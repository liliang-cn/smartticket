import SwiftUI

struct Macro: Codable, Identifiable, Hashable {
    let id: Int
    let title: String
    let category: String?
    let body: String
}

@MainActor
@Observable
final class MacrosModel {
    var macros: [Macro] = []
    var loading = false
    var error: String?

    func load() async {
        loading = true; error = nil
        defer { loading = false }
        do {
            let (items, _): ([Macro], PageMeta?) = try await APIClient.shared.getList("/macros")
            macros = items
        } catch {
            self.error = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }
}

/// A sheet that lists canned replies (macros); tapping one inserts its body.
struct MacroPicker: View {
    var onPick: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var model = MacrosModel()

    var body: some View {
        NavigationStack {
            Group {
                if let error = model.error, model.macros.isEmpty {
                    InlineError(message: error) { Task { await model.load() } }
                } else if model.macros.isEmpty && model.loading {
                    ProgressView().frame(maxWidth: .infinity).padding(.top, 60)
                } else if model.macros.isEmpty {
                    EmptyStateView(systemImage: "text.badge.checkmark", title: "No macros",
                                   message: "Canned replies appear here.")
                } else {
                    List(model.macros) { macro in
                        Button {
                            onPick(macro.body)
                            dismiss()
                        } label: {
                            VStack(alignment: .leading, spacing: 4) {
                                HStack(spacing: 6) {
                                    Text(macro.title).font(.subheadline.weight(.semibold)).foregroundStyle(.primary)
                                    if let cat = macro.category, !cat.isEmpty {
                                        Text(cat).font(.caption2).foregroundStyle(Brand.primary)
                                            .padding(.horizontal, 6).padding(.vertical, 1)
                                            .background(Brand.primary.opacity(0.12), in: Capsule())
                                    }
                                }
                                Text(macro.body).font(.caption).foregroundStyle(.secondary).lineLimit(2)
                            }
                            .padding(.vertical, 2)
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Macros")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") { dismiss() }
                }
            }
            .task { if model.macros.isEmpty { await model.load() } }
        }
        .presentationDetents([.medium, .large])
    }
}
