import AppKit
import ApplicationServices

func fail(_ message: String, status: Int32 = 1) -> Never {
    FileHandle.standardError.write(Data((message + "\n").utf8))
    exit(status)
}

func children(_ node: AXUIElement, _ attribute: String = kAXChildrenAttribute) -> [AXUIElement] {
    var value: CFTypeRef?
    guard AXUIElementCopyAttributeValue(node, attribute as CFString, &value) == .success else { return [] }
    return value as? [AXUIElement] ?? []
}

func find(_ node: AXUIElement, role: String, title: String? = nil, depth: Int = 12) -> AXUIElement? {
    guard depth >= 0 else { return nil }
    var actualRole: CFTypeRef?
    AXUIElementCopyAttributeValue(node, kAXRoleAttribute as CFString, &actualRole)
    if actualRole as? String == role {
        var actualTitle: CFTypeRef?
        AXUIElementCopyAttributeValue(node, kAXTitleAttribute as CFString, &actualTitle)
        if title == nil || actualTitle as? String == title { return node }
    }
    for child in children(node) {
        if let match = find(child, role: role, title: title, depth: depth - 1) { return match }
    }
    return nil
}

func waitFor(_ description: String, _ locate: () -> AXUIElement?) -> AXUIElement {
    for _ in 0..<100 {
        if let element = locate() { return element }
        Thread.sleep(forTimeInterval: 0.1)
    }
    fail("Timed out waiting for \(description)")
}

guard AXIsProcessTrusted(), CGPreflightPostEventAccess() else {
    fail("GitHub runner does not grant Accessibility and keyboard event access", status: 77)
}
guard CommandLine.arguments.count > 1 else { exit(0) }
guard CommandLine.arguments.count >= 3,
      let pid = Int32(CommandLine.arguments[1]),
      let process = NSRunningApplication(processIdentifier: pid) else {
    fail("Expected a running desktop process and a picker operation")
}

let app = AXUIElementCreateApplication(pid)

func pickerButton(_ title: String) -> AXUIElement? {
    for window in children(app, kAXWindowsAttribute) {
        if let sheet = find(window, role: kAXSheetRole),
           let button = find(sheet, role: kAXButtonRole, title: title) {
            return button
        }
    }
    return nil
}

func press(_ button: AXUIElement) {
    let result = AXUIElementPerformAction(button, kAXPressAction as CFString)
    guard result == .success else { fail("AXPress failed: \(result.rawValue)") }
}

func sendKey(_ code: CGKeyCode, flags: CGEventFlags = []) {
    guard NSWorkspace.shared.frontmostApplication?.processIdentifier == pid else {
        fail("Desktop fixture is not the frontmost application on the isolated runner")
    }
    for down in [true, false] {
        guard let event = CGEvent(keyboardEventSource: nil, virtualKey: code, keyDown: down) else {
            fail("Could not create keyboard event")
        }
        event.flags = flags
        event.post(tap: .cghidEventTap)
    }
}

switch CommandLine.arguments[2] {
case "cancel":
    press(waitFor("Cancel in the Open sheet") { pickerButton("Cancel") })
case "open":
    guard CommandLine.arguments.count == 4 else { fail("Expected an absolute directory path") }
    _ = waitFor("Open sheet") { pickerButton("Cancel") }
    process.activate(options: [])
    for _ in 0..<20 {
        if NSWorkspace.shared.frontmostApplication?.processIdentifier == pid { break }
        Thread.sleep(forTimeInterval: 0.1)
    }
    sendKey(5, flags: [.maskCommand, .maskShift]) // Command-Shift-G
    _ = waitFor("Go to Folder field") { find(app, role: kAXTextFieldRole) ?? children(app, kAXWindowsAttribute).lazy.compactMap { find($0, role: kAXTextFieldRole) }.first }
    NSPasteboard.general.clearContents()
    guard NSPasteboard.general.setString(CommandLine.arguments[3], forType: .string) else {
        fail("Could not set the fixture path on the runner pasteboard")
    }
    sendKey(9, flags: .maskCommand) // Paste
    Thread.sleep(forTimeInterval: 1)
    sendKey(36) // Return
    Thread.sleep(forTimeInterval: 0.5)
    press(waitFor("Open in the directory sheet") { pickerButton("Open") })
default:
    fail("Unknown picker operation")
}
