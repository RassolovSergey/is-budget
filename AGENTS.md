# AGENTS.md

## Purpose
This project is a Kickstarter Campaign Operating System for digital STL studios.

## Core Principles
- Separate finance and workflow systems
- Contribution-based payouts
- Full admin configurability
- Transparency and auditability

## Modules

### Finance Pipeline
Handles income → deductions → team pool

### Workflow Graph
Handles tasks, dependencies, and contribution points

### Contribution Ledger
Stores awarded points

### Campaign Lifecycle
Stages + snapshots

### Knowledge System
Tree-based documentation

## Stack
- Go backend
- Next.js frontend
- Python analytics

## Rules
- Do not mix finance and workflow logic
- All calculations must be testable
