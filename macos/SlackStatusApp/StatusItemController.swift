import AppKit

final class StatusItemController: NSObject {
    private let statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    private let popover = NSPopover()
    private let app: NSApplication
    private let viewModel: StatusViewModel

    init(app: NSApplication, service: CLIServiceProtocol = CLIService()) {
        self.app = app
        self.viewModel = StatusViewModel(service: service)
        super.init()
        configureStatusItem()
        configurePopover()
        viewModel.refresh()
    }

    private func configureStatusItem() {
        if let button = statusItem.button {
            button.image = makeMenuBarImage(symbolName: "bubble.left.and.bubble.right")
            button.imagePosition = .imageLeading
            button.action = #selector(togglePopover(_:))
            button.target = self
        }

        viewModel.onStatusStateChange = { [weak self] status in
            self?.updateStatusItemIcon(for: status)
        }
    }

    private func configurePopover() {
        let viewController = PopoverViewController(viewModel: viewModel, onQuit: { [weak self] in
            self?.app.terminate(nil)
        })
        popover.contentViewController = viewController
        popover.behavior = .transient
        popover.animates = true
        popover.contentSize = NSSize(width: 320, height: 280)
    }

    @objc private func togglePopover(_ sender: AnyObject?) {
        guard let button = statusItem.button else { return }

        if popover.isShown {
            closePopover(sender)
        } else {
            popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
            popover.contentViewController?.view.window?.becomeKey()
            viewModel.refresh()
        }
    }

    private func closePopover(_ sender: Any?) {
        popover.performClose(sender)
    }

    private func updateStatusItemIcon(for status: StatusState?) {
        let symbolName = status?.menuBarSymbolName ?? "bubble.left.and.bubble.right"
        statusItem.button?.image = makeMenuBarImage(symbolName: symbolName)
    }

    private func makeMenuBarImage(symbolName: String) -> NSImage? {
        let image = NSImage(
            systemSymbolName: symbolName,
            accessibilityDescription: "Slack Status"
        )
        image?.isTemplate = true
        return image
    }
}
