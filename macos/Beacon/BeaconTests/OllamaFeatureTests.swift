import XCTest
@testable import Beacon

actor OllamaClientStub: OllamaClientProtocol {
    let status: OllamaStatus
    let response: OllamaChatResponse
    let chatDelay: Duration?
    private(set) var chats: [(model: String, context: String, prompt: String)] = []
    private(set) var defaults: [String] = []

    init(
        status: OllamaStatus,
        response: OllamaChatResponse = OllamaChatResponse(model: "alpha:latest", content: "Answer"),
        chatDelay: Duration? = nil
    ) {
        self.status = status
        self.response = response
        self.chatDelay = chatDelay
    }

    func ollamaStatus() async throws -> OllamaStatus { status }

    func setOllamaDefaultModel(_ model: String) async throws -> String {
        defaults.append(model)
        return model
    }

    func ollamaChat(
        model: String,
        context: String,
        prompt: String
    ) async throws -> OllamaChatResponse {
        chats.append((model, context, prompt))
        if let chatDelay {
            try await Task.sleep(for: chatDelay)
        }
        return response
    }
}

@MainActor
final class OllamaFeatureTests: XCTestCase {
    func testSelectionUsesExactNSStringRange() {
        let source = "first 🛰️ line\nsecond line"
        let range = (source as NSString).range(of: "🛰️ line\nsecond")

        XCTAssertEqual(
            LiveMarkdownSelection.text(in: source, range: range),
            "🛰️ line\nsecond"
        )
        XCTAssertEqual(
            LiveMarkdownSelection.text(in: source, range: NSRange(location: 0, length: 0)),
            ""
        )
        XCTAssertEqual(
            LiveMarkdownSelection.text(in: source, range: NSRange(location: 999, length: 1)),
            ""
        )
    }

    func testModelResolutionPrefersConfiguredThenStableFirst() {
        let models = [model("alpha:latest"), model("zeta:latest")]

        XCTAssertEqual(
            NotesAssistantPresentation.resolvedModel(configured: "zeta:latest", models: models),
            "zeta:latest"
        )
        XCTAssertEqual(
            NotesAssistantPresentation.resolvedModel(configured: "missing:latest", models: models),
            "alpha:latest"
        )
        XCTAssertEqual(
            NotesAssistantPresentation.resolvedModel(configured: "", models: []),
            ""
        )
    }

    func testContextResolutionUsesSelectionThenEntireCurrentNote() {
        XCTAssertEqual(
            NotesAssistantPresentation.context(selection: " exact ", note: "whole"),
            NotesAssistantContext(text: " exact ", source: .selection)
        )
        XCTAssertEqual(
            NotesAssistantPresentation.context(selection: "  ", note: "whole\nnote"),
            NotesAssistantContext(text: "whole\nnote", source: .note)
        )
        XCTAssertNil(NotesAssistantPresentation.context(selection: "", note: "\n"))
    }

    func testPanelSizeStaysInsideNotesBounds() {
        for surface in [DashboardSurface.menu, .window] {
            let available = CGSize(width: 400, height: 220)
            let size = NotesAssistantPresentation.panelSize(in: available, surface: surface)
            XCTAssertLessThanOrEqual(size.width, available.width - 24)
            XCTAssertLessThanOrEqual(size.height, available.height - 40)
        }
    }

    func testAppStateAttachesExactSelectionAndSendsOnePrompt() async {
        let status = OllamaStatus(
            models: [model("alpha:latest"), model("zeta:latest")],
            configuredModel: "zeta:latest"
        )
        let client = OllamaClientStub(
            status: status,
            response: OllamaChatResponse(model: "zeta:latest", content: "Useful answer")
        )
        let state = AppState(
            agent: ScriptedAgent(events: []),
            installer: nil,
            notesFallback: nil,
            repositorySyncFallback: nil,
            dependencyLimitsClient: nil,
            ollamaClient: client
        )

        await state.prepareNotesAssistant(selection: "  selected\ntext  ", note: "whole note")
        XCTAssertEqual(state.notesAssistantAttachment, "  selected\ntext  ")
        XCTAssertEqual(state.notesAssistantContextSource, .selection)
        XCTAssertEqual(state.ollamaSelectedModel, "zeta:latest")
        state.notesAssistantPrompt = "  summarize this  "

        await state.sendNotesAssistantPrompt()

        XCTAssertEqual(state.notesAssistantResponse?.content, "Useful answer")
        let chats = await client.chats
        XCTAssertEqual(chats.count, 1)
        XCTAssertEqual(chats.first?.model, "zeta:latest")
        XCTAssertEqual(chats.first?.context, "  selected\ntext  ")
        XCTAssertEqual(chats.first?.prompt, "summarize this")
    }

