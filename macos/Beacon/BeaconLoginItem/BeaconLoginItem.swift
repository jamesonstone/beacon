import AppKit

@main
enum BeaconLoginItem {
    static func main() {
        let helperURL = Bundle.main.bundleURL
        let mainApplicationURL = helperURL
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()

        let configuration = NSWorkspace.OpenConfiguration()
        configuration.activates = false
        configuration.arguments = ["--login"]

        var completed = false
        NSWorkspace.shared.openApplication(
            at: mainApplicationURL,
            configuration: configuration
        ) { _, error in
            if let error {
                NSLog("Beacon login item could not open the main application: %@", error.localizedDescription)
            }
            completed = true
        }

        let deadline = Date().addingTimeInterval(10)
        while !completed, Date() < deadline {
            RunLoop.current.run(until: Date().addingTimeInterval(0.05))
        }
    }
}
