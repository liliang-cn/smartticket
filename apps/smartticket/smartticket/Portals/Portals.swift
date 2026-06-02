import SwiftUI

/// Customer portal — the self-service surface. Customers see only their own
/// tickets (scoped server-side), can open new ones, and browse the KB.
struct CustomerPortal: View {
    var body: some View {
        TabView {
            Tab("My Tickets", systemImage: "ticket") {
                TicketListView(title: "My Tickets", canManage: false, canCreate: true)
            }
            Tab("Knowledge", systemImage: "book") {
                KnowledgeView()
            }
            Tab("Account", systemImage: "person.crop.circle") {
                ProfileView()
            }
        }
    }
}

/// Team-member portal — operators handling the full ticket queue.
struct TeamPortal: View {
    var body: some View {
        TabView {
            Tab("Queue", systemImage: "tray.full") {
                TicketListView(title: "Queue", canManage: true, canCreate: true)
            }
            Tab("Knowledge", systemImage: "book") {
                KnowledgeView()
            }
            Tab("Account", systemImage: "person.crop.circle") {
                ProfileView()
            }
        }
    }
}

/// Admin portal — operators plus deployment-wide overview and user management.
struct AdminPortal: View {
    var body: some View {
        TabView {
            Tab("Dashboard", systemImage: "chart.bar") {
                DashboardView()
            }
            Tab("Tickets", systemImage: "tray.full") {
                TicketListView(title: "Tickets", canManage: true, canCreate: true)
            }
            Tab("Knowledge", systemImage: "book") {
                KnowledgeView()
            }
            Tab("Users", systemImage: "person.2") {
                UsersView()
            }
            Tab("Account", systemImage: "person.crop.circle") {
                ProfileView()
            }
        }
    }
}
