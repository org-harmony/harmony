# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Dates are displayed in the format YYYY-MM-DD.

## [Unreleased]

## [0.1.0] - 2024-01-12

### Added

- Initial release of HARMONY & EIFFEL
- Default templates for PARIS (PAtterns for RequIrements Specification) by Oliver Linssen (see [README.md](README.md))
- EIFFEL ([app/eiffel](src/app/eiffel)) for requirements elicitation
- Functionality for creating, editing and deleting templates and template sets (for grouping templates)
- Functionality for OAuth 2.0 login with GitHub and Google
- Functionality for editing user information
- Main command for running the application as a web server
- Migrations command with initial migration for creating database schema
- Lots and lots of [core](src/core) functionality containing utilities
- Docker setup for production deployments
- HTMX and Bootstrap 5 for frontend
- Translations of the frontend to German and English
- Vendorized dependencies for Go backend (frontend is in public/assets)
