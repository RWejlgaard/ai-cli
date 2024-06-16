// swift-tools-version: 6.0
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "ai",
    platforms: [
        .macOS(.v14)
    ],
    dependencies: [
        .package(url: "https://github.com/apple/swift-argument-parser", from: "1.3.0"),
        .package(url: "https://github.com/MacPaw/OpenAI.git", branch: "main")
    ],
    targets: [
        .executableTarget(
            name: "ai",
            dependencies: [
                .product(name: "OpenAI", package: "OpenAI"),
                .product(name: "ArgumentParser", package: "swift-argument-parser")
            ],
            swiftSettings: [
                .swiftLanguageVersion(.v5)
            ]
        ),
    ]
)
