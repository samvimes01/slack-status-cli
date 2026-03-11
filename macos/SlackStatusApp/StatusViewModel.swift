import Foundation

final class StatusViewModel {
    var onChange: ((ViewState) -> Void)?
    var onStatusStateChange: ((StatusState?) -> Void)?

    private var latestStatusState: StatusState?
    private var requestID: UInt = 0

    private let service: CLIServiceProtocol
    private(set) var state: ViewState = .idle {
        didSet {
            onChange?(state)
            onStatusStateChange?(displayedStatusState(for: state))
        }
    }

    init(service: CLIServiceProtocol) {
        self.service = service
    }

    func refresh() {
        let currentRequestID = beginRequest()
        state = .loading
        service.fetchStatus { [weak self] result in
            self?.consume(result, requestID: currentRequestID)
        }
    }

    func perform(_ command: SlackCommand) {
        perform(command, options: .none)
    }

    func perform(_ command: SlackCommand, options: CommandOptions) {
        if command == .login {
            let currentRequestID = beginRequest()
            state = .loading
            service.login { [weak self] result in
                guard let self else { return }
                guard self.isCurrentRequest(currentRequestID) else { return }
                switch result {
                case .success:
                    self.refresh()
                case .failure(let error):
                    self.state = .failure(error.localizedDescription)
                }
            }
            return
        }

        let currentRequestID = beginRequest()
        state = .loading
        service.run(command, options: options) { [weak self] result in
            self?.consume(result, requestID: currentRequestID)
        }
    }

    private func consume(_ result: Result<StatusState, Error>, requestID: UInt) {
        guard isCurrentRequest(requestID) else {
            return
        }

        switch result {
        case .success(let status):
            latestStatusState = status
            state = .loaded(status)
        case .failure(let error):
            state = .failure(error.localizedDescription)
        }
    }

    private func beginRequest() -> UInt {
        requestID += 1
        return requestID
    }

    private func isCurrentRequest(_ requestID: UInt) -> Bool {
        self.requestID == requestID
    }

    private func displayedStatusState(for state: ViewState) -> StatusState? {
        if case .loaded(let status) = state {
            return status
        }

        return latestStatusState
    }
}
