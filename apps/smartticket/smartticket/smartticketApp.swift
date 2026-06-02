//
//  smartticketApp.swift
//  smartticket
//
//  Created by liliang on 2026/6/2.
//

import SwiftUI

@main
struct SmartTicketApp: App {
    @State private var auth = AuthStore()
    @State private var lang = LanguageStore.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(auth)
                .environment(lang)
                .environment(\.locale, lang.locale ?? Locale.autoupdatingCurrent)
                .tint(Brand.primary)
        }
    }
}
