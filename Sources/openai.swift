import OpenAI
import Foundation

func modelsList(apiKey: String) async -> [String] {
    let openAI = OpenAI(apiToken: apiKey)

    do {
        let models = try await openAI.models()
        let filteredModels = models.data.filter { $0.id.hasPrefix("gpt") }
        return filteredModels.map { $0.id }
    } catch {
        print("Error: \(error)")
        return []
    }
}

func oneShot(apiKey: String, message: String, model: String, systemPrompt: String) async {
    let openAI = OpenAI(apiToken: apiKey)

    let query = ChatQuery(
        messages: [
            .init(role: .system, content: "you're a helpful assistant")!,
            .init(role: .user, content: message)!
        ], model: model
    )

    do {
        for try await result in openAI.chatsStream(query: query) {
            print(result.choices[0].delta.content ?? "", terminator: "")
        }
    } catch {
        print("Error: \(error)")
        // Handle the error here
    }
}

func chat(apiKey: String, model: String, systemPrompt: String) async {
    let openAI = OpenAI(apiToken: apiKey)

    var query = ChatQuery(
        messages: [
            .init(role: .system, content: "you're a helpful assistant")!,
        ], model: model
    )

    var messages = query.messages

    while true {
        // Get user input
        print("\u{001B}[32mYou:\u{001B}[0m ", terminator: "")
        let input = readLine() ?? ""

        messages = query.messages
        messages.append(.init(role: .user, content: input)!)
        query = ChatQuery(
            messages: messages,
            model: model
        )
        
        print("")
        print("\u{001B}[31mAssistant:\u{001B}[0m")
        
        var reply: String = ""
        do {
            for try await result in openAI.chatsStream(query: query) {
                print(result.choices[0].delta.content ?? "", terminator: "")
                reply += result.choices[0].delta.content ?? ""
            }
        } catch {
            print("Error: \(error)")
            // Handle the error here
        }
        
        print("")

        messages = query.messages
        messages.append(.init(role: .assistant, content: reply)!)
        query = ChatQuery(
            messages: messages,
            model: .gpt4_o
        )
    }
}