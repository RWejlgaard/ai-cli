import Foundation
import ArgumentParser
import OpenAI



@main
struct AI: AsyncParsableCommand {
    static var configuration = CommandConfiguration(
        commandName: "ai",
        abstract: "A command-line tool for interacting with OpenAI's GPT-4o model.",
        version: "1.0.0"
        // subcommands: [Models.self]
    )

    @Option(name: .shortAndLong, help: "The API key for OpenAI. (default: $OPENAI_API_KEY environment variable.)")
    var apiKey: String?

    @Option(name: .shortAndLong, help: "The model to use.")
    var model: String = "gpt-4o"

    @Option(name: .long, help: "Override the system prompt")
    var systemPrompt: String = "you're a helpful assistant"

    @Option(name: .shortAndLong, help: "Single message to send")
    var singleMessage: String?

    func run() async throws {
        guard let apiKey = apiKey ?? ProcessInfo.processInfo.environment["OPENAI_API_KEY"] else {
            throw ValidationError("API key is required.")
        }

        if singleMessage != nil {
            await oneShot(apiKey: apiKey, message: singleMessage ?? "", model: model, systemPrompt: systemPrompt)
            return
        }
        await chat(apiKey: apiKey, model: model, systemPrompt: systemPrompt)
    }
}