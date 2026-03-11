import Foundation

protocol CLIServiceProtocol {
    func fetchStatus(completion: @escaping (Result<StatusState, Error>) -> Void)
    func run(_ command: SlackCommand, options: CommandOptions, completion: @escaping (Result<StatusState, Error>) -> Void)
    func login(completion: @escaping (Result<Void, Error>) -> Void)
}

final class CLIService: CLIServiceProtocol {
    private let decoder = JSONDecoder()
    private let queue = DispatchQueue(label: "slack-status.cli-service", qos: .userInitiated)

    func fetchStatus(completion: @escaping (Result<StatusState, Error>) -> Void) {
        runJSONCommand(.status, completion: completion)
    }

    func run(_ command: SlackCommand, options: CommandOptions = .none, completion: @escaping (Result<StatusState, Error>) -> Void) {
        runJSONCommand(command, options: options, completion: completion)
    }

    func login(completion: @escaping (Result<Void, Error>) -> Void) {
        queue.async {
            do {
                let executable = try Self.resolveExecutablePath()
                let process = Process()
                let stderr = Pipe()
                process.executableURL = URL(fileURLWithPath: executable)
                process.arguments = [SlackCommand.login.rawValue]
                process.standardError = stderr
                try process.run()
                process.waitUntilExit()

                if process.terminationStatus == 0 {
                    DispatchQueue.main.async { completion(.success(())) }
                } else {
                    let errorData = stderr.fileHandleForReading.readDataToEndOfFile()
                    let message = String(data: errorData, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
                    DispatchQueue.main.async {
                        completion(.failure(BackendError.processFailure(message: message?.isEmpty == false ? message! : "slack-status login exited with code \(process.terminationStatus).")))
                    }
                }
            } catch {
                DispatchQueue.main.async { completion(.failure(error)) }
            }
        }
    }

    private func runJSONCommand(_ command: SlackCommand, options: CommandOptions = .none, completion: @escaping (Result<StatusState, Error>) -> Void) {
        queue.async {
            do {
                let executable = try Self.resolveExecutablePath()
                let output = try Self.execute(executable: executable, arguments: Self.arguments(for: command, options: options))
                let jsonOutput = try Self.extractJSONObject(from: output)
                let response = try self.decoder.decode(CommandResponse.self, from: jsonOutput)
                guard response.ok else {
                    throw BackendError.processFailure(message: "slack-status reported an unsuccessful \(command.rawValue) command.")
                }
                let statusState = response.state ?? StatusState(
                    currentStatus: CurrentStatus(
                        command: command.rawValue,
                        text: nil,
                        emoji: nil,
                        statusExpiresAt: nil,
                        willReturnTo: nil,
                        source: nil
                    ),
                    workerScheduled: false,
                    workerPID: nil,
                    startAvailableToday: true,
                    lastStartAt: nil,
                    lastStartDay: nil,
                    updatedAt: nil
                )
                DispatchQueue.main.async { completion(.success(statusState)) }
            } catch {
                DispatchQueue.main.async { completion(.failure(error)) }
            }
        }
    }

    private static func resolveExecutablePath() throws -> String {
        let candidates = [
            "/usr/local/bin/slack-status",
            "/opt/homebrew/bin/slack-status",
            Bundle.main.bundleURL
                .appendingPathComponent("Contents/Resources/slack-status")
                .path
        ]

        for candidate in candidates where FileManager.default.isExecutableFile(atPath: candidate) {
            return candidate
        }

        throw BackendError.binaryNotFound
    }

    private static func execute(executable: String, arguments: [String]) throws -> Data {
        let process = Process()
        let stdout = Pipe()
        let stderr = Pipe()

        process.executableURL = URL(fileURLWithPath: executable)
        process.arguments = arguments
        process.standardOutput = stdout
        process.standardError = stderr

        try process.run()
        process.waitUntilExit()

        let output = stdout.fileHandleForReading.readDataToEndOfFile()
        let errorData = stderr.fileHandleForReading.readDataToEndOfFile()

        guard process.terminationStatus == 0 else {
            let message = String(data: errorData, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
            throw BackendError.processFailure(message: message?.isEmpty == false ? message! : "slack-status exited with code \(process.terminationStatus).")
        }

        guard !output.isEmpty else {
            throw BackendError.invalidOutput("empty stdout")
        }

        return output
    }

    private static func arguments(for command: SlackCommand, options: CommandOptions) -> [String] {
        var arguments = ["--json"]

        if let until = options.until {
            arguments.append(contentsOf: ["--until", ISO8601DateFormatter.commandUntil.string(from: until)])
        }

        arguments.append(command.rawValue)

        return arguments
    }

    private static func extractJSONObject(from output: Data) throws -> Data {
        guard let text = String(data: output, encoding: .utf8) else {
            throw BackendError.invalidOutput("stdout was not valid UTF-8")
        }

        guard let jsonStartIndex = text.firstIndex(of: "{" ) else {
            throw BackendError.invalidOutput("missing JSON object in stdout")
        }

        let jsonText = text[jsonStartIndex...].trimmingCharacters(in: .whitespacesAndNewlines)
        guard let jsonData = jsonText.data(using: .utf8) else {
            throw BackendError.invalidOutput("failed to convert extracted JSON back to data")
        }

        return jsonData
    }
}

private extension ISO8601DateFormatter {
    static let commandUntil: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        return formatter
    }()
}
