import AppKit
import SwiftUI

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItemController: StatusItemController?

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSLog("SlackStatusApp launch finished; creating status item controller")
        NSApp.setActivationPolicy(.accessory)
        statusItemController = StatusItemController(app: NSApp)
    }
}

@main
struct SlackStatusApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate

    var body: some Scene {
        Settings {
            EmptyView()
        }
    }
}
