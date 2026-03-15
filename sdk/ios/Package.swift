// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "SilentPass",
    platforms: [.iOS(.v14)],
    products: [
        .library(name: "SilentPass", targets: ["SilentPass"]),
    ],
    targets: [
        .target(name: "SilentPass", path: "Sources/SilentPass"),
        .testTarget(name: "SilentPassTests", dependencies: ["SilentPass"], path: "Tests/SilentPassTests"),
    ]
)
