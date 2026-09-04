import SwiftUI

struct ContentView: View {
    @ObservedObject var store: SessionStore

    var body: some View {
        NavigationStack {
            Group {
                if store.connected {
                    RemoteBrowserView(store: store)
                } else {
                    ConnectionView(store: store)
                }
            }
        }
        .alert("GhostFTP", isPresented: Binding(
            get: { store.errorMessage != nil },
            set: { if !$0 { store.errorMessage = nil } }
        )) {
            Button("OK", role: .cancel) { store.errorMessage = nil }
        } message: {
            Text(store.errorMessage ?? "Unknown error.")
        }
    }
}
