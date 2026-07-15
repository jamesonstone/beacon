import AppKit
import SwiftUI

@main
struct BeaconApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @StateObject private var state = BeaconApplicationModel.shared.state
    @StateObject private var loginItem = BeaconApplicationModel.shared.loginItem

    var body: some Scene {
        MenuBarExtra {
            MenuView(
                state: state,
                loginItem: loginItem,
                surface: .menu,
                openDashboard: { BeaconApplicationModel.shared.showDashboard() }
            )
        } label: {
            BeaconMenuBarLabel(inProgressCount: state.inProgressCount)
        }
        .menuBarExtraStyle(.window)
    }
}

struct BeaconMenuBarLabel: View {
    let inProgressCount: Int

    var body: some View {
        Image(nsImage: BeaconMenuBarIconRenderer.image(count: inProgressCount))
            .renderingMode(.original)
            .accessibilityElement(children: .ignore)
            .accessibilityLabel(BeaconMenuBarPresentation.accessibilityText(inProgressCount))
    }
}

@MainActor
enum BeaconMenuBarIconRenderer {
    static let canvasHeight: CGFloat = 18

    static func image(count: Int) -> NSImage {
        let domeWidth = BeaconMenuBarPresentation.domeWidth(count)
        let size = NSSize(width: domeWidth + 6, height: canvasHeight)
        let image = NSImage(size: size, flipped: false) { rect in
            drawRays(in: rect, domeWidth: domeWidth)
            drawDome(in: rect, domeWidth: domeWidth)
            drawCount(count, in: rect)
            return true
        }
        image.isTemplate = false
        return image
    }

    private static func drawRays(in rect: NSRect, domeWidth: CGFloat) {
        let centerX = rect.midX
        let domeMinX = centerX - domeWidth / 2
        let domeMaxX = centerX + domeWidth / 2
        let rays = NSBezierPath()
        rays.move(to: NSPoint(x: centerX, y: 14.2))
        rays.line(to: NSPoint(x: centerX, y: 17))
        rays.move(to: NSPoint(x: domeMinX + 1.7, y: 13.2))
        rays.line(to: NSPoint(x: domeMinX - 0.4, y: 15.4))
        rays.move(to: NSPoint(x: domeMaxX - 1.7, y: 13.2))
        rays.line(to: NSPoint(x: domeMaxX + 0.4, y: 15.4))
        rays.lineWidth = 1.5
        rays.lineCapStyle = .round
        NSColor(BeaconPalette.cyan).setStroke()
        rays.stroke()

        let centerRay = NSBezierPath()
        centerRay.move(to: NSPoint(x: centerX, y: 14.2))
        centerRay.line(to: NSPoint(x: centerX, y: 17))
        centerRay.lineWidth = 1.5
        centerRay.lineCapStyle = .round
        NSColor(BeaconPalette.gold).setStroke()
        centerRay.stroke()
    }

    private static func drawDome(in rect: NSRect, domeWidth: CGFloat) {
        let centerX = rect.midX
        let minX = centerX - domeWidth / 2
        let maxX = centerX + domeWidth / 2
        let bottomY: CGFloat = 3
        let shoulderY: CGFloat = 7.4
        let topY: CGFloat = 13.8
        let dome = NSBezierPath()
        dome.move(to: NSPoint(x: minX + 1, y: bottomY))
        dome.line(to: NSPoint(x: minX + 2, y: shoulderY))
        dome.curve(
            to: NSPoint(x: centerX, y: topY),
            controlPoint1: NSPoint(x: minX + 2, y: topY - 1.6),
            controlPoint2: NSPoint(x: centerX - domeWidth * 0.22, y: topY)
        )
        dome.curve(
            to: NSPoint(x: maxX - 2, y: shoulderY),
            controlPoint1: NSPoint(x: centerX + domeWidth * 0.22, y: topY),
            controlPoint2: NSPoint(x: maxX - 2, y: topY - 1.6)
        )
        dome.line(to: NSPoint(x: maxX - 1, y: bottomY))
        dome.close()

        NSGraphicsContext.saveGraphicsState()
        let shadow = NSShadow()
        shadow.shadowColor = NSColor(BeaconPalette.cyan).withAlphaComponent(0.7)
        shadow.shadowBlurRadius = 2
        shadow.shadowOffset = .zero
        shadow.set()
        NSGradient(colors: [NSColor(BeaconPalette.gold), NSColor(BeaconPalette.coral)])?
            .draw(in: dome, angle: 90)
        NSGraphicsContext.restoreGraphicsState()

        let base = NSBezierPath(
            roundedRect: NSRect(x: minX - 1, y: 1.4, width: domeWidth + 2, height: 1.8),
            xRadius: 0.9,
            yRadius: 0.9
        )
        NSColor(BeaconPalette.gold).setFill()
        base.fill()
    }

