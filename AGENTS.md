# AGENT GUIDELINES FOR KUCHIPUDI REPOSITORY

This document outlines the essential guidelines for agentic coding agents operating within the Kuchipudi repository. Adhering to these guidelines ensures consistency, maintainability, and efficient collaboration.

## 1. Build, Lint, and Test Commands

All agents must use the following commands for building, linting, and testing the Go codebase. For Python components, agents should be aware of the `requirements.txt` but should not assume explicit build/test commands without further analysis or user instruction.

### Go Commands

*   **Build:**
    ```bash
    go build -ldflags="-s -w" -o bin/kuchipudi ./cmd/kuchipudi
    ```
*   **Run All Tests:**
    ```bash
    go test ./... -v
    ```
*   **Run All Short Tests:**
    ```bash
    go test ./... -v -short
    ```
*   **Run a Single Test:**
    To run a specific test function (e.g., `TestGestureStore_Save` in `internal/store` package):
    ```bash
    go test ./internal/store -v -run TestGestureStore_Save
    ```
    Replace `./internal/store` with the path to the package containing the test and `TestGestureStore_Save` with the exact test function name.
*   **Lint:**
    ```bash
    golangci-lint run
    ```
*   **Format:**
    ```bash
    go fmt ./...
    ```

### Python Commands

*   The project includes Python scripts (e.g., `scripts/mediapipe_service.py`) with dependencies listed in `scripts/requirements.txt`.
*   Install dependencies: `pip install -r scripts/requirements.txt`
*   There are no explicit linting or testing commands defined in the `Makefile` for Python. Agents should not introduce new Python linting/testing frameworks without explicit user approval.

## 2. Code Style Guidelines (Go)

The Kuchipudi codebase primarily follows standard Go idioms and best practices, enforced by `go fmt` and `golangci-lint`.

### 2.1. Imports

*   Group imports into standard library, third-party, and internal/local project packages.
*   Each group should be separated by a blank line.
*   Example:
    ```go
    import (
        "log"
        "sync"

        "github.com/google/uuid"

        "github.com/ayusman/kuchipudi/internal/capture"
        "github.com/ayusman/kuchipudi/internal/detector"
    )
    ```

### 2.2. Formatting

*   Always run `go fmt ./...` to ensure consistent code formatting.
*   Adhere to `gofumpt` style, which `go fmt` typically enforces.

### 2.3. Naming Conventions

*   **Packages:** `camelCase` (e.g., `app`, `store`, `api`, `gesture`, `plugin`).
*   **Structs, Interfaces, Enums (custom types):** `PascalCase` (e.g., `App`, `Config`, `GestureType`, `Detector`).
*   **Exported Functions/Methods, Global Variables, Constants:** `PascalCase` (e.g., `New`, `SetEnabled`, `ErrNotFound`, `IdleFPS`).
*   **Unexported Functions/Methods, Local Variables, Struct Fields:** `camelCase` (e.g., `runPipeline`, `config`, `camera`).
*   **Constants representing types/values:** `PascalCase` (e.g., `GestureTypeStatic`) or `UPPER_SNAKE_CASE` where appropriate (though less common in this codebase).

### 2.4. Types

*   Use structs for composite data types.
*   Enums are typically defined using `const` declarations with a custom string type.
*   Clearly define struct fields and their purposes.

### 2.5. Error Handling

*   Errors are always the last return value of a function.
*   Explicit error checking is mandatory: `if err != nil { return err }`.
*   Use `errors.Is` to check for specific error types (e.g., `errors.Is(err, store.ErrNotFound)`).
*   Use `fmt.Errorf("descriptive message: %w", err)` to wrap errors and add context.
*   Log errors using `log.Printf` or similar mechanisms when appropriate, especially at points where an error should not propagate further up the call stack.

### 2.6. Comments

*   Use package-level comments to describe the overall purpose of a package.
*   Document all exported types, functions, and methods with clear, concise comments explaining their purpose, parameters, and return values.
*   Add comments for complex logic or non-obvious design decisions.
*   Avoid redundant comments that merely re-state what the code does.

### 2.7. Structure and Organization

*   Organize code into logical packages based on functionality (e.g., `internal/app` for main application logic, `internal/store` for database interactions, `internal/server/api` for HTTP API handlers).
*   Follow the constructor pattern for creating new instances of types (e.g., `NewApp`, `NewManager`).
*   Group methods by their receiver type.

## 3. Cursor/Copilot Rules

There are no `.cursor/rules/` or `.github/copilot-instructions.md` files found in this repository. Agents should adhere to the general Go style guidelines outlined above.