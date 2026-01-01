# AICoder

[📖 User Manual](UserManual_EN.md) | [❓ FAQ](faq_en.md) | [English](README_EN.md) | [中文](README.md)

AICoder is a desktop AI programming assistant dashboard built with Wails, Go, and React. It is designed to provide unified configuration management, automated environment deployment, and one-click launch capabilities for multiple AI CLI tools (Anthropic Claude Code, OpenAI Codex, and Google Gemini CLI).

This application is deeply integrated with popular AI programming models, supporting rapid API Key configuration and automatic synchronization.
<img width="806" height="486" alt="image" src="https://github.com/user-attachments/assets/6b81570d-804d-4faa-8b79-79a84ee2fb88" />

## Core Features

*   **🚀 Automatic Environment Preparation**: Automatically detects and prepares the required AI CLI environments (Claude Code, Codex, Gemini) upon startup, supporting automatic installation and version updates.
*   **🖼️ Unified Sidebar UI**: Features a modern vertical sidebar navigation for quick switching between different AI programming tools.
*   **📂 Multi-Project Management (Vibe Coding)**:
    *   **Tabbed Interface**: Manage multiple projects simultaneously and switch contexts quickly using tabs.
    *   **Independent Configuration**: Each project can have its own working directory and launch parameters (e.g., Yolo Mode).
*   **🔄 Multi-Model & Cross-Platform Support**:
    *   Integrated with **Claude Code**, **OpenAI Codex**, and **Google Gemini CLI**.
    *   Supports **Custom** mode for connecting to compatible API endpoints.
    *   **Instant Sync**: Automatically updates configuration files and system environment variables.
*   **🌍 Multi-language Support**: Interface supports English, Simplified Chinese, Traditional Chinese, Korean, Japanese, German, and French.
*   **🖱️ System Tray Support**: Quick model switching, one-click launch, and quitting the application.
*   **⚡ One-Click Launch**: Large buttons to launch the respective CLI tool with pre-configured environments and authentication.

## Quick Start

### 1. Run the Program
Run `AICoder.exe` directly.

### 2. Environment Detection
On the first launch, the program performs an environment self-check. If required runtimes or tools are missing, AICoder will attempt to install them automatically.

### 3. Configure API Key
Enter your API Key in the configuration panel for each tool.
*   If you don't have a Key yet, click the **"Get Key"** button next to the input field to jump to the respective provider's application page.

### 4. Switch and Launch
*   Select your desired AI tool (Claude, Codex, or Gemini) in the left sidebar.
*   **Select Project**: Click a project tab in the "Vibe Coding" area to switch projects.
*   Click **"Launch"**; a terminal window with a pre-configured environment will pop up and run the tool automatically.

## About

*   **Version**: V2.0.0.2
*   **Author**: Dr. Daniel
*   **GitHub**: [RapidAI/cceasy](https://github.com/RapidAI/cceasy)
*   **Resources**: [CS146s Chinese Version](https://github.com/BIT-ENGD/cs146s_cn)

---
*This tool is intended as a configuration management aid. Please ensure you comply with the service terms of each model provider.*
