import Combine
import Foundation
import Network

@MainActor
final class EndpointProbe: ObservableObject {
    enum State: Equatable {
        case idle
        case checking
        case reachable
        case failed(String)
    }

    @Published private(set) var state: State = .idle

    private var connection: NWConnection?
    private var timeoutWorkItem: DispatchWorkItem?

    func check(_ profile: ConnectionProfile) {
        cancel()
        state = .checking

        guard let nwPort = NWEndpoint.Port(rawValue: profile.port) else {
            state = .failed("The selected port is invalid.")
            return
        }

        let connection = NWConnection(
            host: NWEndpoint.Host(profile.host),
            port: nwPort,
            using: .tcp
        )
        self.connection = connection

        let timeout = DispatchWorkItem { [weak self, weak connection] in
            guard let self, let connection, self.connection === connection else { return }
            connection.cancel()
            self.connection = nil
            self.state = .failed("The server did not accept a TCP connection before the timeout.")
        }
        timeoutWorkItem = timeout
        DispatchQueue.main.asyncAfter(deadline: .now() + 6, execute: timeout)

        connection.stateUpdateHandler = { [weak self, weak connection] newState in
            DispatchQueue.main.async {
                guard let self, let connection, self.connection === connection else { return }

                switch newState {
                case .ready:
                    self.timeoutWorkItem?.cancel()
                    self.timeoutWorkItem = nil
                    connection.cancel()
                    self.connection = nil
                    self.state = .reachable
                case .failed(let error):
                    self.timeoutWorkItem?.cancel()
                    self.timeoutWorkItem = nil
                    connection.cancel()
                    self.connection = nil
                    self.state = .failed(error.localizedDescription)
                case .cancelled:
                    break
                default:
                    break
                }
            }
        }

        connection.start(queue: DispatchQueue(label: "com.brendigo.ghostftp.macos.endpoint-probe"))
    }

    func cancel() {
        timeoutWorkItem?.cancel()
        timeoutWorkItem = nil
        connection?.cancel()
        connection = nil
        if state == .checking {
            state = .idle
        }
    }
}
