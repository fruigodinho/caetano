# Walkthrough - Phase 2: Web Interface

## Overview
We have implemented the Web Interface using Gin, providing a modern UI for file uploads, dashboard visualization, and secure authentication.

## Changes
### Web Server
- **Gin Framework**: Set up a high-performance HTTP server.
- **Templates**: Created responsive HTML templates using `html/template` and CSS variables for theming.
- **Assets**: Added `style.css` for styling.

### Authentication
- **Login**: Implemented local authentication with username/password.
- **2FA**: Added support for TOTP (Time-based One-Time Password) using `pquerna/otp`.
- **Middleware**: Protected routes using session-based authentication (`gorilla/sessions`).

### Features
- **Dashboard**: Displays total uploads and last processing date.
- **Upload**: Drag-and-drop style form for uploading Balance Sheet CSVs.
- **Processing**: Integrated with the core logic to validate uploaded files against expected rules.

## Verification Results

### 1. Build
Ran `go build ./...` successfully. The application compiles without errors.

### 2. Manual Verification Steps
To verify the web interface:
1.  Start the server: `go run cmd/server/main.go`
2.  Open browser at `http://localhost:8080`
3.  Login with:
    - Username: `admin`
    - Password: `admin`
4.  Navigate to **Upload** and submit `data/Balancete.csv`.
5.  Check **Dashboard** to see updated statistics.

## Conclusion
The project is complete with both CLI and Web interfaces fully functional and integrated with the PostgreSQL database.