    func testAppStateFallsBackToCurrentNoteAndCanSendWithoutContext() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: "")
        )
        let state = makeState(client: client)

        await state.prepareNotesAssistant(selection: "", note: "draft with unsaved text")
        XCTAssertEqual(state.notesAssistantAttachment, "draft with unsaved text")
        XCTAssertEqual(state.notesAssistantContextSource, .note)

        state.removeNotesAssistantContext()
        state.notesAssistantPrompt = "brainstorm"
        XCTAssertNil(state.notesAssistantContextSource)
        XCTAssertTrue(state.canSendNotesAssistantPrompt)

        await state.sendNotesAssistantPrompt()

        let chats = await client.chats
        XCTAssertEqual(chats.first?.context, "")
        XCTAssertEqual(chats.first?.prompt, "brainstorm")
    }

    func testDismissResetsAssistantAndIgnoresLateResponse() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: ""),
            chatDelay: .milliseconds(40)
        )
        let state = makeState(client: client)
        await state.prepareNotesAssistant(selection: "", note: "whole note")
        state.notesAssistantPrompt = "summarize"

        let send = Task { await state.sendNotesAssistantPrompt() }
        while !state.isSendingOllamaPrompt {
            await Task.yield()
        }
        state.dismissNotesAssistant()
        await send.value

        XCTAssertEqual(state.notesAssistantAttachment, "")
        XCTAssertEqual(state.notesAssistantPrompt, "")
        XCTAssertNil(state.notesAssistantResponse)
        XCTAssertFalse(state.isSendingOllamaPrompt)
    }

    func testQuickSwitcherCommandIsDiscoverableAndRunsSharedAction() {
        var invoked = false
        let command = BeaconCommandItem.notesAssistant(
            detail: "Attach the entire current note",
            action: { invoked = true }
        )

        XCTAssertEqual(command.id, "notes-assistant")
        XCTAssertTrue(command.matches("AI"))
        XCTAssertTrue(command.matches("ollama"))
        XCTAssertTrue(command.matches("entire current note"))
        command.action()
        XCTAssertTrue(invoked)
    }

    func testUnavailableConfiguredModelFallsBackWithoutChangingDefault() async {
        let client = OllamaClientStub(
            status: OllamaStatus(
                models: [model("alpha:latest")],
                configuredModel: "missing:latest"
            )
        )
        let state = AppState(
            agent: ScriptedAgent(events: []),
            installer: nil,
            notesFallback: nil,
            repositorySyncFallback: nil,
            dependencyLimitsClient: nil,
            ollamaClient: client
        )

        await state.refreshOllamaModels()

        XCTAssertEqual(state.ollamaConfiguredModel, "missing:latest")
        XCTAssertEqual(state.ollamaSelectedModel, "alpha:latest")
        XCTAssertNotNil(state.ollamaNotice)
        let defaults = await client.defaults
        XCTAssertTrue(defaults.isEmpty)
    }

    func testSettingsPersistsOnlyAnInstalledModel() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: "")
        )
        let state = AppState(
            agent: ScriptedAgent(events: []),
            installer: nil,
            notesFallback: nil,
            repositorySyncFallback: nil,
            dependencyLimitsClient: nil,
            ollamaClient: client
        )
        await state.refreshOllamaModels()

        await state.setOllamaDefaultModel("alpha:latest")
        await state.setOllamaDefaultModel("missing:latest")

        let defaults = await client.defaults
        XCTAssertEqual(defaults, ["alpha:latest"])
        XCTAssertEqual(state.ollamaConfiguredModel, "alpha:latest")
        XCTAssertNotNil(state.ollamaError)
    }

    private func model(_ name: String) -> OllamaModel {
        OllamaModel(
            name: name,
            size: 42,
            details: OllamaModelDetails(
                format: "gguf",
                family: "test",
                parameterSize: "1B",
                quantizationLevel: "Q4"
            )
        )
    }

    private func makeState(client: OllamaClientProtocol) -> AppState {
        AppState(
            agent: ScriptedAgent(events: []),
            installer: nil,
            notesFallback: nil,
            repositorySyncFallback: nil,
            dependencyLimitsClient: nil,
            ollamaClient: client
        )
    }
}
