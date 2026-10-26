# droid-cli

Scaffold Kotlin Android projects from the terminal — one binary, no Android Studio required.

`droid-cli` generates a modern Android project (Compose, Kotlin DSL, version catalog) via an interactive prompt, so you can go from zero to `./gradlew assembleDebug` in seconds.

## Features

- Interactive prompts for app name, package, and template
- Kotlin + Jetpack Compose + Kotlin DSL (Gradle) + version catalog
- Standard `app/` module layout, ready for Gradle
- Single static binary — no JVM or SDK needed to *generate* a project

## Requirements

- **To build:** Go 1.22+
- **To build the generated project:** JDK 17+, Android SDK, Gradle wrapper (included)

## Installation

### Binary

Download the latest release for Linux / macOS / Windows from [Releases](https://github.com/yahyatinani/droid-cli/releases).

### Go

```bash
go install github.com/you/droid-cli@latest
```

### From source

```bash
go build -o droid-cli .
```

## Usage

```bash
droid-cli
```

Prompts for app name, package name, and template, then writes the project into the current directory.

### Example

```text
$ droid-cli
App Name: MyApp
Package Name: com.example.myapp
Template: Compose (default)

✔ Created MyApp/
  ├── app/src/main/java/com/example/myapp/MainActivity.kt
  ├── app/build.gradle.kts
  ├── build.gradle.kts
  ├── gradle/libs.versions.toml
  └── settings.gradle.kts

Next: cd MyApp && ./gradlew assembleDebug
```

## License

MIT
