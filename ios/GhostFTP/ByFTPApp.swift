import SwiftUI

@main
struct ByFTPApp: App {
    @StateObject private var store = SessionStore()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            ContentView(store: store)
        }
        .onChange(of: scenePhase) { phase in
            if phase == .background {
                store.disconnect()
            }
        }
    }
}
