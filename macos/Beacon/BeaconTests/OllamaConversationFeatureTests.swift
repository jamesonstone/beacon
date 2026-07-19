import XCTest
@testable import Beacon

extension OllamaFeatureTests {
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

    func testLongConversationKeepsVisibleTranscriptAndBoundsRequestHistory() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: ""),
            response: OllamaChatResponse(model: "alpha:latest", content: "Latest answer")
        )
        let state = makeState(client: client)
        await state.refreshOllamaModels()
        state.notesAssistantMessages = (0 ..< 64).flatMap { turn in
            [
                NotesAssistantMessage(role: .user, content: "question \(turn)"),
                NotesAssistantMessage(role: .assistant, content: "answer \(turn)"),
            ]
        }
        state.notesAssistantPrompt = "latest question"

        await state.sendNotesAssistantPrompt()

        XCTAssertEqual(state.notesAssistantMessages.count, 130)
        XCTAssertEqual(state.notesAssistantMessages.first?.content, "question 0")
        XCTAssertEqual(state.notesAssistantMessages.last?.content, "Latest answer")
        XCTAssertEqual(state.ollamaNotice, NotesAssistantRequestHistory.truncationNotice)
        let chats = await client.chats
        XCTAssertEqual(chats.first?.messages.count, 127)
        XCTAssertEqual(chats.first?.messages.first?.content, "question 1")
        XCTAssertEqual(chats.first?.messages.last?.content, "latest question")
    }

    func testRequestHistoryDropsOldestTurnsToStayWithinByteLimit() {
        let largeAnswer = String(repeating: "a", count: 700_000)
        var transcript = (0 ..< 3).flatMap { turn in
            [
                NotesAssistantMessage(role: .user, content: "question \(turn)"),
                NotesAssistantMessage(role: .assistant, content: largeAnswer),
            ]
        }
        transcript.append(NotesAssistantMessage(role: .user, content: "latest question"))

        let request = NotesAssistantRequestHistory.boundedMessages(from: transcript)

        XCTAssertEqual(request.first?.content, "question 1")
        XCTAssertEqual(request.last?.content, "latest question")
        XCTAssertLessThanOrEqual(
            request.reduce(0) { $0 + $1.content.utf8.count },
            NotesAssistantRequestHistory.maxByteCount
        )
    }

    func testOversizedUserMessageIsRejectedBeforeSending() async {
        let client = OllamaClientStub(
            status: OllamaStatus(models: [model("alpha:latest")], configuredModel: "")
        )
        let state = makeState(client: client)
        await state.refreshOllamaModels()
        state.notesAssistantPrompt = String(
            repeating: "a",
            count: NotesAssistantRequestHistory.maxUserMessageByteCount + 1
        )

        await state.sendNotesAssistantPrompt()

        XCTAssertTrue(state.notesAssistantMessages.isEmpty)
        XCTAssertFalse(state.notesAssistantPrompt.isEmpty)
        XCTAssertEqual(
            state.ollamaError,
            "Prompt exceeds the \(NotesAssistantRequestHistory.maxUserMessageByteCount)-byte limit."
        )
        let chats = await client.chats
        XCTAssertTrue(chats.isEmpty)
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
}