    private static func drawCount(_ count: Int, in rect: NSRect) {
        let displayCount = BeaconMenuBarPresentation.displayCount(count)
        let font = NSFont.monospacedDigitSystemFont(
            ofSize: BeaconMenuBarPresentation.countFontSize(count),
            weight: .heavy
        )
        let attributes: [NSAttributedString.Key: Any] = [
            .font: font,
            .foregroundColor: NSColor(red: 0.04, green: 0.03, blue: 0.12, alpha: 1),
        ]
        let size = displayCount.size(withAttributes: attributes)
        displayCount.draw(
            at: NSPoint(x: rect.midX - size.width / 2, y: 3.5),
            withAttributes: attributes
        )
    }
}

enum BeaconMenuBarPresentation {
    static func displayCount(_ count: Int) -> String {
        count > 99 ? "99+" : String(max(0, count))
    }

    static func domeWidth(_ count: Int) -> CGFloat {
        switch displayCount(count).count {
        case 1: 14
        case 2: 18
        default: 24
        }
    }

    static func countFontSize(_ count: Int) -> CGFloat {
        switch displayCount(count).count {
        case 1: 9
        case 2: 8
        default: 6.5
        }
    }

    static func accessibilityText(_ count: Int) -> String {
        if count <= 0 {
            return "Beacon, no items in progress"
        }
        return "Beacon, \(count) items in progress"
    }
}

enum BeaconPalette {
    static let cyan = Color(red: 0.20, green: 0.91, blue: 1.0)
    static let mint = Color(red: 0.42, green: 1.0, blue: 0.76)
    static let lavender = Color(red: 0.70, green: 0.58, blue: 1.0)
    static let pink = Color(red: 1.0, green: 0.36, blue: 0.76)
    static let coral = Color(red: 1.0, green: 0.47, blue: 0.56)
    static let gold = Color(red: 1.0, green: 0.82, blue: 0.46)

    static let neonGradient = LinearGradient(
        colors: [cyan, lavender, pink],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static let panelBackground = LinearGradient(
        colors: [
            cyan.opacity(0.055),
            lavender.opacity(0.045),
            pink.opacity(0.035),
        ],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static let switcherBackground = LinearGradient(
        colors: [
            Color(red: 0.035, green: 0.028, blue: 0.055),
            Color(red: 0.075, green: 0.045, blue: 0.090),
        ],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static func softGradient(_ accent: Color) -> LinearGradient {
        LinearGradient(
            colors: [accent.opacity(0.18), lavender.opacity(0.09), cyan.opacity(0.035)],
            startPoint: .topLeading,
            endPoint: .bottomTrailing
        )
    }

    static func borderGradient(_ accent: Color) -> LinearGradient {
        LinearGradient(
            colors: [accent.opacity(0.8), lavender.opacity(0.38), cyan.opacity(0.18)],
            startPoint: .leading,
            endPoint: .trailing
        )
    }
}
