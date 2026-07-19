import XCTest
@testable import Beacon

actor OllamaClientStub: OllamaClientProtocol {
    let status: OllamaStatus
    let response: OllamaChatResponse
    let chatDelay: Duration?
    let failingChatCall: Int?
    private(set) var chats: [(model: String, context: String, messages: [OllamaChatMessage])] = []
    private(set) var defaults: [String] = []

    init(
        status: OllamaStatus,
        response: OllamaChatResponse = OllamaChatResponse(model: "alpha:latest", content: "Answer"),
        chatDelay: Duration? = nil,
        failingChatCall: Int? = nil
    ) {
        self.status = status
        self.response = response
        self.chatDelay = chatDelay
        self.failingChatCall = failingChatCall
    }

    func ollamaStatus() async throws -> OllamaStatus { status }

    func setOllamaDefaultModel(_ model: String) async throws -> String {
        defaults.append(model)
        return model
    }

    func ollamaChat(
        model: String,
        context: String,
        messages: [OllamaChatMessage]
    ) async throws -> OllamaChatResponse {
        chats.append((model, context, messages))
        if let chatDelay {
            try await Task.sleep(for: chatDelay)
        }
        if let failingChatCall, chats.count == failingChatCall {
            throw TestError.failed
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

    func testConversationPanelIsLargerAndStaysInsideBeaconBounds() {
        for surface in [DashboardSurface.menu, .window] {
            let available = CGSize(width: 700, height: 760)
            let compact = NotesAssistantPresentation.panelSize(in: available, surface: surface)
            let conversation = NotesAssistantPresentation.conversationPanelSize(in: available, surface: surface)
            XCTAssertGreaterThan(conversation.width, compact.width)
            XCTAssertGreaterThan(conversation.height, compact.height)
            XCTAssertLessThanOrEqual(conversation.width, available.width - 24)
            XCTAssertLessThanOrEqual(conversation.height, available.height - 24)
        }
        XCTAssertEqual(NotesAssistantPresentation.buttonSymbol, "dot.radiowaves.left.and.right")
        XCTAssertTrue(NotesAssistantPresentation.shouldPrepareSession(currentMode: nil))
        XCTAssertFalse(NotesAssistantPresentation.shouldPrepareSession(currentMode: .compact))
        XCTAssertFalse(NotesAssistantPresentation.shouldPrepareSession(currentMode: .conversation))
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

        state.prepareNotesAssistant(selection: "  selected\ntext  ", note: "whole note")
        await state.refreshOllamaModels()
        XCTAssertEqual(state.notesAssistantAttachment, "  selected\ntext  ")
        XCTAssertEqual(state.notesAssistantContextSource, .selection)
        XCTAssertEqual(state.ollamaSelectedModel, "zeta:latest")
        state.notesAssistantPrompt = "  summarize this  "

        await state.sendNotesAssistantPrompt()

        XCTAssertEqual(state.notesAssistantMessages.map(\.role), [.user, .assistant])
        XCTAssertEqual(state.notesAssistantMessages.map(\.content), ["summarize this", "Useful answer"])
        let chats = await client.chats
        XCTAssertEqual(chats.count, 1)
        XCTAssertEqual(chats.first?.model, "zeta:latest")
        XCTAssertEqual(chats.first?.context, "  selected\ntext  ")
        XCTAssertEqual(chats.first?.messages, [OllamaChatMessage(role: .user, content: "summarize this")])
    }

    func testAppStateFallsBackToCurrentNoteAndCanSendWithoutContext() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: "")
        )
        let state = makeState(client: client)

        state.prepareNotesAssistant(selection: "", note: "draft with unsaved text")
        await state.refreshOllamaModels()
        XCTAssertEqual(state.notesAssistantAttachment, "draft with unsaved text")
        XCTAssertEqual(state.notesAssistantContextSource, .note)

        state.removeNotesAssistantContext()
        state.notesAssistantPrompt = "brainstorm"
        XCTAssertNil(state.notesAssistantContextSource)
        XCTAssertTrue(state.canSendNotesAssistantPrompt)

        await state.sendNotesAssistantPrompt()

        let chats = await client.chats
        XCTAssertEqual(chats.first?.context, "")
        XCTAssertEqual(chats.first?.messages, [OllamaChatMessage(role: .user, content: "brainstorm")])
    }

    func testAppStatePreservesEveryTurnAndSendsCompleteConversation() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: ""),
            response: OllamaChatResponse(model: "alpha:latest", content: "A useful answer")
        )
        let state = makeState(client: client)
        state.prepareNotesAssistant(selection: "", note: "whole note")
        await state.refreshOllamaModels()

        state.notesAssistantPrompt = "first question"
        await state.sendNotesAssistantPrompt()
        state.notesAssistantPrompt = "follow up"
        await state.sendNotesAssistantPrompt()

        XCTAssertEqual(state.notesAssistantMessages.map(\.role), [.user, .assistant, .user, .assistant])
        XCTAssertEqual(
            state.notesAssistantMessages.map(\.content),
            ["first question", "A useful answer", "follow up", "A useful answer"]
        )
        XCTAssertEqual(state.notesAssistantPrompt, "")
        let chats = await client.chats
        XCTAssertEqual(chats.count, 2)
        XCTAssertEqual(
            chats[1].messages,
            [
                OllamaChatMessage(role: .user, content: "first question"),
                OllamaChatMessage(role: .assistant, content: "A useful answer"),
                OllamaChatMessage(role: .user, content: "follow up"),
            ]
        )
    }

    func testSecondSurfaceChangesPresentationWithoutResettingSharedSession() {
        let state = makeState(client: OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: "")
        ))

        XCTAssertTrue(state.presentNotesAssistant(.compact, selection: "selection", note: "first note"))
        state.notesAssistantPrompt = "unsent follow up"
        state.notesAssistantMessages = [
            NotesAssistantMessage(role: .user, content: "first question"),
            NotesAssistantMessage(role: .assistant, content: "first answer", model: "alpha:latest"),
        ]

        XCTAssertFalse(state.presentNotesAssistant(.conversation, selection: "new selection", note: "new note"))
        XCTAssertEqual(state.notesAssistantMode, .conversation)
        XCTAssertEqual(state.notesAssistantAttachment, "selection")
        XCTAssertEqual(state.notesAssistantPrompt, "unsent follow up")
        XCTAssertEqual(state.notesAssistantMessages.map(\.content), ["first question", "first answer"])

        state.dismissNotesAssistant()
        XCTAssertNil(state.notesAssistantMode)
        XCTAssertEqual(state.notesAssistantPrompt, "")
        XCTAssertTrue(state.notesAssistantMessages.isEmpty)
    }

    func testFailedFollowUpPreservesHistoryAndRestoresUnsentPrompt() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: ""),
            response: OllamaChatResponse(model: "alpha:latest", content: "First answer"),
            failingChatCall: 2
        )
        let state = makeState(client: client)
        state.prepareNotesAssistant(selection: "", note: "whole note")
        await state.refreshOllamaModels()

        state.notesAssistantPrompt = "first question"
        await state.sendNotesAssistantPrompt()
        state.notesAssistantPrompt = "retry this follow up"
        await state.sendNotesAssistantPrompt()

        XCTAssertEqual(state.notesAssistantMessages.map(\.content), ["first question", "First answer"])
        XCTAssertEqual(state.notesAssistantPrompt, "retry this follow up")
        XCTAssertNotNil(state.ollamaError)
        let chats = await client.chats
        XCTAssertEqual(chats[1].messages.map(\.content), ["first question", "First answer", "retry this follow up"])
    }

    func testDismissResetsAssistantAndIgnoresLateResponse() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: ""),
            chatDelay: .milliseconds(40)
        )
        let state = makeState(client: client)
        state.prepareNotesAssistant(selection: "", note: "whole note")
        await state.refreshOllamaModels()
        state.notesAssistantPrompt = "summarize"

        let send = Task { await state.sendNotesAssistantPrompt() }
        while !state.isSendingOllamaPrompt {
            await Task.yield()
        }
        state.dismissNotesAssistant()
        await send.value

        XCTAssertEqual(state.notesAssistantAttachment, "")
        XCTAssertEqual(state.notesAssistantPrompt, "")
        XCTAssertTrue(state.notesAssistantMessages.isEmpty)
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
