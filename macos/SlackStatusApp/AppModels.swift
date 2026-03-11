import Foundation

enum StatusPresentation {
    case cleared
    case work
    case lunch
    case unknown
}

enum SlackCommand: String, CaseIterable {
    case status
    case start
    case work
    case lunch
    case clear
    case login
}

struct CommandOptions {
    let until: Date?

    static let none = CommandOptions(until: nil)
}

enum TimeSelectionDefaults {
    static func defaultStartUntil(now: Date = Date(), calendar: Calendar = .current) -> Date {
        defaultWorkUntil(now: now, calendar: calendar)
    }

    static func defaultWorkUntil(now: Date = Date(), calendar: Calendar = .current) -> Date {
        var components = calendar.dateComponents([.year, .month, .day], from: now)
        components.hour = 18
        components.minute = 0
        components.second = 0

        let todayAtSix = calendar.date(from: components)
        if let todayAtSix, todayAtSix > now {
            return todayAtSix
        }

        if let nextDayAtSix = calendar.date(byAdding: .day, value: 1, to: todayAtSix ?? now) {
            return nextDayAtSix
        }

        return now.addingTimeInterval(24 * 60 * 60)
    }

    static func defaultLunchUntil(now: Date = Date()) -> Date {
        now.addingTimeInterval(60 * 60)
    }
}

struct CommandResponse: Decodable {
    let command: String
    let ok: Bool
    let state: StatusState?
}

struct CurrentStatus: Decodable {
    let command: String?
    let text: String?
    let emoji: String?
    let statusExpiresAt: String?
    let willReturnTo: String?
    let source: String?

    enum CodingKeys: String, CodingKey {
        case command
        case text
        case emoji
        case statusExpiresAt = "status_expires_at"
        case willReturnTo = "will_return_to"
        case source
    }
}

struct StatusState: Decodable {
    let currentStatus: CurrentStatus
    let workerScheduled: Bool
    let workerPID: Int?
    let startAvailableToday: Bool
    let lastStartAt: String?
    let lastStartDay: String?
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case currentStatus = "current_status"
        case workerScheduled = "worker_scheduled"
        case workerPID = "worker_pid"
        case startAvailableToday = "start_available_today"
        case lastStartAt = "last_start_at"
        case lastStartDay = "last_start_day"
        case updatedAt = "updated_at"
    }
}

enum BackendError: LocalizedError {
    case binaryNotFound
    case invalidOutput(String)
    case processFailure(message: String)

    var errorDescription: String? {
        switch self {
        case .binaryNotFound:
            return "The slack-status CLI binary could not be found. Install it first."
        case .invalidOutput(let message):
            return "Received invalid output from slack-status: \(message)"
        case .processFailure(let message):
            return message
        }
    }
}

enum ViewState {
    case idle
    case loading
    case loaded(StatusState)
    case failure(String)
}

extension StatusState {
    var presentation: StatusPresentation {
        switch currentStatus.command?.lowercased() {
        case "work", "start":
            return .work
        case "lunch":
            return .lunch
        case "clear":
            return .cleared
        case .none:
            return (currentStatus.text?.isEmpty == false || currentStatus.emoji?.isEmpty == false) ? .unknown : .cleared
        default:
            return .unknown
        }
    }

    var statusSymbolName: String {
        switch presentation {
        case .work:
            return "desktopcomputer"
        case .lunch:
            return "fork.knife"
        case .cleared:
            return "circle.slash"
        case .unknown:
            return "bubble.left.and.bubble.right"
        }
    }

    var menuBarSymbolName: String {
        switch presentation {
        case .work:
            return "desktopcomputer"
        case .lunch:
            return "fork.knife.circle.fill"
        case .cleared:
            return "bubble.left.and.bubble.right"
        case .unknown:
            return "bubble.left.and.bubble.right"
        }
    }

    var titleText: String {
        guard let text = currentStatus.text, !text.isEmpty else {
            return "No current status"
        }
        return text
    }

    var subtitleText: String {
        var parts: [String] = []

        if let command = currentStatus.command, !command.isEmpty {
            parts.append(commandLabel(for: command))
        }
        if let statusExpiresAt, !statusExpiresAt.isEmpty {
            parts.append("until \(friendlyDateTime(statusExpiresAt))")
        }
        if let willReturnTo = currentStatus.willReturnTo, !willReturnTo.isEmpty {
            parts.append("returns to \(friendlyDateTime(willReturnTo))")
        }
        if workerScheduled {
            parts.append("worker scheduled")
        }

        return parts.isEmpty ? "Slack status is currently cleared" : parts.joined(separator: " • ")
    }

    var accessibilitySummaryText: String {
        if let command = currentStatus.command, command.lowercased() == "lunch" {
            if let willReturnTo = currentStatus.willReturnTo, !willReturnTo.isEmpty {
                return "Back at \(friendlyReturnDateTime(willReturnTo))"
            }
            if let statusExpiresAt, !statusExpiresAt.isEmpty {
                return "Until \(friendlyReturnDateTime(statusExpiresAt))"
            }
        }

        if let statusExpiresAt, !statusExpiresAt.isEmpty {
            return "Until \(friendlyReturnDateTime(statusExpiresAt))"
        }

        return subtitleText
    }

    private var statusExpiresAt: String? {
        currentStatus.statusExpiresAt
    }

    private func commandLabel(for command: String) -> String {
        switch command.lowercased() {
        case "work", "start":
            return "Working"
        case "lunch":
            return "At lunch"
        case "clear":
            return "Status cleared"
        default:
            return command
        }
    }

    private func friendlyDateTime(_ value: String) -> String {
        guard let date = parseBackendDate(value) else {
            return value
        }

        return DateFormatter.statusFriendly.string(from: date)
    }

    private func friendlyReturnDateTime(_ value: String) -> String {
        guard let date = parseBackendDate(value) else {
            return value
        }

        return DateFormatter.statusVeryFriendly.string(from: date)
    }

    private func parseBackendDate(_ value: String) -> Date? {
        ISO8601DateFormatter.backendDateWithFractionalSeconds.date(from: value)
            ?? ISO8601DateFormatter.backendDate.date(from: value)
    }
}

private extension ISO8601DateFormatter {
    static let backendDateWithFractionalSeconds: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    static let backendDate: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()
}

private extension DateFormatter {
    static let statusFriendly: DateFormatter = {
        let formatter = DateFormatter()
        formatter.doesRelativeDateFormatting = true
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        return formatter
    }()

    static let statusVeryFriendly: DateFormatter = {
        let formatter = DateFormatter()
        formatter.doesRelativeDateFormatting = true
        formatter.dateStyle = .full
        formatter.timeStyle = .short
        return formatter
    }()
}
