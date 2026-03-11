import AppKit

final class PopoverViewController: NSViewController {
    private let viewModel: StatusViewModel
    private let onQuit: () -> Void

    private let titleLabel = NSTextField(labelWithString: "Slack Status")
    private let statusIconView = NSImageView()
    private let statusLabel = NSTextField(wrappingLabelWithString: "Loading…")
    private let detailLabel = NSTextField(wrappingLabelWithString: "")
    private let errorLabel = NSTextField(wrappingLabelWithString: "")
    private let spinner = NSProgressIndicator()
    private let actionsLabel = NSTextField(labelWithString: "Actions")

    private lazy var startButton = makeButton(title: "Start", command: .start)
    private lazy var workButton = makeButton(title: "Work", command: .work)
    private lazy var lunchButton = makeButton(title: "Lunch", command: .lunch)
    private lazy var clearButton = makeButton(title: "Clear", command: .clear)
    private lazy var refreshButton = makeButton(title: "Refresh", command: .status)
    private lazy var loginButton = makeButton(title: "Login", command: .login)
    private let startTimePicker = makeTimePicker()
    private let workTimePicker = makeTimePicker()
    private let lunchTimePicker = makeTimePicker()
    private lazy var quitButton: NSButton = {
        let button = NSButton(title: "Quit", target: self, action: #selector(handleQuit))
        button.bezelStyle = .rounded
        return button
    }()

    init(viewModel: StatusViewModel, onQuit: @escaping () -> Void) {
        self.viewModel = viewModel
        self.onQuit = onQuit
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func loadView() {
        view = NSView()
        buildUI()
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        bindViewModel()
    }

    private func buildUI() {
        titleLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        statusIconView.contentTintColor = .secondaryLabelColor
        statusIconView.imageScaling = .scaleProportionallyDown
        statusIconView.setContentHuggingPriority(.required, for: .horizontal)
        statusLabel.font = .systemFont(ofSize: 14, weight: .medium)
        statusLabel.maximumNumberOfLines = 2
        detailLabel.textColor = .secondaryLabelColor
        detailLabel.maximumNumberOfLines = 3
        errorLabel.textColor = .systemRed
        errorLabel.maximumNumberOfLines = 0
        errorLabel.isHidden = true
        actionsLabel.font = .systemFont(ofSize: 12, weight: .semibold)
        actionsLabel.textColor = .secondaryLabelColor

        spinner.style = .spinning
        spinner.controlSize = .small
        spinner.isDisplayedWhenStopped = false

        startTimePicker.dateValue = TimeSelectionDefaults.defaultStartUntil()
        workTimePicker.dateValue = TimeSelectionDefaults.defaultWorkUntil()
        lunchTimePicker.dateValue = TimeSelectionDefaults.defaultLunchUntil()

        let startRow = makeTimedActionRow(button: startButton, picker: startTimePicker)
        let workRow = makeTimedActionRow(button: workButton, picker: workTimePicker)
        let lunchRow = makeTimedActionRow(button: lunchButton, picker: lunchTimePicker)
        let clearRow = makeSingleActionRow(button: clearButton)
        let refreshRow = makeSingleActionRow(button: refreshButton)
        let loginRow = makeSingleActionRow(button: loginButton)
        let quitRow = makeSingleActionRow(button: quitButton)

        let headerRow = NSStackView(views: [titleLabel, spinner])
        headerRow.orientation = .horizontal
        headerRow.alignment = .centerY
        headerRow.spacing = 8

        let statusRow = NSStackView(views: [statusIconView, statusLabel])
        statusRow.orientation = .horizontal
        statusRow.alignment = .firstBaseline
        statusRow.spacing = 8

        let actionsStack = NSStackView(views: [startRow, workRow, lunchRow, clearRow, refreshRow, loginRow, quitRow])
        actionsStack.orientation = .vertical
        actionsStack.spacing = 8

        let stack = NSStackView(views: [headerRow, statusRow, detailLabel, errorLabel, actionsLabel, actionsStack])
        stack.orientation = .vertical
        stack.spacing = 12
        stack.translatesAutoresizingMaskIntoConstraints = false

        view.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: view.topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: view.bottomAnchor, constant: -16),
            view.widthAnchor.constraint(equalToConstant: 360),
            statusIconView.widthAnchor.constraint(equalToConstant: 18),
            statusIconView.heightAnchor.constraint(equalToConstant: 18)
        ])
    }

    private func bindViewModel() {
        viewModel.onChange = { [weak self] state in
            self?.render(state)
        }
        render(viewModel.state)
    }

    private func render(_ state: ViewState) {
        switch state {
        case .idle:
            spinner.stopAnimation(nil)
            setButtonsEnabled(true, startAvailableToday: true)
            statusIconView.image = makeStatusImage(symbolName: "bubble.left.and.bubble.right")
            statusLabel.stringValue = "Slack status idle"
            detailLabel.stringValue = "Use an action to fetch or update status."
            errorLabel.isHidden = true
        case .loading:
            spinner.startAnimation(nil)
            setButtonsEnabled(false, startAvailableToday: false)
            statusIconView.image = makeStatusImage(symbolName: "arrow.triangle.2.circlepath")
            statusLabel.stringValue = "Working…"
            detailLabel.stringValue = "Contacting the CLI backend."
            errorLabel.isHidden = true
        case .loaded(let status):
            spinner.stopAnimation(nil)
            setButtonsEnabled(true, startAvailableToday: status.startAvailableToday)
            statusIconView.image = makeStatusImage(symbolName: status.statusSymbolName)
            statusLabel.stringValue = status.titleText
            detailLabel.stringValue = status.accessibilitySummaryText
            errorLabel.isHidden = true
        case .failure(let message):
            spinner.stopAnimation(nil)
            setButtonsEnabled(true, startAvailableToday: false)
            statusIconView.image = makeStatusImage(symbolName: "exclamationmark.triangle")
            statusLabel.stringValue = "Backend unavailable"
            detailLabel.stringValue = "The menu bar app could not complete the requested command."
            errorLabel.stringValue = message
            errorLabel.isHidden = false
        }
    }

    private func setButtonsEnabled(_ enabled: Bool, startAvailableToday: Bool) {
        startButton.isEnabled = enabled && startAvailableToday
        startTimePicker.isEnabled = enabled && startAvailableToday
        [workButton, lunchButton, clearButton, refreshButton, loginButton, quitButton].forEach { $0.isEnabled = enabled }
        [workTimePicker, lunchTimePicker].forEach { $0.isEnabled = enabled }
    }

    private func makeButton(title: String, command: SlackCommand) -> NSButton {
        let button = NSButton(title: title, target: self, action: #selector(handleCommand(_:)))
        button.bezelStyle = .rounded
        button.identifier = NSUserInterfaceItemIdentifier(command.rawValue)
        return button
    }

    private static func makeTimePicker() -> NSDatePicker {
        let picker = NSDatePicker()
        picker.datePickerElements = [.hourMinute]
        picker.datePickerStyle = .textFieldAndStepper
        picker.datePickerMode = .single
        picker.controlSize = .small
        picker.timeZone = .current
        picker.locale = .current
        picker.translatesAutoresizingMaskIntoConstraints = false
        return picker
    }

    private func makeTimedActionRow(button: NSButton, picker: NSDatePicker) -> NSStackView {
        button.setContentHuggingPriority(.required, for: .horizontal)
        picker.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        picker.widthAnchor.constraint(greaterThanOrEqualToConstant: 120).isActive = true
        let row = NSStackView(views: [button, picker])
        row.orientation = .horizontal
        row.alignment = .centerY
        row.spacing = 8
        row.distribution = .fill
        return row
    }

    private func makeSingleActionRow(button: NSButton) -> NSStackView {
        let row = NSStackView(views: [button])
        row.orientation = .horizontal
        row.alignment = .centerY
        row.distribution = .fillEqually
        return row
    }

    @objc private func handleCommand(_ sender: NSButton) {
        guard let identifier = sender.identifier?.rawValue, let command = SlackCommand(rawValue: identifier) else {
            return
        }
        if command == .status {
            viewModel.refresh()
        } else {
            viewModel.perform(command, options: options(for: command))
        }
    }

    @objc private func handleQuit() {
        onQuit()
    }

    private func makeStatusImage(symbolName: String) -> NSImage? {
        let image = NSImage(systemSymbolName: symbolName, accessibilityDescription: nil)
        image?.isTemplate = true
        return image
    }

    private func options(for command: SlackCommand) -> CommandOptions {
        switch command {
        case .start:
            return CommandOptions(until: combinedDate(from: startTimePicker.dateValue))
        case .work:
            return CommandOptions(until: combinedDate(from: workTimePicker.dateValue))
        case .lunch:
            return CommandOptions(until: combinedDate(from: lunchTimePicker.dateValue))
        default:
            return .none
        }
    }

    private func combinedDate(from pickerDate: Date) -> Date {
        let calendar = Calendar.current
        let now = Date()
        let timeComponents = calendar.dateComponents([.hour, .minute], from: pickerDate)
        var candidateComponents = calendar.dateComponents([.year, .month, .day], from: now)
        candidateComponents.hour = timeComponents.hour
        candidateComponents.minute = timeComponents.minute
        candidateComponents.second = 0

        let todayCandidate = calendar.date(from: candidateComponents) ?? pickerDate
        if todayCandidate > now {
            return todayCandidate
        }

        return calendar.date(byAdding: .day, value: 1, to: todayCandidate) ?? todayCandidate.addingTimeInterval(24 * 60 * 60)
    }
}
